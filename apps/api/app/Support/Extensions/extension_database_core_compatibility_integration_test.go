package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	"github.com/zhuchunshu/sforum/apps/api/database/coreauthority"
)

func TestPostgresExtensionDatabaseHighRiskLeaseObservesMigratedCoreFence(t *testing.T) {
	ctx, pool := openExtensionDatabaseRawCoreTestPool(t)
	var originalCurrent, originalTarget, originalStatus string
	if err := pool.QueryRow(ctx, `
		SELECT current_version, target_version, status
		FROM public.sforum_core_runtime_state
		WHERE singleton = TRUE
	`).Scan(&originalCurrent, &originalTarget, &originalStatus); err != nil {
		t.Fatalf("load durable Core runtime state: %v", err)
	}
	if originalCurrent != "1.0.0" || originalTarget != originalCurrent || originalStatus != "ready" {
		t.Fatalf(
			"high-risk lease test requires ready Core 1.0.0, got current=%q target=%q status=%q",
			originalCurrent, originalTarget, originalStatus,
		)
	}
	restoreCoreRuntimeState := func() {
		restoreCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		connection, err := pool.Acquire(restoreCtx)
		if err != nil {
			t.Errorf("acquire Core state restore connection: %v", err)
			return
		}
		defer connection.Release()
		if err := lockExtensionDatabasePhysicalAuthoritySession(restoreCtx, connection); err != nil {
			t.Errorf("lock Core state restore: %v", err)
			return
		}
		defer unlockExtensionDatabasePhysicalAuthoritySession(restoreCtx, connection)
		if _, err := connection.Exec(restoreCtx, `
			UPDATE public.sforum_core_runtime_state
			SET current_version = $1, target_version = $2, status = $3,
			    migrated_at = CASE WHEN $3 = 'ready' THEN statement_timestamp() ELSE NULL END,
			    revision = revision + 1, updated_at = statement_timestamp()
			WHERE singleton = TRUE
		`, originalCurrent, originalTarget, originalStatus); err != nil {
			t.Errorf("restore durable Core runtime state: %v", err)
		}
	}
	t.Cleanup(restoreCoreRuntimeState)

	extensionID := fmt.Sprintf("p5.core-fence.%d", time.Now().UnixNano())
	artifact := insertExtensionDatabaseRuntimeLeaseFixture(
		t, ctx, pool, extensionID, "1.0.0", "core-fence",
		[]string{extensionmanifest.DatabaseGrantRawCore},
	)
	identifiers, err := ExtensionDatabaseIdentifiersFor(extensionID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanupExtensionDatabaseRuntimeLeaseFixture(t, pool, extensionID, identifiers) })
	registry := NewPostgresExtensionDatabaseRegistry(pool, nil)
	credential, err := registry.IssueRuntimeLease(ctx, ExtensionDatabaseRuntimeLeaseIssue{
		Artifact: artifact, RuntimeInstanceID: "source-runtime",
		Authority: ExtensionDatabaseLeaseAuthority{
			Kind: ExtensionDatabaseLeaseIssuerActor, ActorUserID: 1101, AuditEventID: 1102,
		},
	})
	if err != nil {
		t.Fatalf("issue compatible high-risk lease: %v", err)
	}

	lockConnection, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer lockConnection.Release()
	if err := lockExtensionDatabasePhysicalAuthoritySession(ctx, lockConnection); err != nil {
		t.Fatal(err)
	}
	locked := true
	defer func() {
		if locked {
			_ = unlockExtensionDatabasePhysicalAuthoritySession(context.Background(), lockConnection)
		}
	}()
	if _, err := lockConnection.Exec(ctx, `
		UPDATE public.sforum_core_runtime_state
		SET target_version = '2.0.0', status = 'migrating', migrated_at = NULL,
		    revision = revision + 1, updated_at = statement_timestamp()
		WHERE singleton = TRUE
	`); err != nil {
		t.Fatal(err)
	}

	waiterConfig := pool.Config().Copy()
	applicationName := "sforum-core-version-heartbeat-waiter"
	waiterConfig.ConnConfig.RuntimeParams["application_name"] = applicationName
	waiterPool, err := pgxpool.NewWithConfig(ctx, waiterConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer waiterPool.Close()
	waiterRegistry := NewPostgresExtensionDatabaseRegistry(waiterPool, nil)
	heartbeatResult := make(chan error, 1)
	go func() {
		_, err := waiterRegistry.HeartbeatRuntimeLease(
			ctx, extensionDatabaseRuntimeCredentialRef(credential), credential.Revision,
		)
		heartbeatResult <- err
	}()
	waitForExtensionDatabasePhysicalAuthorityWait(t, ctx, pool, applicationName, heartbeatResult)

	if _, err := lockConnection.Exec(ctx, `
		UPDATE public.sforum_core_runtime_state
		SET current_version = '2.0.0', target_version = '2.0.0', status = 'ready',
		    migrated_at = statement_timestamp(), revision = revision + 1,
		    updated_at = statement_timestamp()
		WHERE singleton = TRUE
	`); err != nil {
		t.Fatal(err)
	}
	if err := unlockExtensionDatabasePhysicalAuthoritySession(ctx, lockConnection); err != nil {
		t.Fatal(err)
	}
	locked = false
	if err := <-heartbeatResult; !errors.Is(err, ErrExtensionDatabaseCoreIncompatible) {
		t.Fatalf("heartbeat after incompatible Core publication error = %v", err)
	}
	if _, err := registry.IssueRuntimeLease(ctx, ExtensionDatabaseRuntimeLeaseIssue{
		Artifact: artifact, RuntimeInstanceID: "new-runtime",
		Authority: ExtensionDatabaseLeaseAuthority{
			Kind: ExtensionDatabaseLeaseIssuerActor, ActorUserID: 1103, AuditEventID: 1104,
		},
	}); !errors.Is(err, ErrExtensionDatabaseCoreIncompatible) {
		t.Fatalf("new lease after incompatible Core publication error = %v", err)
	}
}

func lockExtensionDatabasePhysicalAuthoritySession(ctx context.Context, connection *pgxpool.Conn) error {
	_, err := connection.Exec(ctx, `
		SELECT pg_advisory_lock(
			hashtext(current_database()),
			hashtext($1)
		)
	`, coreauthority.PhysicalAuthorityLockName)
	return err
}

func unlockExtensionDatabasePhysicalAuthoritySession(ctx context.Context, connection *pgxpool.Conn) error {
	var unlocked bool
	if err := connection.QueryRow(ctx, `
		SELECT pg_advisory_unlock(
			hashtext(current_database()),
			hashtext($1)
		)
	`, coreauthority.PhysicalAuthorityLockName).Scan(&unlocked); err != nil {
		return err
	}
	if !unlocked {
		return errors.New("physical authority session lock was not held")
	}
	return nil
}

func waitForExtensionDatabasePhysicalAuthorityWait(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	applicationName string,
	result <-chan error,
) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		var waiting bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
			  SELECT 1 FROM pg_stat_activity
			  WHERE application_name = $1
			    AND position('pg_advisory_xact_lock' in query) > 0
			)
		`, applicationName).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		select {
		case err := <-result:
			t.Fatalf("high-risk lease returned before physical fence: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("high-risk lease did not reach physical authority fence")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestPostgresLifecycleBoundaryStatePublishesAndRestoresEveryOperation(t *testing.T) {
	tests := []struct {
		operation extensions.LifecycleMachineOperation
		position  int
		mode      LifecycleBoundaryPublicationMode
	}{
		{extensions.LifecycleMachineInstall, 8, LifecycleBoundaryActivate},
		{extensions.LifecycleMachineEnable, 5, LifecycleBoundaryActivate},
		{extensions.LifecycleMachineDisable, 3, LifecycleBoundaryDeactivate},
		{extensions.LifecycleMachineUpgrade, 8, LifecycleBoundaryActivate},
		{extensions.LifecycleMachineRollback, 6, LifecycleBoundaryActivate},
		{extensions.LifecycleMachineUninstall, 3, LifecycleBoundaryDeactivate},
	}
	for _, test := range tests {
		t.Run(string(test.operation), func(t *testing.T) {
			fixture := newLifecycleStatePublicationIntegration(t, test.operation, test.position, test.mode)
			transaction, err := fixture.state.PrepareLifecycleStatePublication(fixture.ctx, fixture.request, fixture.mode)
			if err != nil {
				t.Fatal(err)
			}
			assertLifecycleStateTransaction(t, fixture.ctx, transaction, LifecycleBoundaryTransactionSource)
			assertLifecycleStateExtension(t, fixture, fixture.sourceState)

			if err := transaction.Publish(fixture.ctx); err != nil {
				t.Fatal(err)
			}
			if err := transaction.Publish(fixture.ctx); err != nil {
				t.Fatalf("idempotent publish: %v", err)
			}
			assertLifecycleStateTransaction(t, fixture.ctx, transaction, LifecycleBoundaryTransactionTarget)
			assertLifecycleStateExtension(t, fixture, fixture.targetState)

			if err := transaction.Restore(fixture.ctx); err != nil {
				t.Fatal(err)
			}
			if err := transaction.Restore(fixture.ctx); err != nil {
				t.Fatalf("idempotent restore: %v", err)
			}
			assertLifecycleStateTransaction(t, fixture.ctx, transaction, LifecycleBoundaryTransactionSource)
			assertLifecycleStateExtension(t, fixture, fixture.sourceState)
		})
	}
}

func TestPostgresLifecycleBoundaryStateRecoversAfterProcessRestart(t *testing.T) {
	fixture := newLifecycleStatePublicationIntegration(
		t, extensions.LifecycleMachineUpgrade, 8, LifecycleBoundaryActivate,
	)
	transaction, err := fixture.state.PrepareLifecycleStatePublication(fixture.ctx, fixture.request, fixture.mode)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.Publish(fixture.ctx); err != nil {
		t.Fatal(err)
	}
	assertLifecycleStateExtension(t, fixture, fixture.targetState)

	// 丢弃原 adapter/transaction，模拟 Host 进程退出。新的对象只用 operation/step
	// 重建 source snapshot，不持有前一进程的补偿 closure。
	restartedStore := extensions.NewPostgresStore(fixture.pool)
	restartedState := NewPostgresLifecycleBoundaryState(restartedStore)
	restarted, err := restartedState.PrepareLifecycleStatePublication(fixture.ctx, fixture.request, fixture.mode)
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycleStateTransaction(t, fixture.ctx, restarted, LifecycleBoundaryTransactionTarget)
	if err := restarted.Restore(fixture.ctx); err != nil {
		t.Fatal(err)
	}
	assertLifecycleStateExtension(t, fixture, fixture.sourceState)
}

func TestPostgresLifecycleBoundaryStateTracksNoopDisabledUninstall(t *testing.T) {
	fixture := newLifecycleStatePublicationIntegration(
		t, extensions.LifecycleMachineUninstall, 3, LifecycleBoundaryDeactivate,
	)
	disabled, err := fixture.store.Disable(fixture.ctx, fixture.extensionID)
	if err != nil {
		t.Fatal(err)
	}
	fixture.sourceState = lifecycleStateVectorFromExtension(disabled)
	fixture.targetState = fixture.sourceState
	transaction, err := fixture.state.PrepareLifecycleStatePublication(fixture.ctx, fixture.request, fixture.mode)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.Publish(fixture.ctx); err != nil {
		t.Fatal(err)
	}
	// 即使物理 row 已经是 disabled，durable phase 仍必须记录 target，避免重启
	// 后把“未发布”和“已发布但无物理差异”混为一谈。
	assertLifecycleStateTransaction(t, fixture.ctx, transaction, LifecycleBoundaryTransactionTarget)
	if err := transaction.Restore(fixture.ctx); err != nil {
		t.Fatal(err)
	}
	assertLifecycleStateTransaction(t, fixture.ctx, transaction, LifecycleBoundaryTransactionSource)
	assertLifecycleStateExtension(t, fixture, fixture.sourceState)
}

func TestPostgresLifecycleBoundaryStateFencesConcurrentAndStaleAttempts(t *testing.T) {
	fixture := newLifecycleStatePublicationIntegration(
		t, extensions.LifecycleMachineEnable, 5, LifecycleBoundaryActivate,
	)
	first, err := fixture.state.PrepareLifecycleStatePublication(fixture.ctx, fixture.request, fixture.mode)
	if err != nil {
		t.Fatal(err)
	}
	second, err := fixture.state.PrepareLifecycleStatePublication(fixture.ctx, fixture.request, fixture.mode)
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wait sync.WaitGroup
	for _, transaction := range []LifecycleBoundaryTransaction{first, second} {
		wait.Add(1)
		go func(transaction LifecycleBoundaryTransaction) {
			defer wait.Done()
			<-start
			errs <- transaction.Publish(fixture.ctx)
		}(transaction)
	}
	close(start)
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("same exact publication must be idempotent: %v", err)
		}
	}
	assertLifecycleStateExtension(t, fixture, fixture.targetState)

	nextRequest := fixture.request
	nextRequest.Attempt++
	if err := fixture.journal.PrepareLifecyclePublication(fixture.ctx, nextRequest, fixture.mode); err != nil {
		t.Fatal(err)
	}
	next, err := fixture.state.PrepareLifecycleStatePublication(fixture.ctx, nextRequest, fixture.mode)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Restore(fixture.ctx); !errors.Is(err, extensions.ErrLifecycleStatePublicationConflict) {
		t.Fatalf("stale attempt restore error = %v", err)
	}
	if err := next.Restore(fixture.ctx); err != nil {
		t.Fatal(err)
	}
	assertLifecycleStateExtension(t, fixture, fixture.sourceState)
}

func TestPostgresLifecycleBoundaryStateUsesOperationFirstLockOrder(t *testing.T) {
	fixture := newLifecycleStatePublicationIntegration(
		t, extensions.LifecycleMachineUpgrade, 8, LifecycleBoundaryActivate,
	)
	ctx, cancel := context.WithTimeout(fixture.ctx, 10*time.Second)
	defer cancel()

	const workers = 24
	start := make(chan struct{})
	errs := make(chan error, workers*2)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(2)
		go func() {
			defer wait.Done()
			<-start
			errs <- fixture.journal.PrepareLifecyclePublication(ctx, fixture.request, fixture.mode)
		}()
		go func() {
			defer wait.Done()
			<-start
			_, err := fixture.state.PrepareLifecycleStatePublication(ctx, fixture.request, fixture.mode)
			errs <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("operation-first concurrent prepare: %v", err)
		}
	}
}

func TestPostgresLifecycleBoundaryStateCommitUnknownCannotRestoreSource(t *testing.T) {
	fixture := newLifecycleStatePublicationIntegration(
		t, extensions.LifecycleMachineEnable, 5, LifecycleBoundaryActivate,
	)
	transaction, err := fixture.state.PrepareLifecycleStatePublication(fixture.ctx, fixture.request, fixture.mode)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.Publish(fixture.ctx); err != nil {
		t.Fatal(err)
	}
	// 模拟 marker 已在服务端提交但调用者未收到确认。重启后的状态 adapter 只能
	// 向 target 收敛，绝不能根据旧调用错误反补偿 source。
	if err := fixture.journal.CommitLifecyclePublication(fixture.ctx, fixture.request, fixture.mode); err != nil {
		t.Fatal(err)
	}
	restarted := NewPostgresLifecycleBoundaryState(extensions.NewPostgresStore(fixture.pool))
	recovered, err := restarted.PrepareLifecycleStatePublication(fixture.ctx, fixture.request, fixture.mode)
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycleStateTransaction(t, fixture.ctx, recovered, LifecycleBoundaryTransactionTarget)
	if err := recovered.Restore(fixture.ctx); !errors.Is(err, extensions.ErrLifecycleStatePublicationCommitted) {
		t.Fatalf("postcommit restore error = %v", err)
	}
	assertLifecycleStateExtension(t, fixture, fixture.targetState)
}

func TestPostgresLifecycleBoundaryStateRejectsPhysicalDrift(t *testing.T) {
	fixture := newLifecycleStatePublicationIntegration(
		t, extensions.LifecycleMachineEnable, 5, LifecycleBoundaryActivate,
	)
	transaction, err := fixture.state.PrepareLifecycleStatePublication(fixture.ctx, fixture.request, fixture.mode)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE extensions SET status = 'enabled' WHERE id = $1
	`, fixture.extensionID); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Publish(fixture.ctx); !errors.Is(err, extensions.ErrLifecycleStatePublicationConflict) {
		t.Fatalf("physical drift error = %v", err)
	}
}

func TestPostgresLifecycleBoundaryStateRollsBackAtomicWriteFailure(t *testing.T) {
	fixture := newLifecycleStatePublicationIntegration(
		t, extensions.LifecycleMachineEnable, 5, LifecycleBoundaryActivate,
	)
	transaction, err := fixture.state.PrepareLifecycleStatePublication(fixture.ctx, fixture.request, fixture.mode)
	if err != nil {
		t.Fatal(err)
	}
	functionName := fmt.Sprintf("test_lifecycle_state_fail_%d", time.Now().UnixNano())
	triggerName := functionName + "_trigger"
	quotedFunction := pgx.Identifier{functionName}.Sanitize()
	quotedTrigger := pgx.Identifier{triggerName}.Sanitize()
	if _, err := fixture.pool.Exec(fixture.ctx, fmt.Sprintf(`
		CREATE FUNCTION %s() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
		  RAISE EXCEPTION 'forced lifecycle state publication failure';
		END
		$$
	`, quotedFunction)); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, fmt.Sprintf(`
		CREATE TRIGGER %s
		BEFORE UPDATE OF status, active_version_id, staged_version_id ON extensions
		FOR EACH ROW WHEN (OLD.id = '%s')
		EXECUTE FUNCTION %s()
	`, quotedTrigger, strings.ReplaceAll(fixture.extensionID, "'", "''"), quotedFunction)); err != nil {
		_, _ = fixture.pool.Exec(fixture.ctx, fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, quotedFunction))
		t.Fatal(err)
	}
	dropFailure := func() {
		_, _ = fixture.pool.Exec(context.Background(), fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON extensions`, quotedTrigger))
		_, _ = fixture.pool.Exec(context.Background(), fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, quotedFunction))
	}
	t.Cleanup(dropFailure)

	if err := transaction.Publish(fixture.ctx); err == nil {
		t.Fatal("forced state publication failure unexpectedly succeeded")
	}
	assertLifecycleStateTransaction(t, fixture.ctx, transaction, LifecycleBoundaryTransactionSource)
	assertLifecycleStateExtension(t, fixture, fixture.sourceState)
	dropFailure()
	if err := transaction.Publish(fixture.ctx); err != nil {
		t.Fatal(err)
	}
	assertLifecycleStateTransaction(t, fixture.ctx, transaction, LifecycleBoundaryTransactionTarget)
	assertLifecycleStateExtension(t, fixture, fixture.targetState)
}

type lifecycleStateIntegrationVector struct {
	status            string
	activeID          int64
	stagedID          int64
	version           string
	packageDigest     string
	packagePath       string
	adminDigest       string
	manifestVersion   string
	lifecycleContract string
}

type lifecycleStatePublicationIntegration struct {
	ctx         context.Context
	pool        *pgxpool.Pool
	store       *extensions.PostgresStore
	state       *PostgresLifecycleBoundaryState
	journal     *PostgresLifecycleBoundaryPublicationJournal
	extensionID string
	request     LifecycleBoundaryRequest
	mode        LifecycleBoundaryPublicationMode
	sourceState lifecycleStateIntegrationVector
	targetState lifecycleStateIntegrationVector
}

func newLifecycleStatePublicationIntegration(
	t *testing.T,
	operation extensions.LifecycleMachineOperation,
	position int,
	mode LifecycleBoundaryPublicationMode,
) lifecycleStatePublicationIntegration {
	t.Helper()
	databaseURL := strings.TrimSpace(os.Getenv("SFORUM_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("SFORUM_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	store := extensions.NewPostgresStore(pool)
	extensionID := fmt.Sprintf("state.publication.%s.%d", operation, time.Now().UnixNano())
	root := t.TempDir()
	v1 := exactCoordinatorTestExtension(extensionID, "1.0.0", strings.Repeat("a", 64), "state.lifecycle@1", 1)
	installed, err := store.SaveInstalled(ctx, extensions.SaveInstalledInput{
		Manifest: v1.Manifest, PackagePath: filepath.Join(root, "v1"), PackageDigest: v1.PackageDigest,
		AdminFrontendDigest: strings.Repeat("1", 64),
	})
	if err != nil {
		pool.Close()
		t.Fatal(err)
	}
	var source *extensions.Extension
	target := installed
	switch operation {
	case extensions.LifecycleMachineInstall:
		v2 := exactCoordinatorTestExtension(extensionID, "2.0.0", strings.Repeat("b", 64), "state.lifecycle@2", 2)
		staged, stageErr := store.SaveInstalled(ctx, extensions.SaveInstalledInput{
			Manifest: v2.Manifest, PackagePath: filepath.Join(root, "v2"), PackageDigest: v2.PackageDigest,
			AdminFrontendDigest: strings.Repeat("2", 64),
		})
		if stageErr != nil || staged.StagedVersion == nil {
			t.Fatalf("stage inert first-enable candidate = %#v, %v", staged.StagedVersion, stageErr)
		}
		target = lifecycleStateExtensionVersion(staged, *staged.StagedVersion)
	case extensions.LifecycleMachineEnable:
	case extensions.LifecycleMachineDisable, extensions.LifecycleMachineUninstall:
		enabled, enableErr := store.Enable(ctx, extensionID, extensions.TypePlugin)
		if enableErr != nil {
			t.Fatal(enableErr)
		}
		target = enabled
		source = cloneLifecycleStateExtension(enabled)
	case extensions.LifecycleMachineUpgrade:
		enabled, enableErr := store.Enable(ctx, extensionID, extensions.TypePlugin)
		if enableErr != nil {
			t.Fatal(enableErr)
		}
		v2 := exactCoordinatorTestExtension(extensionID, "2.0.0", strings.Repeat("b", 64), "state.lifecycle@2", 2)
		staged, stageErr := store.SaveInstalled(ctx, extensions.SaveInstalledInput{
			Manifest: v2.Manifest, PackagePath: filepath.Join(root, "v2"), PackageDigest: v2.PackageDigest,
			AdminFrontendDigest: strings.Repeat("2", 64),
		})
		if stageErr != nil || staged.StagedVersion == nil {
			t.Fatalf("stage upgrade = %#v, %v", staged.StagedVersion, stageErr)
		}
		source = cloneLifecycleStateExtension(enabled)
		target = lifecycleStateExtensionVersion(staged, *staged.StagedVersion)
	case extensions.LifecycleMachineRollback:
		if _, enableErr := store.Enable(ctx, extensionID, extensions.TypePlugin); enableErr != nil {
			t.Fatal(enableErr)
		}
		v2 := exactCoordinatorTestExtension(extensionID, "2.0.0", strings.Repeat("b", 64), "state.lifecycle@2", 2)
		staged, stageErr := store.SaveInstalled(ctx, extensions.SaveInstalledInput{
			Manifest: v2.Manifest, PackagePath: filepath.Join(root, "v2"), PackageDigest: v2.PackageDigest,
			AdminFrontendDigest: strings.Repeat("2", 64),
		})
		if stageErr != nil || staged.StagedVersion == nil {
			t.Fatalf("stage rollback source = %#v, %v", staged.StagedVersion, stageErr)
		}
		promoted, promoteErr := store.PromoteStagedVersion(ctx, extensions.StagedVersionCASInput{
			ExtensionID:             extensionID,
			ExpectedActiveVersionID: staged.ActiveVersionID, ExpectedActiveVersion: staged.Version,
			ExpectedActivePackageDigest: staged.PackageDigest,
			ExpectedStagedVersionID:     staged.StagedVersion.ID, ExpectedStagedVersion: staged.StagedVersion.Version,
			ExpectedPackageDigest: staged.StagedVersion.PackageDigest,
		})
		if promoteErr != nil {
			t.Fatal(promoteErr)
		}
		v3 := exactCoordinatorTestExtension(extensionID, "3.0.0", strings.Repeat("d", 64), "state.lifecycle@3", 3)
		withCandidate, candidateErr := store.SaveInstalled(ctx, extensions.SaveInstalledInput{
			Manifest: v3.Manifest, PackagePath: filepath.Join(root, "v3"), PackageDigest: v3.PackageDigest,
			AdminFrontendDigest: strings.Repeat("3", 64),
		})
		if candidateErr != nil || withCandidate.StagedVersion == nil {
			t.Fatalf("stage rollback-preserved candidate = %#v, %v", withCandidate.StagedVersion, candidateErr)
		}
		v1Snapshot, versionErr := store.GetExtensionVersion(ctx, extensions.ExactExtensionVersionInput{
			ExtensionID: extensionID, Version: installed.Version, PackageDigest: installed.PackageDigest,
		})
		if versionErr != nil {
			t.Fatal(versionErr)
		}
		if withCandidate.ActiveVersionID != promoted.ActiveVersionID {
			t.Fatalf("staging rollback candidate changed active version: %#v", withCandidate)
		}
		source = cloneLifecycleStateExtension(withCandidate)
		target = lifecycleStateExtensionVersion(withCandidate, v1Snapshot)
	default:
		t.Fatalf("unsupported fixture operation %q", operation)
	}

	request := lifecycleStateBoundaryRequest(operation, position, mode, source, target)
	var removalMode any
	if operation == extensions.LifecycleMachineUninstall {
		removalMode = extensions.LifecycleRemovalPreserve
	}
	err = pool.QueryRow(ctx, `
		INSERT INTO extension_lifecycle_operations (
			extension_id, extension_version, package_digest, artifact_digests,
			operation, plan_version, idempotency_key, request_fingerprint,
			authority_type, authority_snapshot, removal_mode
		) VALUES ($1, $2, $3, '{}'::jsonb, $4, $5, $6, $7, 'builtin', '{}'::jsonb, $8)
		RETURNING id
	`, extensionID, target.Version, target.PackageDigest, operation,
		target.Manifest.Lifecycle.ContractVersion, "state-publication:"+extensionID,
		strings.Repeat("c", 64), removalMode).Scan(&request.OperationID)
	if err != nil {
		pool.Close()
		t.Fatal(err)
	}
	journal := NewPostgresLifecycleBoundaryPublicationJournal(pool)
	if err := journal.PrepareLifecyclePublication(ctx, request, mode); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM extension_lifecycle_operations WHERE id = $1`, request.OperationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM extensions WHERE id = $1`, extensionID)
		pool.Close()
	})
	current, err := store.Get(ctx, extensionID)
	if err != nil {
		t.Fatal(err)
	}
	sourceState := lifecycleStateVectorFromExtension(current)
	targetState := lifecycleStateVectorFromExtension(target)
	targetState.status = extensions.StatusEnabled
	targetState.stagedID = sourceState.stagedID
	switch operation {
	case extensions.LifecycleMachineDisable, extensions.LifecycleMachineUninstall:
		targetState.status = extensions.StatusDisabled
	case extensions.LifecycleMachineInstall, extensions.LifecycleMachineUpgrade:
		targetState.stagedID = 0
	}
	return lifecycleStatePublicationIntegration{
		ctx: ctx, pool: pool, store: store,
		state: NewPostgresLifecycleBoundaryState(store), journal: journal,
		extensionID: extensionID, request: request, mode: mode,
		sourceState: sourceState, targetState: targetState,
	}
}

func lifecycleStateBoundaryRequest(
	operation extensions.LifecycleMachineOperation,
	position int,
	mode LifecycleBoundaryPublicationMode,
	source *extensions.Extension,
	target extensions.Extension,
) LifecycleBoundaryRequest {
	path, _ := extensions.RecommendedLifecyclePath(operation)
	request := LifecycleBoundaryRequest{
		Operation: operation, Position: position,
		StepID:  fmt.Sprintf("lifecycle.%s.%02d.host.%s", operation, position, path[position].State),
		Attempt: 1, TargetExtension: target,
		TargetBinding: extensions.LifecycleRuntimeBinding{
			ExtensionID: target.ID, ExtensionVersion: target.Version,
			PackageDigest: target.PackageDigest, VersionID: target.ActiveVersionID,
		},
	}
	if mode == LifecycleBoundaryActivate {
		request.TargetBinding.RuntimeInstanceID = "target-runtime"
	}
	if source != nil {
		request.SourceExtension = cloneLifecycleStateExtension(*source)
		request.SourceBinding = extensions.LifecycleRuntimeBinding{
			ExtensionID: source.ID, ExtensionVersion: source.Version,
			PackageDigest: source.PackageDigest, VersionID: source.ActiveVersionID,
			RuntimeInstanceID: "source-runtime",
		}
	}
	if operation == extensions.LifecycleMachineUninstall {
		request.RemovalMode = extensions.LifecycleRemovalPreserve
	}
	return request
}

func lifecycleStateExtensionVersion(base extensions.Extension, version extensions.ExtensionVersion) extensions.Extension {
	base.Version, base.Manifest = version.Version, version.Manifest
	base.PackageDigest, base.AdminFrontendDigest = version.PackageDigest, version.AdminFrontendDigest
	base.PackagePath, base.ActiveVersionID = version.PackagePath, version.ID
	base.InstalledAt, base.StagedVersion = version.InstalledAt, nil
	return base
}

func cloneLifecycleStateExtension(extension extensions.Extension) *extensions.Extension {
	clone := extension
	return &clone
}

func lifecycleStateVectorFromExtension(extension extensions.Extension) lifecycleStateIntegrationVector {
	state := lifecycleStateIntegrationVector{
		status: extension.Status, activeID: extension.ActiveVersionID,
		version: extension.Version, packageDigest: extension.PackageDigest,
		packagePath: extension.PackagePath, adminDigest: extension.AdminFrontendDigest,
		manifestVersion: extension.Manifest.Version,
	}
	if extension.Manifest.Lifecycle != nil {
		state.lifecycleContract = extension.Manifest.Lifecycle.ContractVersion
	}
	if extension.StagedVersion != nil {
		state.stagedID = extension.StagedVersion.ID
	}
	return state
}

func assertLifecycleStateTransaction(
	t *testing.T,
	ctx context.Context,
	transaction LifecycleBoundaryTransaction,
	want LifecycleBoundaryTransactionState,
) {
	t.Helper()
	got, err := transaction.Inspect(ctx)
	if err != nil || got != want {
		t.Fatalf("transaction state = %q, %v; want %q", got, err, want)
	}
}

func assertLifecycleStateExtension(
	t *testing.T,
	fixture lifecycleStatePublicationIntegration,
	want lifecycleStateIntegrationVector,
) {
	t.Helper()
	current, err := fixture.store.Get(fixture.ctx, fixture.extensionID)
	if err != nil {
		t.Fatal(err)
	}
	got := lifecycleStateVectorFromExtension(current)
	if got != want {
		t.Fatalf("extension state = %#v; want %#v", got, want)
	}
}

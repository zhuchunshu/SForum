package extensions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresExecutableTrustStoreConsumesChallengeOnce(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("SFORUM_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("SFORUM_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	extensionID := "trust.integration." + suffix
	var actorID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (username, username_lower, email, email_lower, display_name)
		VALUES ($1, $1, $2, $2, 'Trust Integration') RETURNING id
	`, "trust_"+suffix, "trust_"+suffix+"@example.test").Scan(&actorID); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, actorID)
	if _, err := pool.Exec(ctx, `INSERT INTO extensions (id, type, name) VALUES ($1, 'plugin', 'Trust Integration')`, extensionID); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM extensions WHERE id = $1`, extensionID)

	identity := TrustIdentity{
		ExtensionID: extensionID, ExtensionVersion: "1.0.0",
		PackageDigest: strings.Repeat("a", 64), Action: TrustActionEnable,
		ImpactDigest: strings.Repeat("b", 64),
	}
	store := NewPostgresExecutableTrustStore(pool)
	if err := store.CreateChallenge(ctx, TrustChallengeRecord{
		TokenHash: strings.Repeat("c", 64), ActorUserID: actorID, Identity: identity,
		ArtifactDigests: map[string]string{"package": identity.PackageDigest},
		Impact: TrustImpact{
			SchemaVersion: TrustImpactSchemaV1, Action: TrustActionEnable,
			ExtensionID: extensionID, ExtensionVersion: "1.0.0",
			PackageDigest: identity.PackageDigest, ArtifactDigests: map[string]string{"package": identity.PackageDigest},
			Digest: identity.ImpactDigest,
		},
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}

	input := TrustConsumeInput{TokenHash: strings.Repeat("c", 64), ActorUserID: actorID, Identity: identity}
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := store.ConsumeChallenge(ctx, input)
			results <- err
		}()
	}
	wait.Wait()
	close(results)
	succeeded := 0
	replayed := 0
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrTrustChallengeReplayed):
			replayed++
		default:
			t.Fatalf("unexpected consume result: %v", err)
		}
	}
	if succeeded != 1 || replayed != 1 {
		t.Fatalf("consume results succeeded=%d replayed=%d", succeeded, replayed)
	}
	granted, err := store.HasLiveGrant(ctx, identity)
	if err != nil || !granted {
		t.Fatalf("live grant=%v err=%v", granted, err)
	}
	if err := store.RevokeAll(ctx, extensionID, actorID, "integration_test"); err != nil {
		t.Fatal(err)
	}
	granted, err = store.HasLiveGrant(ctx, identity)
	if err != nil || granted {
		t.Fatalf("revoked grant=%v err=%v", granted, err)
	}
}

func TestPostgresActivationAttemptsDriveBootLoopSkip(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("SFORUM_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("SFORUM_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	extension := Extension{
		ID:   "activation.integration." + fmt.Sprintf("%d", time.Now().UnixNano()),
		Name: "Activation Integration", Version: "1.0.0", Type: TypePlugin,
		Status: StatusEnabled, Source: SourceUploaded, PackageDigest: strings.Repeat("d", 64),
		Manifest: Manifest{Backend: ManifestBackend{Entry: "backend/plugin"}},
	}
	if _, err := pool.Exec(ctx, `INSERT INTO extensions (id, type, name, status) VALUES ($1, 'plugin', $2, 'enabled')`, extension.ID, extension.Name); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM extensions WHERE id = $1`, extension.ID)
	store := NewPostgresStore(pool)
	attempt, err := store.BeginActivationAttempt(ctx, extension, ActivationTriggerStartup, "boot-1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteActivationAttempt(ctx, attempt.ID, ActivationStatusFailed, "startup_crash"); err != nil {
		t.Fatal(err)
	}
	coordinator := NewActivationCoordinator(store)
	skip, err := coordinator.ShouldSkipStartup(ctx, extension, "boot-2")
	if err != nil || !skip {
		t.Fatalf("skip=%v err=%v", skip, err)
	}
	latest, err := store.LatestActivationAttempt(ctx, extension.ID, extension.PackageDigest)
	if err != nil || latest.Status != ActivationStatusSkipped || latest.FailureReason != "boot_loop_guard" {
		t.Fatalf("latest=%#v err=%v", latest, err)
	}
}

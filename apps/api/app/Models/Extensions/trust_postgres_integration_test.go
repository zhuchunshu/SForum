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
	type consumeResult struct {
		grant TrustGrant
		err   error
	}
	results := make(chan consumeResult, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			grant, err := store.ConsumeChallenge(ctx, input)
			results <- consumeResult{grant: grant, err: err}
		}()
	}
	wait.Wait()
	close(results)
	succeeded := 0
	replayed := 0
	var createdGrant TrustGrant
	for result := range results {
		switch {
		case result.err == nil:
			succeeded++
			createdGrant = result.grant
		case errors.Is(result.err, ErrTrustChallengeReplayed):
			replayed++
		default:
			t.Fatalf("unexpected consume result: %v", result.err)
		}
	}
	if succeeded != 1 || replayed != 1 || !createdGrant.created {
		t.Fatalf("consume results succeeded=%d replayed=%d grant=%#v", succeeded, replayed, createdGrant)
	}
	granted, err := store.HasLiveGrant(ctx, identity)
	if err != nil || !granted {
		t.Fatalf("live grant=%v err=%v", granted, err)
	}
	liveGrant, err := store.LiveGrant(ctx, identity)
	if err != nil || liveGrant.ID <= 0 || liveGrant.RevokedAt != nil || liveGrant.RevokedByUserID != 0 {
		t.Fatalf("loaded live grant=%#v err=%v", liveGrant, err)
	}
	mismatched := createdGrant
	mismatched.ImpactDigest = strings.Repeat("d", 64)
	if err := store.revokeExactGrant(ctx, mismatched, actorID, "wrong_identity"); !errors.Is(err, ErrTrustGrantNotFound) {
		t.Fatalf("mismatched exact revoke error=%v", err)
	}
	if granted, err := store.HasLiveGrant(ctx, identity); err != nil || !granted {
		t.Fatalf("mismatched revoke changed live grant=%t err=%v", granted, err)
	}
	if err := store.revokeExactGrant(ctx, createdGrant, actorID, "activation_failed"); err != nil {
		t.Fatal(err)
	}
	if err := store.revokeExactGrant(ctx, createdGrant, actorID, "activation_failed_replay"); err != nil {
		t.Fatalf("exact revoke must be idempotent: %v", err)
	}
	granted, err = store.HasLiveGrant(ctx, identity)
	if err != nil || granted {
		t.Fatalf("exact revoked grant=%v err=%v", granted, err)
	}

	// 重新授权后保留 RevokeAll 的生产覆盖。
	secondTokenHash := strings.Repeat("e", 64)
	if err := store.CreateChallenge(ctx, TrustChallengeRecord{
		TokenHash: secondTokenHash, ActorUserID: actorID, Identity: identity,
		ArtifactDigests: map[string]string{"package": identity.PackageDigest},
		Impact: TrustImpact{
			SchemaVersion: TrustImpactSchemaV1, Action: TrustActionEnable,
			ExtensionID: extensionID, ExtensionVersion: identity.ExtensionVersion,
			PackageDigest: identity.PackageDigest, ArtifactDigests: map[string]string{"package": identity.PackageDigest},
			Digest: identity.ImpactDigest,
		},
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConsumeChallenge(ctx, TrustConsumeInput{
		TokenHash: secondTokenHash, ActorUserID: actorID, Identity: identity,
	}); err != nil {
		t.Fatal(err)
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

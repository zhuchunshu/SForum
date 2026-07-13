package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestPostgresStoreRollbackExtensionVersionExactCASAndSnapshots(t *testing.T) {
	fixture := newStagedCASTestFixture(t, "exact-rollback")
	promoted, err := fixture.store.PromoteStagedVersion(fixture.ctx, fixture.input())
	if err != nil {
		t.Fatal(err)
	}
	rollback := rollbackToFixtureActive(fixture, promoted)

	wrongActive := rollback
	wrongActive.ExpectedActiveVersion = "2.0.1"
	if _, err := fixture.store.RollbackExtensionVersion(fixture.ctx, wrongActive); !errors.Is(err, ErrExtensionVersionConflict) {
		t.Fatalf("wrong active version error = %v", err)
	}
	foreign := newStagedCASTestFixture(t, "foreign-rollback")
	foreignTarget := rollback
	foreignTarget.TargetVersionID = foreign.activeID
	if _, err := fixture.store.RollbackExtensionVersion(fixture.ctx, foreignTarget); !errors.Is(err, ErrExtensionVersionConflict) {
		t.Fatalf("foreign rollback target error = %v", err)
	}

	rolledBack, err := fixture.store.RollbackExtensionVersion(fixture.ctx, rollback)
	if err != nil {
		t.Fatal(err)
	}
	if rolledBack.ActiveVersionID != fixture.activeID || rolledBack.Version != "1.0.0" ||
		rolledBack.PackageDigest != fixture.activeDigest || rolledBack.Status != StatusEnabled || rolledBack.StagedVersion != nil {
		t.Fatalf("rollback result = %#v", rolledBack)
	}
	if _, err := fixture.store.RollbackExtensionVersion(fixture.ctx, rollback); !errors.Is(err, ErrExtensionVersionConflict) {
		t.Fatalf("rollback replay error = %v", err)
	}

	activeSnapshot, err := fixture.store.GetExtensionVersion(fixture.ctx, ExactExtensionVersionInput{
		ExtensionID: fixture.id, Version: "1.0.0", PackageDigest: fixture.activeDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if activeSnapshot.ID != fixture.activeID || activeSnapshot.Manifest.Version != "1.0.0" ||
		activeSnapshot.PackagePath != fixture.activePath {
		t.Fatalf("exact active snapshot = %#v", activeSnapshot)
	}
	encoded, err := json.Marshal(activeSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	var publicSnapshot map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &publicSnapshot); err != nil {
		t.Fatal(err)
	}
	if _, leaked := publicSnapshot["id"]; leaked {
		t.Fatalf("database id leaked from version snapshot: %s", encoded)
	}
	if _, err := fixture.store.GetExtensionVersion(fixture.ctx, ExactExtensionVersionInput{
		ExtensionID: fixture.id, Version: "1.0.0", PackageDigest: strings.Repeat("f", 64),
	}); !errors.Is(err, ErrExtensionVersionNotFound) {
		t.Fatalf("wrong exact digest error = %v", err)
	}
	versions, err := fixture.store.ListExtensionVersions(fixture.ctx, fixture.id)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 || versions[0].ID != fixture.candidateID || versions[1].ID != fixture.activeID {
		t.Fatalf("version snapshots = %#v", versions)
	}
	if _, err := fixture.store.ListExtensionVersions(fixture.ctx, "missing.extension"); !errors.Is(err, ErrExtensionNotFound) {
		t.Fatalf("missing extension version list error = %v", err)
	}
	assertImmutableVersionRetained(t, fixture, fixture.activeID)
	assertImmutableVersionRetained(t, fixture, fixture.candidateID)
	assertStagedCASPackagesRetained(t, fixture)
}

func TestPostgresStoreRollbackExtensionVersionConcurrentOneWinner(t *testing.T) {
	fixture := newStagedCASTestFixture(t, "rollback-concurrent")
	v2, err := fixture.store.PromoteStagedVersion(fixture.ctx, fixture.input())
	if err != nil {
		t.Fatal(err)
	}
	v3Path := filepath.Join(t.TempDir(), "v3")
	if err := os.MkdirAll(v3Path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v3Path, "artifact.marker"), []byte("v3"), 0o600); err != nil {
		t.Fatal(err)
	}
	v3Digest := strings.Repeat("c", 64)
	stagedV3, err := fixture.store.SaveInstalled(fixture.ctx, SaveInstalledInput{
		Manifest: stagedVersionTestManifest(fixture.id, "3.0.0"), PackagePath: v3Path, PackageDigest: v3Digest,
	})
	if err != nil || stagedV3.StagedVersion == nil {
		t.Fatalf("stage v3 = %#v, %v", stagedV3, err)
	}
	if _, err := fixture.store.RollbackExtensionVersion(fixture.ctx, RollbackExtensionVersionInput{
		ExtensionID:             fixture.id,
		ExpectedActiveVersionID: v2.ActiveVersionID, ExpectedActiveVersion: v2.Version,
		ExpectedActivePackageDigest: v2.PackageDigest,
		TargetVersionID:             stagedV3.StagedVersion.ID, TargetVersion: stagedV3.StagedVersion.Version,
		TargetPackageDigest: stagedV3.StagedVersion.PackageDigest,
	}); !errors.Is(err, ErrExtensionVersionConflict) {
		t.Fatalf("staged rollback target error = %v", err)
	}
	v3, err := fixture.store.PromoteStagedVersion(fixture.ctx, StagedVersionCASInput{
		ExtensionID:             fixture.id,
		ExpectedActiveVersionID: v2.ActiveVersionID, ExpectedActiveVersion: v2.Version,
		ExpectedActivePackageDigest: v2.PackageDigest,
		ExpectedStagedVersionID:     stagedV3.StagedVersion.ID, ExpectedStagedVersion: stagedV3.StagedVersion.Version,
		ExpectedPackageDigest: stagedV3.StagedVersion.PackageDigest,
	})
	if err != nil {
		t.Fatal(err)
	}

	toV1 := RollbackExtensionVersionInput{
		ExtensionID:             fixture.id,
		ExpectedActiveVersionID: v3.ActiveVersionID, ExpectedActiveVersion: v3.Version,
		ExpectedActivePackageDigest: v3.PackageDigest,
		TargetVersionID:             fixture.activeID, TargetVersion: "1.0.0", TargetPackageDigest: fixture.activeDigest,
	}
	toV2 := toV1
	toV2.TargetVersionID = fixture.candidateID
	toV2.TargetVersion = "2.0.0"
	toV2.TargetPackageDigest = fixture.candidateDigest

	start := make(chan struct{})
	results := make(chan error, 2)
	for _, input := range []RollbackExtensionVersionInput{toV1, toV2} {
		go func(input RollbackExtensionVersionInput) {
			<-start
			_, err := fixture.store.RollbackExtensionVersion(fixture.ctx, input)
			results <- err
		}(input)
	}
	close(start)
	var succeeded, conflicted int
	for range 2 {
		switch err := <-results; {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrExtensionVersionConflict):
			conflicted++
		default:
			t.Fatalf("concurrent rollback error = %v", err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("concurrent rollback succeeded=%d conflicted=%d", succeeded, conflicted)
	}
	current, err := fixture.store.Get(fixture.ctx, fixture.id)
	if err != nil {
		t.Fatal(err)
	}
	if current.ActiveVersionID != fixture.activeID && current.ActiveVersionID != fixture.candidateID {
		t.Fatalf("unexpected rollback winner = %#v", current)
	}
	versions, err := fixture.store.ListExtensionVersions(fixture.ctx, fixture.id)
	if err != nil || len(versions) != 3 {
		t.Fatalf("retained concurrent versions = %#v, %v", versions, err)
	}
	if _, err := os.Stat(filepath.Join(v3Path, "artifact.marker")); err != nil {
		t.Fatalf("v3 artifact was removed: %v", err)
	}
}

func TestPostgresStoreExactVersionSnapshotsDistinguishSameSemanticVersion(t *testing.T) {
	fixture := newStagedCASTestFixture(t, "same-version-snapshots")
	replacementPath := filepath.Join(t.TempDir(), "v1-repacked")
	if err := os.MkdirAll(replacementPath, 0o755); err != nil {
		t.Fatal(err)
	}
	replacementDigest := strings.Repeat("d", 64)
	repacked, err := fixture.store.SaveInstalled(fixture.ctx, SaveInstalledInput{
		Manifest:    stagedVersionTestManifest(fixture.id, "1.0.0"),
		PackagePath: replacementPath, PackageDigest: replacementDigest,
	})
	if err != nil || repacked.StagedVersion == nil {
		t.Fatalf("stage repacked semantic version = %#v, %v", repacked, err)
	}

	original, err := fixture.store.GetExtensionVersion(fixture.ctx, ExactExtensionVersionInput{
		ExtensionID: fixture.id, Version: "1.0.0", PackageDigest: fixture.activeDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := fixture.store.GetExtensionVersion(fixture.ctx, ExactExtensionVersionInput{
		ExtensionID: fixture.id, Version: "1.0.0", PackageDigest: replacementDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if original.ID == replacement.ID || original.PackagePath == replacement.PackagePath ||
		replacement.ID != repacked.StagedVersion.ID || replacement.PackagePath != replacementPath {
		t.Fatalf("same-version exact snapshots original=%#v replacement=%#v", original, replacement)
	}
	versions, err := fixture.store.ListExtensionVersions(fixture.ctx, fixture.id)
	if err != nil || len(versions) != 3 {
		t.Fatalf("same-version history = %#v, %v", versions, err)
	}
	assertImmutableVersionRetained(t, fixture, fixture.activeID)
	assertImmutableVersionRetained(t, fixture, fixture.candidateID)
	assertStagedCASPackagesRetained(t, fixture)
}

func TestPostgresStoreRollbackExtensionVersionRollsBackWriteFailure(t *testing.T) {
	fixture := newStagedCASTestFixture(t, "rollback-write-failure")
	promoted, err := fixture.store.PromoteStagedVersion(fixture.ctx, fixture.input())
	if err != nil {
		t.Fatal(err)
	}
	rollback := rollbackToFixtureActive(fixture, promoted)
	functionName := fmt.Sprintf("test_version_rollback_fail_%d", time.Now().UnixNano())
	triggerName := functionName + "_trigger"
	quotedFunction := pgx.Identifier{functionName}.Sanitize()
	quotedTrigger := pgx.Identifier{triggerName}.Sanitize()
	if _, err := fixture.pool.Exec(fixture.ctx, fmt.Sprintf(`
		CREATE FUNCTION %s() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
		  RAISE EXCEPTION 'forced exact version rollback failure';
		END
		$$
	`, quotedFunction)); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, fmt.Sprintf(`
		CREATE TRIGGER %s
		BEFORE UPDATE OF active_version_id ON extensions
		FOR EACH ROW WHEN (OLD.id = '%s')
		EXECUTE FUNCTION %s()
	`, quotedTrigger, strings.ReplaceAll(fixture.id, "'", "''"), quotedFunction)); err != nil {
		_, _ = fixture.pool.Exec(fixture.ctx, fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, quotedFunction))
		t.Fatal(err)
	}
	dropFailureTrigger := func() {
		_, _ = fixture.pool.Exec(context.Background(), fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON extensions`, quotedTrigger))
		_, _ = fixture.pool.Exec(context.Background(), fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, quotedFunction))
	}
	t.Cleanup(dropFailureTrigger)

	if _, err := fixture.store.RollbackExtensionVersion(fixture.ctx, rollback); err == nil {
		t.Fatal("forced rollback failure unexpectedly succeeded")
	}
	assertStagedCASState(t, fixture, fixture.candidateID, fixture.candidateDigest, 0)
	assertImmutableVersionRetained(t, fixture, fixture.activeID)
	assertImmutableVersionRetained(t, fixture, fixture.candidateID)
	assertStagedCASPackagesRetained(t, fixture)

	dropFailureTrigger()
	if _, err := fixture.store.RollbackExtensionVersion(fixture.ctx, rollback); err != nil {
		t.Fatal(err)
	}
}

func rollbackToFixtureActive(fixture stagedCASTestFixture, current Extension) RollbackExtensionVersionInput {
	return RollbackExtensionVersionInput{
		ExtensionID:             fixture.id,
		ExpectedActiveVersionID: current.ActiveVersionID, ExpectedActiveVersion: current.Version,
		ExpectedActivePackageDigest: current.PackageDigest,
		TargetVersionID:             fixture.activeID, TargetVersion: "1.0.0", TargetPackageDigest: fixture.activeDigest,
	}
}

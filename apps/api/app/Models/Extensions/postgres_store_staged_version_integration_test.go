package extensions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresStoreStaticUpgradeStagesWithoutChangingActiveVersion(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("SFORUM_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("SFORUM_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	id := fmt.Sprintf("test.staged-version.%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM extensions WHERE id = $1`, id)
	})
	store := NewPostgresStore(pool)

	v1 := stagedVersionTestManifest(id, "1.0.0")
	first, err := store.SaveInstalled(ctx, SaveInstalledInput{
		Manifest: v1, PackagePath: "/tmp/staged-v1", PackageDigest: strings.Repeat("a", 64),
	})
	if err != nil {
		t.Fatalf("save initial version: %v", err)
	}
	if first.Version != "1.0.0" || first.ActiveVersionID == 0 || first.StagedVersion != nil || first.Status != StatusInstalled {
		t.Fatalf("unexpected initial extension: %#v", first)
	}
	activeVersionID := first.ActiveVersionID
	if _, err := store.Enable(ctx, id, TypePlugin); err != nil {
		t.Fatalf("enable initial version: %v", err)
	}

	v2 := stagedVersionTestManifest(id, "2.0.0")
	staged, err := store.SaveInstalled(ctx, SaveInstalledInput{
		Manifest: v2, PackagePath: "/tmp/staged-v2", PackageDigest: strings.Repeat("b", 64),
	})
	if err != nil {
		t.Fatalf("stage upgrade: %v", err)
	}
	assertStaticUpgradeStaged(t, staged, activeVersionID)

	replayed, err := store.SaveInstalled(ctx, SaveInstalledInput{
		Manifest: v2, PackagePath: "/tmp/staged-v2-replay", PackageDigest: strings.Repeat("b", 64),
	})
	if err != nil {
		t.Fatalf("replay staged upgrade: %v", err)
	}
	assertStaticUpgradeStaged(t, replayed, activeVersionID)
	if replayed.StagedVersion.ID != staged.StagedVersion.ID {
		t.Fatalf("replay allocated another immutable version: first=%d replay=%d", staged.StagedVersion.ID, replayed.StagedVersion.ID)
	}

	var versionCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM extension_versions WHERE extension_id = $1
	`, id).Scan(&versionCount); err != nil {
		t.Fatal(err)
	}
	if versionCount != 2 {
		t.Fatalf("version count=%d, want 2", versionCount)
	}

	typeChange := stagedVersionTestManifest(id, "3.0.0")
	typeChange.Type = TypeTheme
	if _, err := store.SaveInstalled(ctx, SaveInstalledInput{
		Manifest: typeChange, PackagePath: "/tmp/staged-v3", PackageDigest: strings.Repeat("c", 64),
	}); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("extension type change error=%v, want ErrInvalidManifest", err)
	}
}

func stagedVersionTestManifest(id, version string) Manifest {
	return Manifest{ID: id, Name: "Staged Version Test", Version: version, Type: TypePlugin}
}

func assertStaticUpgradeStaged(t *testing.T, extension Extension, activeVersionID int64) {
	t.Helper()
	if extension.Version != "1.0.0" || extension.PackageDigest != strings.Repeat("a", 64) {
		t.Fatalf("static upload changed active artifact: %#v", extension)
	}
	if extension.ActiveVersionID != activeVersionID || extension.Status != StatusEnabled {
		t.Fatalf("static upload changed active identity/status: %#v", extension)
	}
	if extension.StagedVersion == nil || extension.StagedVersion.Version != "2.0.0" ||
		extension.StagedVersion.PackageDigest != strings.Repeat("b", 64) {
		t.Fatalf("candidate was not staged: %#v", extension.StagedVersion)
	}
}

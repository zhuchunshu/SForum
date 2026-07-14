package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestPostgresProviderSlotSelectionExactArtifactCASAuditAndInvalidation(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("SFORUM_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("SFORUM_TEST_DATABASE_URL or DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var tablesReady bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('extension_provider_slot_selections') IS NOT NULL`).Scan(&tablesReady); err != nil {
		t.Fatal(err)
	}
	if !tablesReady {
		t.Skip("provider slot selection migration is not applied")
	}

	registry, owner, provider := providerSelectionFixture(t, "next")
	unique := fmt.Sprintf("provider-slot-%d", time.Now().UnixNano())
	owner.ID, owner.Manifest.ID = unique+".owner", unique+".owner"
	provider.ID, provider.Manifest.ID = unique+".candidate", unique+".candidate"
	owner.Manifest.Providers[0].ID = owner.ID + ".delivery"
	owner.Manifest.Providers[0].ContractVersion = owner.Manifest.Providers[0].ID + "@1"
	owner.Manifest.Providers[0].Slot = owner.ID + ".slot"
	provider.Manifest.Providers[0].ID = provider.ID + ".delivery"
	provider.Manifest.Providers[0].TargetID = owner.Manifest.Providers[0].ID
	provider.Manifest.Providers[0].ContractVersion = owner.Manifest.Providers[0].ContractVersion
	provider.Manifest.Providers[0].Slot = owner.Manifest.Providers[0].Slot
	provider.Manifest.Dependencies[0].ID = owner.ID
	registry = NewVersionedProviderSlotRegistry()
	if err := registry.ReplaceRuntime(owner, "owner-runtime"); err != nil {
		t.Fatal(err)
	}
	if err := registry.ReplaceRuntime(provider, "provider-runtime"); err != nil {
		t.Fatal(err)
	}

	var actorID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users
		(username,username_lower,email,email_lower,display_name) VALUES ($1,$1,$2,$2,$1) RETURNING id`,
		unique, unique+"@example.test").Scan(&actorID); err != nil {
		t.Fatal(err)
	}
	ownerVersionID := insertProviderSlotSelectionExtension(t, ctx, pool, owner, unique+"-owner")
	providerVersionID := insertProviderSlotSelectionExtension(t, ctx, pool, provider, unique+"-candidate")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM extension_provider_slot_selections WHERE contract_id=$1`, owner.Manifest.Providers[0].ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM extension_provider_slot_selection_events WHERE contract_id=$1`, owner.Manifest.Providers[0].ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM extensions WHERE id=ANY($1)`, []string{owner.ID, provider.ID})
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, actorID)
	})

	store := NewPostgresProviderSlotSelectionStore(pool)
	api := NewProviderSlotSelectionAPI(registry, store)
	selection, err := api.Select(ctx, owner.Manifest.Providers[0].ID, provider.Manifest.Providers[0].ID, 0, actorID, 71)
	if err != nil {
		t.Fatal(err)
	}
	if selection.ContractVersionID != ownerVersionID || selection.ProviderVersionID != providerVersionID || selection.Revision != 1 {
		t.Fatalf("selection = %#v", selection)
	}
	if _, err := api.Select(ctx, selection.ContractID, selection.CandidateID, 0, actorID, 72); !errors.Is(err, ErrProviderSlotSelectionRevisionConflict) {
		t.Fatalf("stale selection CAS = %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE extensions SET status='disabled' WHERE id=$1`, provider.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Selected(ctx, selection.ContractID); !errors.Is(err, ErrProviderSlotSelectionStale) {
		t.Fatalf("disabled provider selection = %v", err)
	}
	desired, err := store.Desired(ctx, selection.ContractID)
	if err != nil || desired.Revision != 1 {
		t.Fatalf("desired stale selection = %#v, %v", desired, err)
	}
	count, err := api.InvalidateExtension(ctx, InvalidateProviderSlotRequest{
		ExtensionID: provider.ID, ActorUserID: actorID, AuditEventID: 73, ReasonCode: "extension_disabled",
	})
	if err != nil || count != 1 {
		t.Fatalf("invalidate count=%d err=%v", count, err)
	}
	events, err := api.Events(ctx, selection.ContractID, 10)
	if err != nil || len(events) != 2 || events[0].Action != "invalidate" || events[0].PreviousSelection == nil {
		t.Fatalf("events = %#v, %v", events, err)
	}
}

func insertProviderSlotSelectionExtension(t *testing.T, ctx context.Context, pool *pgxpool.Pool, extension extensions.Extension, path string) int64 {
	t.Helper()
	if _, err := pool.Exec(ctx, `INSERT INTO extensions (id,type,name,status,source,is_system,is_deletable)
		VALUES ($1,'plugin',$1,'enabled','uploaded',false,true)`, extension.ID); err != nil {
		t.Fatal(err)
	}
	var versionID int64
	if err := pool.QueryRow(ctx, `INSERT INTO extension_versions
		(extension_id,version,manifest,package_path,package_digest) VALUES ($1,$2,'{}'::jsonb,$3,$4) RETURNING id`,
		extension.ID, extension.Version, "/tmp/"+path, extension.PackageDigest).Scan(&versionID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE extensions SET active_version_id=$2 WHERE id=$1`, extension.ID, versionID); err != nil {
		t.Fatal(err)
	}
	return versionID
}

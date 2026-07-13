package main

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

func TestPostgresRecoveryRepositoryWorksWithMalformedPackages(t *testing.T) {
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
	builtinID := "recovery.builtin." + suffix
	firstID := "recovery.first." + suffix
	secondID := "recovery.second." + suffix
	for _, item := range []struct {
		id, source string
		system     bool
	}{
		{builtinID, "builtin", true},
		{firstID, "uploaded", false},
		{secondID, "uploaded", false},
	} {
		if err := insertRecoveryFixture(ctx, pool, item.id, item.source, item.system); err != nil {
			t.Fatal(err)
		}
		defer pool.Exec(ctx, `DELETE FROM extensions WHERE id = $1`, item.id)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO mail_provider_selection (slot, extension_id) VALUES ('mail.provider', $1)`, firstID); err != nil {
		t.Fatal(err)
	}

	repository := &postgresRecoveryRepository{pool: pool}
	if _, err := repository.Disable(ctx, builtinID); !errors.Is(err, errRecoveryProtected) {
		t.Fatalf("disable protected extension: %v", err)
	}
	disabled, err := repository.Disable(ctx, firstID)
	if err != nil || disabled.Status != "disabled" {
		t.Fatalf("disable first=%#v err=%v", disabled, err)
	}
	var selectedMail int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM mail_provider_selection WHERE extension_id = $1`, firstID).Scan(&selectedMail); err != nil || selectedMail != 0 {
		t.Fatalf("mail provider selection count=%d err=%v", selectedMail, err)
	}
	items, err := repository.DisableAllThirdParty(ctx)
	if err != nil || !recoveryItemsContain(items, firstID) || !recoveryItemsContain(items, secondID) {
		t.Fatalf("disable all items=%#v err=%v", items, err)
	}
	var builtinStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM extensions WHERE id = $1`, builtinID).Scan(&builtinStatus); err != nil || builtinStatus != "enabled" {
		t.Fatalf("builtin status=%q err=%v", builtinStatus, err)
	}
	var audits int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_events
		WHERE action = 'extension.cli_recovery'
		  AND (metadata->'extensionIds') ?| $1::text[]
	`, []string{firstID, secondID}).Scan(&audits); err != nil || audits < 2 {
		t.Fatalf("recovery audits=%d err=%v", audits, err)
	}
}

func recoveryItemsContain(items []recoveryExtension, extensionID string) bool {
	for _, item := range items {
		if item.ID == extensionID {
			return true
		}
	}
	return false
}

func insertRecoveryFixture(ctx context.Context, pool *pgxpool.Pool, id, source string, system bool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var versionID int64
	if _, err := tx.Exec(ctx, `
		INSERT INTO extensions (id, type, name, status, source, is_system, is_deletable)
		VALUES ($1, 'plugin', $1, 'enabled', $2, $3, NOT $3)
	`, id, source, system); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO extension_versions (extension_id, version, manifest, package_path, package_digest)
		VALUES ($1, '1.0.0', '{}'::jsonb, '/missing/malformed/package', $2)
		RETURNING id
	`, id, strings.Repeat("a", 64)).Scan(&versionID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE extensions SET active_version_id = $2 WHERE id = $1`, id, versionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

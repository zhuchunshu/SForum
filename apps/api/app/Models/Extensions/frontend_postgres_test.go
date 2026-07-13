package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPostgresFrontendTrustStoreCreateCanonicalizesComponents(t *testing.T) {
	grant := frontendGrantFixture()
	db := &fakeFrontendTrustDB{rows: []pgx.Row{frontendGrantTestRow(grant)}}
	store := newPostgresFrontendTrustStore(db)

	created, err := store.CreateFrontendGrant(context.Background(), FrontendTrustGrantInput{
		ExtensionID: grant.ExtensionID, ExtensionVersion: grant.ExtensionVersion,
		PackageDigest: grant.PackageDigest, AdminFrontendDigest: grant.AdminFrontendDigest,
		APIVersion: grant.APIVersion, ComponentIDs: []string{"settings", "settings"}, GrantedByUserID: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.ComponentIDs) != 1 || created.ComponentIDs[0] != "settings" {
		t.Fatalf("components not canonical: %#v", created.ComponentIDs)
	}
	if len(db.queries) != 1 || !strings.Contains(db.queries[0].sql, "ON CONFLICT") {
		t.Fatalf("unexpected insert query: %#v", db.queries)
	}
	if got := string(db.queries[0].args[5].([]byte)); got != `["settings"]` {
		t.Fatalf("unexpected component JSON: %s", got)
	}
}

func TestPostgresFrontendTrustStoreCreateRejectsChangedPackageIdentity(t *testing.T) {
	existing := frontendGrantFixture()
	db := &fakeFrontendTrustDB{rows: []pgx.Row{
		frontendTrustErrorRow(pgx.ErrNoRows),
		frontendGrantTestRow(existing),
	}}
	_, err := newPostgresFrontendTrustStore(db).CreateFrontendGrant(context.Background(), FrontendTrustGrantInput{
		ExtensionID: existing.ExtensionID, ExtensionVersion: existing.ExtensionVersion,
		PackageDigest: strings.Repeat("c", 64), AdminFrontendDigest: existing.AdminFrontendDigest,
		APIVersion: existing.APIVersion, ComponentIDs: existing.ComponentIDs,
	})
	if !errors.Is(err, ErrFrontendGrantConflict) {
		t.Fatalf("expected immutable conflict, got %v", err)
	}
}

func TestPostgresFrontendTrustStoreRevokeIsImmediateAndIdempotent(t *testing.T) {
	revoked := frontendGrantFixture()
	now := time.Now()
	revoked.RevokedAt = &now
	db := &fakeFrontendTrustDB{rows: []pgx.Row{frontendGrantTestRow(revoked)}}
	store := newPostgresFrontendTrustStore(db)

	result, err := store.RevokeFrontendGrant(context.Background(), FrontendRevocationInput{
		ExtensionID: revoked.ExtensionID, ExtensionVersion: revoked.ExtensionVersion,
		AdminFrontendDigest: revoked.AdminFrontendDigest, RequestedByUserID: 9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.RevokedAt == nil || !strings.Contains(db.queries[0].sql, "SET revoked_at = now()") {
		t.Fatalf("revoke was not direct: %#v", result)
	}
}

func TestPostgresFrontendTrustStoreRevokeAllCanFilterExtension(t *testing.T) {
	db := &fakeFrontendTrustDB{execTag: pgconn.NewCommandTag("UPDATE 2")}
	if err := newPostgresFrontendTrustStore(db).RevokeAllFrontendGrants(context.Background(), "demo.plugin", 7); err != nil {
		t.Fatal(err)
	}
	if len(db.execs) != 1 || db.execs[0].args[0] != "demo.plugin" || !strings.Contains(db.execs[0].sql, "revoked_at = now()") {
		t.Fatalf("unexpected revoke all query: %#v", db.execs)
	}
}

func frontendGrantFixture() FrontendTrustGrant {
	return FrontendTrustGrant{
		ID: 41, ExtensionID: "demo.plugin", ExtensionVersion: "1.0.0",
		PackageDigest: strings.Repeat("a", 64), AdminFrontendDigest: strings.Repeat("b", 64),
		APIVersion: 1, ComponentIDs: []string{"settings"}, GrantedByUserID: 7,
		GrantedAt: time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC),
	}
}

type frontendTrustQuery struct {
	sql  string
	args []any
}

type fakeFrontendTrustDB struct {
	rows         []pgx.Row
	queryResults []pgx.Rows
	queryErr     error
	queries      []frontendTrustQuery
	execs        []frontendTrustQuery
	execTag      pgconn.CommandTag
	execErr      error
}

func (db *fakeFrontendTrustDB) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	db.queries = append(db.queries, frontendTrustQuery{sql: sql, args: args})
	if len(db.rows) == 0 {
		return frontendTrustErrorRow(errors.New("unexpected QueryRow"))
	}
	row := db.rows[0]
	db.rows = db.rows[1:]
	return row
}

func (db *fakeFrontendTrustDB) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	db.queries = append(db.queries, frontendTrustQuery{sql: sql, args: args})
	if db.queryErr != nil {
		return nil, db.queryErr
	}
	if len(db.queryResults) == 0 {
		return nil, errors.New("unexpected Query")
	}
	rows := db.queryResults[0]
	db.queryResults = db.queryResults[1:]
	return rows, nil
}

func (db *fakeFrontendTrustDB) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	db.execs = append(db.execs, frontendTrustQuery{sql: sql, args: args})
	return db.execTag, db.execErr
}

type frontendTrustRowFunc func(dest ...any) error

func (row frontendTrustRowFunc) Scan(dest ...any) error { return row(dest...) }

func frontendTrustErrorRow(err error) pgx.Row {
	return frontendTrustRowFunc(func(...any) error { return err })
}

func frontendGrantTestRow(grant FrontendTrustGrant) pgx.Row {
	components, _ := json.Marshal(grant.ComponentIDs)
	return frontendTrustRowFunc(func(dest ...any) error {
		if len(dest) != 11 {
			return errors.New("unexpected frontend grant scan width")
		}
		*dest[0].(*int64) = grant.ID
		*dest[1].(*string) = grant.ExtensionID
		*dest[2].(*string) = grant.ExtensionVersion
		*dest[3].(*string) = grant.PackageDigest
		*dest[4].(*string) = grant.AdminFrontendDigest
		*dest[5].(*int) = grant.APIVersion
		*dest[6].(*[]byte) = components
		*dest[7].(*int64) = grant.GrantedByUserID
		*dest[8].(*time.Time) = grant.GrantedAt
		*dest[9].(**time.Time) = grant.RevokedAt
		*dest[10].(*int64) = grant.RevokedByUserID
		return nil
	})
}

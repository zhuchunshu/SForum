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

func TestPostgresFrontendTrustStoreCreateCanonicalizesJSONSets(t *testing.T) {
	grantedAt := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	db := &fakeFrontendTrustDB{rows: []pgx.Row{frontendGrantTestRow(FrontendTrustGrant{
		ID:                  41,
		ExtensionID:         "demo.plugin",
		ExtensionVersion:    "1.0.0",
		PackageDigest:       strings.Repeat("a", 64),
		AdminFrontendDigest: strings.Repeat("b", 64),
		APIVersion:          1,
		ContributionPoints:  []string{"admin.test.secondary", "admin.test.fixture"},
		ComponentIDs:        []string{"secondary", "primary"},
		GrantedByUserID:     7,
		GrantedAt:           grantedAt,
	})}}
	store := newPostgresFrontendTrustStore(db)

	grant, err := store.CreateFrontendGrant(context.Background(), FrontendTrustGrantInput{
		ExtensionID:         "demo.plugin",
		ExtensionVersion:    "1.0.0",
		PackageDigest:       strings.Repeat("a", 64),
		AdminFrontendDigest: strings.Repeat("b", 64),
		APIVersion:          1,
		ContributionPoints:  []string{"admin.test.fixture", "admin.test.secondary", "admin.test.fixture"},
		ComponentIDs:        []string{"primary", "secondary", "primary"},
		GrantedByUserID:     7,
	})
	if err != nil {
		t.Fatalf("create frontend grant: %v", err)
	}
	if !equalStrings(grant.ContributionPoints, []string{"admin.test.fixture", "admin.test.secondary"}) {
		t.Fatalf("scan must return sorted points, got %#v", grant.ContributionPoints)
	}
	if !equalStrings(grant.ComponentIDs, []string{"primary", "secondary"}) {
		t.Fatalf("scan must return sorted components, got %#v", grant.ComponentIDs)
	}
	if len(db.queries) != 1 || !strings.Contains(db.queries[0].sql, "ON CONFLICT") {
		t.Fatalf("expected immutable insert query, got %#v", db.queries)
	}
	if got := string(db.queries[0].args[5].([]byte)); got != `["admin.test.fixture","admin.test.secondary"]` {
		t.Fatalf("points JSON is not canonical: %s", got)
	}
	if got := string(db.queries[0].args[6].([]byte)); got != `["primary","secondary"]` {
		t.Fatalf("components JSON is not canonical: %s", got)
	}
}

func TestPostgresFrontendTrustStoreCreateReusesOnlyIdenticalLiveGrant(t *testing.T) {
	existing := frontendGrantFixture()
	tests := []struct {
		name    string
		input   FrontendTrustGrantInput
		wantErr error
	}{
		{
			name: "identical grant is idempotent",
			input: FrontendTrustGrantInput{
				ExtensionID:         existing.ExtensionID,
				ExtensionVersion:    existing.ExtensionVersion,
				PackageDigest:       existing.PackageDigest,
				AdminFrontendDigest: existing.AdminFrontendDigest,
				APIVersion:          existing.APIVersion,
				ContributionPoints:  []string{"admin.test.fixture"},
				ComponentIDs:        []string{"fixture.panel"},
				GrantedByUserID:     999,
			},
		},
		{
			name: "different declaration never overwrites",
			input: FrontendTrustGrantInput{
				ExtensionID:         existing.ExtensionID,
				ExtensionVersion:    existing.ExtensionVersion,
				PackageDigest:       existing.PackageDigest,
				AdminFrontendDigest: existing.AdminFrontendDigest,
				APIVersion:          existing.APIVersion + 1,
				ContributionPoints:  []string{"admin.test.fixture"},
				ComponentIDs:        []string{"fixture.panel"},
				GrantedByUserID:     999,
			},
			wantErr: ErrFrontendGrantConflict,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := &fakeFrontendTrustDB{rows: []pgx.Row{
				frontendTrustErrorRow(pgx.ErrNoRows),
				frontendGrantTestRow(existing),
			}}
			grant, err := newPostgresFrontendTrustStore(db).CreateFrontendGrant(context.Background(), test.input)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
			if test.wantErr == nil && (grant.ID != existing.ID || grant.GrantedByUserID != existing.GrantedByUserID) {
				t.Fatalf("idempotent create overwrote audit identity: %#v", grant)
			}
			if len(db.queries) != 2 || !strings.Contains(db.queries[1].sql, "revoked_at IS NULL") {
				t.Fatalf("expected live-grant fallback query, got %#v", db.queries)
			}
		})
	}
}

func TestPostgresFrontendTrustStoreDoesNotReusePendingGrant(t *testing.T) {
	existing := frontendGrantFixture()
	requestedAt := existing.GrantedAt.Add(time.Minute)
	existing.RevocationRequestedAt = &requestedAt
	db := &fakeFrontendTrustDB{rows: []pgx.Row{
		frontendTrustErrorRow(pgx.ErrNoRows),
		frontendGrantTestRow(existing),
	}}

	_, err := newPostgresFrontendTrustStore(db).CreateFrontendGrant(context.Background(), FrontendTrustGrantInput{
		ExtensionID:         existing.ExtensionID,
		ExtensionVersion:    existing.ExtensionVersion,
		PackageDigest:       existing.PackageDigest,
		AdminFrontendDigest: existing.AdminFrontendDigest,
		APIVersion:          existing.APIVersion,
		ContributionPoints:  existing.ContributionPoints,
		ComponentIDs:        existing.ComponentIDs,
		GrantedByUserID:     99,
	})
	if !errors.Is(err, ErrFrontendGrantStateConflict) {
		t.Fatalf("expected pending grant state conflict, got %v", err)
	}
}

func TestPostgresFrontendTrustStoreRequestsRevocationWithCAS(t *testing.T) {
	pending := frontendGrantFixture()
	requestedAt := pending.GrantedAt.Add(time.Hour)
	pending.RevocationRequestedAt = &requestedAt
	pending.RevocationRequestedByUserID = 9

	t.Run("active grant becomes pending", func(t *testing.T) {
		db := &fakeFrontendTrustDB{rows: []pgx.Row{frontendGrantTestRow(pending)}}
		grant, err := newPostgresFrontendTrustStore(db).RequestFrontendRevocation(context.Background(), FrontendRevocationInput{
			ExtensionID:         pending.ExtensionID,
			ExtensionVersion:    pending.ExtensionVersion,
			PackageDigest:       pending.PackageDigest,
			AdminFrontendDigest: pending.AdminFrontendDigest,
			RequestedByUserID:   9,
		})
		if err != nil {
			t.Fatalf("request revocation: %v", err)
		}
		if grant.RevocationRequestedAt == nil || grant.RevocationRequestedByUserID != 9 {
			t.Fatalf("revocation request was not persisted: %#v", grant)
		}
		if !strings.Contains(db.queries[0].sql, "revocation_requested_at IS NULL") || !strings.Contains(db.queries[0].sql, "revoked_at IS NULL") {
			t.Fatalf("revocation update is not a CAS: %s", db.queries[0].sql)
		}
		if !strings.Contains(db.queries[0].sql, "admin_frontend_digest = $3") || db.queries[0].args[2] != pending.AdminFrontendDigest {
			t.Fatalf("revocation must target the dedicated admin digest: %#v", db.queries[0])
		}
	})

	t.Run("pending grant rejects a second transition", func(t *testing.T) {
		db := &fakeFrontendTrustDB{rows: []pgx.Row{
			frontendTrustErrorRow(pgx.ErrNoRows),
			frontendGrantTestRow(pending),
		}}
		_, err := newPostgresFrontendTrustStore(db).RequestFrontendRevocation(context.Background(), FrontendRevocationInput{
			ExtensionID:         pending.ExtensionID,
			ExtensionVersion:    pending.ExtensionVersion,
			PackageDigest:       pending.PackageDigest,
			AdminFrontendDigest: pending.AdminFrontendDigest,
			RequestedByUserID:   10,
		})
		if !errors.Is(err, ErrFrontendGrantStateConflict) {
			t.Fatalf("expected state conflict, got %v", err)
		}
	})

	t.Run("missing live grant stays distinguishable", func(t *testing.T) {
		db := &fakeFrontendTrustDB{rows: []pgx.Row{
			frontendTrustErrorRow(pgx.ErrNoRows),
			frontendTrustErrorRow(pgx.ErrNoRows),
		}}
		_, err := newPostgresFrontendTrustStore(db).RequestFrontendRevocation(context.Background(), FrontendRevocationInput{
			ExtensionID:         pending.ExtensionID,
			ExtensionVersion:    pending.ExtensionVersion,
			PackageDigest:       pending.PackageDigest,
			AdminFrontendDigest: pending.AdminFrontendDigest,
			RequestedByUserID:   10,
		})
		if !errors.Is(err, ErrFrontendGrantNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})
}

func TestPostgresFrontendTrustStoreFinalizesOnlyPendingGrantsOutsideRelease(t *testing.T) {
	db := &fakeFrontendTrustDB{execTag: pgconn.NewCommandTag("UPDATE 2")}
	store := newPostgresFrontendTrustStore(db)
	if err := store.FinalizeFrontendRevocations(context.Background(), 88); err != nil {
		t.Fatalf("finalize frontend revocations: %v", err)
	}
	if len(db.execs) != 1 || len(db.execs[0].args) != 1 || db.execs[0].args[0] != int64(88) {
		t.Fatalf("unexpected finalize call: %#v", db.execs)
	}
	sql := compactSQL(db.execs[0].sql)
	for _, clause := range []string{
		"FROM web_releases",
		"status = 'active'",
		"revocation_requested_at IS NOT NULL",
		"revoked_at IS NULL",
		"FROM web_release_extensions",
		"release_extensions.web_release_id = target_release.id",
		"release_extensions.extension_id = grants.extension_id",
		"release_extensions.extension_version = grants.extension_version",
		"release_extensions.admin_frontend_digest = grants.admin_frontend_digest",
	} {
		if !strings.Contains(sql, clause) {
			t.Fatalf("finalize SQL missing %q: %s", clause, sql)
		}
	}
	if strings.Contains(sql, "target_release.created_at") {
		t.Fatalf("active releases that already exclude a plugin must support immediate revocation: %s", sql)
	}
}

func TestPostgresFrontendTrustStoreRejectsInvalidStoredJSON(t *testing.T) {
	grant := frontendGrantFixture()
	db := &fakeFrontendTrustDB{rows: []pgx.Row{frontendGrantRawJSONTestRow(grant, []byte(`{}`), []byte(`[]`))}}
	_, err := newPostgresFrontendTrustStore(db).FrontendGrant(context.Background(), grant.ExtensionID, grant.ExtensionVersion, grant.AdminFrontendDigest)
	if err == nil || !strings.Contains(err.Error(), "decode frontend trust contribution points") {
		t.Fatalf("expected stable JSON scan error, got %v", err)
	}
	if len(db.queries) != 1 || !strings.Contains(db.queries[0].sql, "admin_frontend_digest = $3") || db.queries[0].args[2] != grant.AdminFrontendDigest {
		t.Fatalf("exact trust lookup must bind the dedicated admin digest: %#v", db.queries)
	}
}

func TestPostgresFrontendTrustStoreListsHistoricalLiveGrants(t *testing.T) {
	first := frontendGrantFixture()
	second := first
	second.ID = 13
	second.ExtensionVersion = "2.0.0"
	db := &fakeFrontendTrustDB{queryResults: []pgx.Rows{&frontendGrantRows{rows: []pgx.Row{
		frontendGrantTestRow(first),
		frontendGrantTestRow(second),
	}}}}

	items, err := newPostgresFrontendTrustStore(db).LiveFrontendGrants(context.Background(), first.ExtensionID)
	if err != nil {
		t.Fatalf("list live frontend grants: %v", err)
	}
	if len(items) != 2 || items[0].ID != first.ID || items[1].ID != second.ID {
		t.Fatalf("historical live grants were not returned: %#v", items)
	}
	if len(db.queries) != 1 || !strings.Contains(db.queries[0].sql, "revoked_at IS NULL") || db.queries[0].args[0] != first.ExtensionID {
		t.Fatalf("unexpected live grants query: %#v", db.queries)
	}
}

func TestPostgresFrontendTrustStoreBulkRequestsAndDirectlyFinalizes(t *testing.T) {
	grant := frontendGrantFixture()
	requestedAt := grant.GrantedAt.Add(time.Hour)
	grant.RevocationRequestedAt = &requestedAt
	grant.RevocationRequestedByUserID = 7
	db := &fakeFrontendTrustDB{
		queryResults: []pgx.Rows{&frontendGrantRows{rows: []pgx.Row{frontendGrantTestRow(grant)}}},
		rows:         []pgx.Row{frontendGrantTestRow(grant)},
	}
	store := newPostgresFrontendTrustStore(db)

	items, err := store.RequestAllFrontendRevocations(context.Background(), 9)
	if err != nil {
		t.Fatalf("request all frontend revocations: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected bulk revocation result: %#v", items)
	}
	bulkSQL := compactSQL(db.queries[0].sql)
	for _, clause := range []string{
		"WHEN revocation_requested_at IS NULL THEN $1",
		"revocation_requested_at = COALESCE(revocation_requested_at, now())",
		"WHERE revoked_at IS NULL",
	} {
		if !strings.Contains(bulkSQL, clause) {
			t.Fatalf("bulk revocation SQL missing %q: %s", clause, bulkSQL)
		}
	}

	if _, err := store.FinalizeFrontendRevocation(context.Background(), FrontendFinalizeInput{
		ExtensionID: grant.ExtensionID, ExtensionVersion: grant.ExtensionVersion, PackageDigest: grant.PackageDigest, AdminFrontendDigest: grant.AdminFrontendDigest,
	}); err != nil {
		t.Fatalf("finalize frontend revocation: %v", err)
	}
	finalizeSQL := compactSQL(db.queries[1].sql)
	if !strings.Contains(finalizeSQL, "revocation_requested_at IS NOT NULL") || !strings.Contains(finalizeSQL, "revoked_at IS NULL") {
		t.Fatalf("direct finalize is not pending-state CAS: %s", finalizeSQL)
	}
}

func frontendGrantFixture() FrontendTrustGrant {
	return FrontendTrustGrant{
		ID:                  12,
		ExtensionID:         "fixture.plugin",
		ExtensionVersion:    "1.0.0",
		PackageDigest:       strings.Repeat("b", 64),
		AdminFrontendDigest: strings.Repeat("c", 64),
		APIVersion:          1,
		ContributionPoints:  []string{"admin.test.fixture"},
		ComponentIDs:        []string{"fixture.panel"},
		GrantedByUserID:     3,
		GrantedAt:           time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC),
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

func (row frontendTrustRowFunc) Scan(dest ...any) error {
	return row(dest...)
}

type frontendGrantRows struct {
	rows   []pgx.Row
	index  int
	closed bool
	err    error
}

func (rows *frontendGrantRows) Close()     { rows.closed = true }
func (rows *frontendGrantRows) Err() error { return rows.err }
func (rows *frontendGrantRows) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT 0")
}
func (rows *frontendGrantRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (rows *frontendGrantRows) Next() bool {
	if rows.index >= len(rows.rows) {
		return false
	}
	rows.index++
	return true
}
func (rows *frontendGrantRows) Scan(dest ...any) error {
	if rows.index < 1 || rows.index > len(rows.rows) {
		return errors.New("Scan called without Next")
	}
	return rows.rows[rows.index-1].Scan(dest...)
}
func (rows *frontendGrantRows) Values() ([]any, error) { return nil, errors.New("unexpected Values") }
func (rows *frontendGrantRows) RawValues() [][]byte    { return nil }
func (rows *frontendGrantRows) Conn() *pgx.Conn        { return nil }

func frontendTrustErrorRow(err error) pgx.Row {
	return frontendTrustRowFunc(func(...any) error { return err })
}

func frontendGrantTestRow(grant FrontendTrustGrant) pgx.Row {
	points, _ := json.Marshal(grant.ContributionPoints)
	components, _ := json.Marshal(grant.ComponentIDs)
	return frontendGrantRawJSONTestRow(grant, points, components)
}

func frontendGrantRawJSONTestRow(grant FrontendTrustGrant, points []byte, components []byte) pgx.Row {
	return frontendTrustRowFunc(func(dest ...any) error {
		if len(dest) != 14 {
			return errors.New("unexpected frontend grant scan width")
		}
		*dest[0].(*int64) = grant.ID
		*dest[1].(*string) = grant.ExtensionID
		*dest[2].(*string) = grant.ExtensionVersion
		*dest[3].(*string) = grant.PackageDigest
		*dest[4].(*string) = grant.AdminFrontendDigest
		*dest[5].(*int) = grant.APIVersion
		*dest[6].(*[]byte) = points
		*dest[7].(*[]byte) = components
		*dest[8].(*int64) = grant.GrantedByUserID
		*dest[9].(*time.Time) = grant.GrantedAt
		*dest[10].(**time.Time) = grant.RevocationRequestedAt
		*dest[11].(*int64) = grant.RevocationRequestedByUserID
		*dest[12].(**time.Time) = grant.RevokedAt
		*dest[13].(*int64) = grant.RevokedByUserID
		return nil
	})
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func compactSQL(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

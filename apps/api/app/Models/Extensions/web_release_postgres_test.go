package extensions

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPostgresWebReleaseStoreCreatePersistsChildrenAndInitialEventInOneTransaction(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	created := webReleaseFixture(42, 9, WebReleaseQueued, now)
	tx := newFakeWebReleaseTx()
	tx.rows = append(tx.rows, rowWithValues(releaseRowValues(created)))
	db := &fakeWebReleaseDB{tx: tx}
	store := newPostgresWebReleaseStore(db)
	actorID := int64(7)
	previousID := int64(41)
	compositionSnapshot := json.RawMessage(`{"webSource":"test-sha","sdkVersion":1}`)

	release, err := store.CreateWebRelease(context.Background(), WebReleaseCreateInput{
		TriggerKind:         "plugin.enable",
		TriggerExtensionID:  "demo.plugin",
		CompositionHash:     testCompositionHash(compositionSnapshot),
		CompositionSnapshot: compositionSnapshot,
		ActiveThemeID:       DefaultThemeID,
		ThemeVersion:        "1.0.0",
		ThemeLayerPath:      "layer",
		ThemePackageDigest:  strings.Repeat("b", 64),
		PreviousReleaseID:   &previousID,
		RequestedByUserID:   &actorID,
		Extensions: []WebReleaseExtensionInput{{
			ExtensionID:         "demo.plugin",
			ExtensionVersion:    "1.2.3",
			PackageDigest:       strings.Repeat("c", 64),
			AdminFrontendDigest: strings.Repeat("f", 64),
			FrontendRoot:        "frontend/admin",
			ComponentMap:        map[string]string{"job-cell": "components/JobCell.vue"},
			APIVersion:          1,
			TrustedComponents:   []ManifestContribution{{Point: "admin.test.fixture", ID: "job-cell"}},
			LocaleMap:           map[string]string{"en-US": "locales/en-US.json", "zh-CN": "locales/zh-CN.json"},
			LocaleMapDigest:     strings.Repeat("d", 64),
			LockfileDigest:      strings.Repeat("e", 64),
			SortOrder:           10,
		}},
		Effects: []WebReleaseEffectInput{{
			ExtensionID:    "demo.plugin",
			PreviousStatus: StatusDisabled,
			TargetStatus:   StatusEnabled,
		}},
		Reason:  "web_release.plugin_enable_queued",
		Message: "Queued trusted frontend release",
	})
	if err != nil {
		t.Fatalf("create web release: %v", err)
	}
	if release.ID != created.ID || !tx.committed {
		t.Fatalf("release was not committed: release=%#v committed=%t", release, tx.committed)
	}
	assertCallOrder(t, tx.calls,
		"query-row:insert into web_releases",
		"exec:insert into web_release_extensions",
		"exec:insert into web_release_extension_effects",
		"exec:insert into web_release_events",
	)
	extensionCall := findSQLCall(t, tx.calls, "insert into web_release_extensions")
	if string(extensionCall.args[6].([]byte)) != `{"job-cell":"components/JobCell.vue"}` {
		t.Fatalf("component map was not stored as deterministic JSON: %s", extensionCall.args[6])
	}
	if string(extensionCall.args[9].([]byte)) != `{"en-US":"locales/en-US.json","zh-CN":"locales/zh-CN.json"}` {
		t.Fatalf("locale map was not stored as deterministic JSON: %s", extensionCall.args[9])
	}
	releaseCall := findSQLCall(t, tx.calls, "insert into web_releases")
	if got := string(releaseCall.args[3].([]byte)); got != `{"sdkVersion":1,"webSource":"test-sha"}` {
		t.Fatalf("composition snapshot was not stored canonically: %s", got)
	}
	eventCall := findSQLCall(t, tx.calls, "insert into web_release_events")
	if eventCall.args[2] != nil || eventCall.args[3] != string(WebReleaseQueued) {
		t.Fatalf("unexpected initial event transition args: %#v", eventCall.args)
	}
}

func TestPostgresWebReleaseStoreCreateRollsBackWhenAChildFails(t *testing.T) {
	created := webReleaseFixture(42, 9, WebReleaseQueued, time.Now())
	tx := newFakeWebReleaseTx()
	tx.rows = append(tx.rows, rowWithValues(releaseRowValues(created)))
	tx.execResults = append(tx.execResults, fakeExecResult{err: errors.New("child insert failed")})
	store := newPostgresWebReleaseStore(&fakeWebReleaseDB{tx: tx})

	_, err := store.CreateWebRelease(context.Background(), WebReleaseCreateInput{
		TriggerKind:     "plugin.enable",
		CompositionHash: testCompositionHash(nil),
		ActiveThemeID:   DefaultThemeID,
		ThemeVersion:    "1.0.0",
		ThemeLayerPath:  "layer",
		Extensions: []WebReleaseExtensionInput{{
			ExtensionID:      "demo.plugin",
			ExtensionVersion: "1.0.0",
			PackageDigest:    strings.Repeat("b", 64),
			FrontendRoot:     "frontend/admin",
			APIVersion:       1,
			LocaleMapDigest:  strings.Repeat("c", 64),
			LockfileDigest:   strings.Repeat("d", 64),
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "child insert failed") {
		t.Fatalf("expected child insert failure, got %v", err)
	}
	if tx.committed || !tx.rolledBack {
		t.Fatalf("failed child must roll back whole transaction: committed=%t rolledBack=%t", tx.committed, tx.rolledBack)
	}
	if hasSQLCall(tx.calls, "insert into web_release_events") {
		t.Fatal("initial event must not be attempted after a child insert failure")
	}
}

func TestPostgresWebReleaseStoreCreateTxUsesCallerTransactionWithoutCommitting(t *testing.T) {
	created := webReleaseFixture(55, 12, WebReleaseQueued, time.Now())
	runner := newFakeWebReleaseTx()
	runner.rows = append(runner.rows, rowWithValues(releaseRowValues(created)))
	tx := &fakePGXWebReleaseTx{runner: runner}
	store := &PostgresWebReleaseStore{}

	release, err := store.CreateWebReleaseTx(context.Background(), tx, WebReleaseCreateInput{
		TriggerKind:     "rebuild",
		CompositionHash: testCompositionHash(nil),
		ActiveThemeID:   DefaultThemeID,
		ThemeVersion:    "1.0.0",
		ThemeLayerPath:  "layer",
	})
	if err != nil {
		t.Fatalf("create web release in caller transaction: %v", err)
	}
	if release.ID != created.ID {
		t.Fatalf("unexpected release: %#v", release)
	}
	if tx.committed || tx.rolledBack {
		t.Fatal("CreateWebReleaseTx must leave commit and rollback ownership with its caller")
	}
	assertCallOrder(t, runner.calls,
		"query-row:insert into web_releases",
		"exec:insert into web_release_events",
	)
}

func TestPostgresWebReleaseStoreRejectsCompositionHashMismatchBeforeWriting(t *testing.T) {
	tx := newFakeWebReleaseTx()
	store := newPostgresWebReleaseStore(&fakeWebReleaseDB{tx: tx})

	_, err := store.CreateWebRelease(context.Background(), WebReleaseCreateInput{
		TriggerKind:         "rebuild",
		CompositionHash:     strings.Repeat("a", 64),
		CompositionSnapshot: json.RawMessage(`{"webSource":"different"}`),
		ActiveThemeID:       DefaultThemeID,
		ThemeVersion:        "1.0.0",
		ThemeLayerPath:      "layer",
	})
	if !errors.Is(err, ErrWebReleaseCompositionMismatch) {
		t.Fatalf("expected composition mismatch, got %v", err)
	}
	if len(tx.calls) != 0 || tx.committed {
		t.Fatalf("composition mismatch performed writes: %#v", tx.calls)
	}
}

func TestPostgresWebReleaseStoreActivationReplacesOldActiveBeforeTarget(t *testing.T) {
	for _, test := range []struct {
		name         string
		compensation bool
		oldNext      WebReleaseStatus
		reason       string
	}{
		{name: "normal replacement", oldNext: WebReleaseInactive, reason: "web_release.replaced"},
		{name: "compensation replacement", compensation: true, oldNext: WebReleaseRolledBack, reason: "web_release.compensated"},
	} {
		t.Run(test.name, func(t *testing.T) {
			now := time.Now()
			oldActiveID := int64(21)
			active := webReleaseFixture(22, 11, WebReleaseActive, now)
			active.PreviousReleaseID = &oldActiveID
			tx := newFakeWebReleaseTx()
			tx.rows = append(tx.rows,
				rowWithValues([]any{string(WebReleaseActivating), sql.NullInt64{Int64: oldActiveID, Valid: true}}),
				rowWithValues([]any{oldActiveID}),
				rowWithValues(releaseRowValues(active)),
			)
			store := newPostgresWebReleaseStore(&fakeWebReleaseDB{tx: tx})

			release, err := store.TransitionWebRelease(context.Background(), WebReleaseTransitionInput{
				ID:             22,
				ExpectedStatus: WebReleaseActivating,
				NextStatus:     WebReleaseActive,
				Compensation:   test.compensation,
				Reason:         "web_release.activated",
			})
			if err != nil {
				t.Fatalf("activate web release: %v", err)
			}
			if release.Status != WebReleaseActive || !tx.committed {
				t.Fatalf("target release was not activated: %#v", release)
			}
			assertCallOrder(t, tx.calls,
				"query-row:select status, previous_release_id from web_releases",
				"query-row:select id from web_releases where status = 'active'",
				"exec:update web_releases set status = $2",
				"exec:insert into web_release_events",
				"query-row:update web_releases set status = $3",
				"exec:insert into web_release_events",
			)
			oldUpdate := firstSQLCall(t, tx.calls, "exec", "update web_releases set status = $2")
			if oldUpdate.args[1] != string(test.oldNext) || oldUpdate.args[2] != test.reason {
				t.Fatalf("unexpected old active transition args: %#v", oldUpdate.args)
			}
		})
	}
}

func TestPostgresWebReleaseStoreTransitionRejectsStaleExpectedStatus(t *testing.T) {
	tx := newFakeWebReleaseTx()
	tx.rows = append(tx.rows, rowWithValues([]any{string(WebReleaseBuilding), sql.NullInt64{}}))
	store := newPostgresWebReleaseStore(&fakeWebReleaseDB{tx: tx})

	_, err := store.TransitionWebRelease(context.Background(), WebReleaseTransitionInput{
		ID:             10,
		ExpectedStatus: WebReleaseVerifying,
		NextStatus:     WebReleaseReady,
	})
	if !errors.Is(err, ErrWebReleaseStale) {
		t.Fatalf("expected stale transition error, got %v", err)
	}
	if tx.committed || !tx.rolledBack {
		t.Fatalf("stale transition must roll back: committed=%t rolledBack=%t", tx.committed, tx.rolledBack)
	}
	if hasSQLCall(tx.calls, "update web_releases") || hasSQLCall(tx.calls, "insert into web_release_events") {
		t.Fatalf("stale transition performed writes: %#v", tx.calls)
	}
}

func TestPostgresWebReleaseStoreTransitionRollsBackWhenEventInsertFails(t *testing.T) {
	updated := webReleaseFixture(10, 5, WebReleaseVerifying, time.Now())
	tx := newFakeWebReleaseTx()
	tx.rows = append(tx.rows,
		rowWithValues([]any{string(WebReleaseBuilding), sql.NullInt64{}}),
		rowWithValues(releaseRowValues(updated)),
	)
	tx.execResults = append(tx.execResults, fakeExecResult{err: errors.New("event insert failed")})
	store := newPostgresWebReleaseStore(&fakeWebReleaseDB{tx: tx})

	_, err := store.TransitionWebRelease(context.Background(), WebReleaseTransitionInput{
		ID:             10,
		ExpectedStatus: WebReleaseBuilding,
		NextStatus:     WebReleaseVerifying,
		Reason:         "web_release.verifying",
	})
	if err == nil || !strings.Contains(err.Error(), "event insert failed") {
		t.Fatalf("expected event insert failure, got %v", err)
	}
	if tx.committed || !tx.rolledBack {
		t.Fatalf("event failure must roll back status update: committed=%t rolledBack=%t", tx.committed, tx.rolledBack)
	}
	assertCallOrder(t, tx.calls,
		"query-row:select status, previous_release_id from web_releases",
		"query-row:update web_releases set status = $3",
		"exec:insert into web_release_events",
	)
}

func TestPostgresWebReleaseStoreSameStateTransitionIsReadOnly(t *testing.T) {
	failed := webReleaseFixture(10, 5, WebReleaseFailed, time.Now())
	tx := newFakeWebReleaseTx()
	tx.rows = append(tx.rows,
		rowWithValues([]any{string(WebReleaseFailed), sql.NullInt64{}}),
		rowWithValues(releaseRowValues(failed)),
	)
	store := newPostgresWebReleaseStore(&fakeWebReleaseDB{tx: tx})

	release, err := store.TransitionWebRelease(context.Background(), WebReleaseTransitionInput{
		ID:             failed.ID,
		ExpectedStatus: WebReleaseFailed,
		NextStatus:     WebReleaseFailed,
		PublicMessage:  "must not overwrite immutable failure",
		Reason:         "must.not.append",
	})
	if err != nil {
		t.Fatalf("repeat terminal transition: %v", err)
	}
	if release.PublicMessage != failed.PublicMessage {
		t.Fatalf("same-state transition mutated immutable release: %#v", release)
	}
	if hasSQLCall(tx.calls, "update web_releases") || hasSQLCall(tx.calls, "insert into web_release_events") {
		t.Fatalf("same-state transition performed writes: %#v", tx.calls)
	}
}

func TestPostgresWebReleaseStoreActivationRejectsUnexpectedActiveRelease(t *testing.T) {
	expectedID := int64(20)
	tx := newFakeWebReleaseTx()
	tx.rows = append(tx.rows,
		rowWithValues([]any{string(WebReleaseActivating), sql.NullInt64{Int64: expectedID, Valid: true}}),
		rowWithValues([]any{int64(21)}),
	)
	store := newPostgresWebReleaseStore(&fakeWebReleaseDB{tx: tx})

	_, err := store.TransitionWebRelease(context.Background(), WebReleaseTransitionInput{
		ID:             22,
		ExpectedStatus: WebReleaseActivating,
		NextStatus:     WebReleaseActive,
	})
	if !errors.Is(err, ErrWebReleaseStale) {
		t.Fatalf("expected stale active release error, got %v", err)
	}
	if hasSQLCall(tx.calls, "update web_releases") || hasSQLCall(tx.calls, "insert into web_release_events") {
		t.Fatalf("stale activation performed writes: %#v", tx.calls)
	}
}

func TestLatestProgressWebReleaseForExtensionHidesStaleFailedAfterActive(t *testing.T) {
	now := time.Date(2026, 7, 12, 4, 0, 0, 0, time.UTC)
	// 模拟插件列表查询：最新相关发布已是 active（更早 failed 不应再挂到行上）。
	active := webReleaseFixture(12, 20, WebReleaseActive, now)
	active.TriggerExtensionID = "sforum.smtp"
	db := newFakeWebReleaseDB()
	db.rows = append(db.rows, rowWithValues(releaseRowValues(active)))
	store := newPostgresWebReleaseStore(db)

	_, err := store.LatestProgressWebReleaseForExtension(context.Background(), "sforum.smtp")
	if !errors.Is(err, ErrWebReleaseNotFound) {
		t.Fatalf("expected active latest release to hide progress row, got %v", err)
	}
	call := firstSQLCall(t, db.calls, "query-row", "from web_releases wr")
	normalized := normalizeSQL(call.query)
	if !strings.Contains(normalized, "'active'") || !strings.Contains(normalized, "'failed'") {
		t.Fatalf("progress lookup must consider active and failed: %s", call.query)
	}
}

func TestLatestProgressWebReleaseForExtensionReturnsLatestFailed(t *testing.T) {
	now := time.Date(2026, 7, 12, 4, 0, 0, 0, time.UTC)
	failed := webReleaseFixture(11, 19, WebReleaseFailed, now)
	failed.TriggerExtensionID = "sforum.smtp"
	failed.PublicMessage = "Web release build failed."
	db := newFakeWebReleaseDB()
	db.rows = append(db.rows, rowWithValues(releaseRowValues(failed)))
	store := newPostgresWebReleaseStore(db)

	release, err := store.LatestProgressWebReleaseForExtension(context.Background(), "sforum.smtp")
	if err != nil {
		t.Fatalf("load progress release: %v", err)
	}
	if release.ID != 11 || release.Status != WebReleaseFailed {
		t.Fatalf("unexpected progress release: %#v", release)
	}
}

func TestPostgresWebReleaseStoreDependencySnapshotIsOneTimeCAS(t *testing.T) {
	input := WebReleaseDependencySnapshotInput{
		WebReleaseID: 9,
		ExtensionID:  "demo.plugin",
		ResolvedDependencies: []Dependency{
			{Name: "zeta", Version: "2.0.0", Integrity: "sha512-z"},
			{Name: "alpha", Version: "1.0.0", Integrity: "sha512-a"},
		},
		Digest: strings.Repeat("a", 64),
	}

	t.Run("first write and identical retry", func(t *testing.T) {
		db := newFakeWebReleaseDB()
		db.execResults = append(db.execResults, fakeExecResult{affected: 1})
		store := newPostgresWebReleaseStore(db)
		if err := store.RecordWebReleaseDependencySnapshot(context.Background(), input); err != nil {
			t.Fatalf("record dependency snapshot: %v", err)
		}
		call := findSQLCall(t, db.calls, "update web_release_extensions")
		normalized := normalizeSQL(call.query)
		if !strings.Contains(normalized, "resolved_dependency_snapshot_digest = ''") ||
			!strings.Contains(normalized, "release.status in ('resolving', 'installing', 'building')") {
			t.Fatalf("snapshot update is not compare-and-set: %s", call.query)
		}
		if string(call.args[2].([]byte)) != `[{"name":"alpha","version":"1.0.0","integrity":"sha512-a"},{"name":"zeta","version":"2.0.0","integrity":"sha512-z"}]` {
			t.Fatalf("dependencies were not canonically sorted: %s", call.args[2])
		}
	})

	t.Run("different second write conflicts", func(t *testing.T) {
		db := newFakeWebReleaseDB()
		db.execResults = append(db.execResults, fakeExecResult{affected: 0})
		db.rows = append(db.rows, rowWithValues([]any{strings.Repeat("b", 64), string(WebReleaseBuilding)}))
		store := newPostgresWebReleaseStore(db)
		err := store.RecordWebReleaseDependencySnapshot(context.Background(), input)
		if !errors.Is(err, ErrWebReleaseDependencySnapshotConflict) {
			t.Fatalf("expected dependency snapshot conflict, got %v", err)
		}
	})

	t.Run("terminal release rejects a late first write", func(t *testing.T) {
		db := newFakeWebReleaseDB()
		db.execResults = append(db.execResults, fakeExecResult{affected: 0})
		db.rows = append(db.rows, rowWithValues([]any{"", string(WebReleaseFailed)}))
		err := newPostgresWebReleaseStore(db).RecordWebReleaseDependencySnapshot(context.Background(), input)
		if !errors.Is(err, ErrWebReleaseDependencySnapshotConflict) {
			t.Fatalf("expected terminal dependency snapshot conflict, got %v", err)
		}
	})
}

func TestPostgresWebReleaseStoreListUsesBoundedStablePagination(t *testing.T) {
	for _, test := range []struct {
		name       string
		input      WebReleaseListInput
		wantPage   int
		wantPer    int
		wantOffset int
	}{
		{name: "recommended defaults", wantPage: 1, wantPer: 20, wantOffset: 0},
		{name: "maximum page size", input: WebReleaseListInput{Page: 3, PerPage: 500}, wantPage: 3, wantPer: 100, wantOffset: 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := newFakeWebReleaseDB()
			db.rows = append(db.rows, rowWithValues([]any{int64(0)}))
			db.queryResults = append(db.queryResults, &fakeWebReleaseRows{})
			store := newPostgresWebReleaseStore(db)

			page, err := store.ListWebReleases(context.Background(), test.input)
			if err != nil {
				t.Fatalf("list web releases: %v", err)
			}
			if page.Page != test.wantPage || page.PerPage != test.wantPer {
				t.Fatalf("unexpected normalized page: %#v", page)
			}
			listCall := firstSQLCall(t, db.calls, "query", "order by desired_generation desc, id desc")
			if listCall.args[1] != test.wantPer || listCall.args[2] != test.wantOffset {
				t.Fatalf("unexpected pagination args: %#v", listCall.args)
			}
		})
	}
}

func TestPostgresWebReleaseStoreDetailQueriesChildrenInDeterministicOrder(t *testing.T) {
	release := webReleaseFixture(33, 8, WebReleaseReady, time.Now())
	tx := newFakeWebReleaseTx()
	tx.rows = append(tx.rows, rowWithValues(releaseRowValues(release)))
	tx.queryResults = append(tx.queryResults,
		&fakeWebReleaseRows{},
		&fakeWebReleaseRows{},
		&fakeWebReleaseRows{},
	)
	db := &fakeWebReleaseDB{tx: tx}
	store := newPostgresWebReleaseStore(db)

	detail, err := store.WebRelease(context.Background(), release.ID)
	if err != nil {
		t.Fatalf("load web release detail: %v", err)
	}
	if detail.ID != release.ID {
		t.Fatalf("unexpected release detail: %#v", detail)
	}
	if !tx.committed {
		t.Fatal("detail read transaction was not committed")
	}
	firstSQLCall(t, tx.calls, "query-row", "for share")
	firstSQLCall(t, tx.calls, "query", "order by sort_order, extension_id")
	firstSQLCall(t, tx.calls, "query", "order by extension_id")
	firstSQLCall(t, tx.calls, "query", "order by created_at, id")
}

type fakeSQLCall struct {
	kind  string
	query string
	args  []any
}

type fakeExecResult struct {
	affected int64
	err      error
}

type fakeWebReleaseSQL struct {
	calls        []fakeSQLCall
	rows         []webReleaseRow
	queryResults []webReleaseRows
	execResults  []fakeExecResult
}

func (f *fakeWebReleaseSQL) Exec(_ context.Context, query string, args ...any) (int64, error) {
	f.calls = append(f.calls, fakeSQLCall{kind: "exec", query: query, args: args})
	if len(f.execResults) == 0 {
		return 1, nil
	}
	result := f.execResults[0]
	f.execResults = f.execResults[1:]
	return result.affected, result.err
}

func (f *fakeWebReleaseSQL) Query(_ context.Context, query string, args ...any) (webReleaseRows, error) {
	f.calls = append(f.calls, fakeSQLCall{kind: "query", query: query, args: args})
	if len(f.queryResults) == 0 {
		return nil, errors.New("unexpected Query call")
	}
	rows := f.queryResults[0]
	f.queryResults = f.queryResults[1:]
	return rows, nil
}

func (f *fakeWebReleaseSQL) QueryRow(_ context.Context, query string, args ...any) webReleaseRow {
	f.calls = append(f.calls, fakeSQLCall{kind: "query-row", query: query, args: args})
	if len(f.rows) == 0 {
		return fakeWebReleaseRow{err: errors.New("unexpected QueryRow call")}
	}
	row := f.rows[0]
	f.rows = f.rows[1:]
	return row
}

type fakeWebReleaseDB struct {
	*fakeWebReleaseSQL
	tx       *fakeWebReleaseTx
	beginErr error
}

func newFakeWebReleaseDB() *fakeWebReleaseDB {
	return &fakeWebReleaseDB{fakeWebReleaseSQL: &fakeWebReleaseSQL{}}
}

func (f *fakeWebReleaseDB) Begin(context.Context) (webReleaseTransaction, error) {
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	if f.tx == nil {
		f.tx = newFakeWebReleaseTx()
	}
	return f.tx, nil
}

type fakeWebReleaseTx struct {
	*fakeWebReleaseSQL
	committed  bool
	rolledBack bool
}

func newFakeWebReleaseTx() *fakeWebReleaseTx {
	return &fakeWebReleaseTx{fakeWebReleaseSQL: &fakeWebReleaseSQL{}}
}

func (f *fakeWebReleaseTx) Commit(context.Context) error {
	f.committed = true
	return nil
}

func (f *fakeWebReleaseTx) Rollback(context.Context) error {
	f.rolledBack = true
	return nil
}

type fakeWebReleaseRow struct {
	values []any
	err    error
}

func rowWithValues(values []any) fakeWebReleaseRow {
	return fakeWebReleaseRow{values: values}
}

func (f fakeWebReleaseRow) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(dest) != len(f.values) {
		return fmt.Errorf("scan destination mismatch: got %d want %d", len(dest), len(f.values))
	}
	for index := range dest {
		if err := assignFakeScanValue(dest[index], f.values[index]); err != nil {
			return fmt.Errorf("scan index %d: %w", index, err)
		}
	}
	return nil
}

type fakeWebReleaseRows struct {
	values [][]any
	index  int
	err    error
	closed bool
}

func (f *fakeWebReleaseRows) Close() {
	f.closed = true
}

func (f *fakeWebReleaseRows) Err() error {
	return f.err
}

func (f *fakeWebReleaseRows) Next() bool {
	if f.index >= len(f.values) {
		f.closed = true
		return false
	}
	f.index++
	return true
}

func (f *fakeWebReleaseRows) Scan(dest ...any) error {
	if f.index < 1 || f.index > len(f.values) {
		return errors.New("Scan called without Next")
	}
	return (fakeWebReleaseRow{values: f.values[f.index-1]}).Scan(dest...)
}

func (f *fakeWebReleaseRows) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT 0")
}

func (f *fakeWebReleaseRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (f *fakeWebReleaseRows) Values() ([]any, error) {
	if f.index < 1 || f.index > len(f.values) {
		return nil, errors.New("Values called without Next")
	}
	return f.values[f.index-1], nil
}

func (f *fakeWebReleaseRows) RawValues() [][]byte {
	return nil
}

func (f *fakeWebReleaseRows) Conn() *pgx.Conn {
	return nil
}

type fakePGXWebReleaseTx struct {
	runner     *fakeWebReleaseTx
	committed  bool
	rolledBack bool
}

func (f *fakePGXWebReleaseTx) Begin(context.Context) (pgx.Tx, error) {
	return f, nil
}

func (f *fakePGXWebReleaseTx) Commit(context.Context) error {
	f.committed = true
	return nil
}

func (f *fakePGXWebReleaseTx) Rollback(context.Context) error {
	f.rolledBack = true
	return nil
}

func (f *fakePGXWebReleaseTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("unexpected CopyFrom")
}

func (f *fakePGXWebReleaseTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return nil
}

func (f *fakePGXWebReleaseTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (f *fakePGXWebReleaseTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("unexpected Prepare")
}

func (f *fakePGXWebReleaseTx) Exec(ctx context.Context, query string, arguments ...any) (pgconn.CommandTag, error) {
	affected, err := f.runner.Exec(ctx, query, arguments...)
	return pgconn.NewCommandTag(fmt.Sprintf("UPDATE %d", affected)), err
}

func (f *fakePGXWebReleaseTx) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	rows, err := f.runner.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return rows.(*fakeWebReleaseRows), nil
}

func (f *fakePGXWebReleaseTx) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return f.runner.QueryRow(ctx, query, args...)
}

func (f *fakePGXWebReleaseTx) Conn() *pgx.Conn {
	return nil
}

func webReleaseFixture(id int64, generation int64, status WebReleaseStatus, now time.Time) WebRelease {
	compositionSnapshot := json.RawMessage(`{}`)
	return WebRelease{
		ID:                   id,
		DesiredGeneration:    generation,
		TriggerKind:          "test",
		CompositionHash:      testCompositionHash(compositionSnapshot),
		CompositionSnapshot:  compositionSnapshot,
		ActiveThemeID:        DefaultThemeID,
		ThemeVersion:         "1.0.0",
		ThemeLayerPath:       "layer",
		Status:               status,
		ActivationCheckpoint: WebReleaseCheckpointPending,
		ReloadMode:           WebReleaseReloadPrompt,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func testCompositionHash(snapshot json.RawMessage) string {
	canonical, err := canonicalJSONObject(snapshot)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(canonical))
}

func releaseRowValues(release WebRelease) []any {
	return []any{
		release.ID,
		release.DesiredGeneration,
		release.TriggerKind,
		release.TriggerExtensionID,
		release.CompositionHash,
		release.CompositionSnapshot,
		release.ActiveThemeID,
		release.ThemeVersion,
		release.ThemeLayerPath,
		release.ThemePackageDigest,
		string(release.Status),
		release.ActivationCheckpoint,
		release.ReloadMode,
		release.ArtifactPath,
		release.ArtifactDigest,
		release.ServerEntry,
		release.BuildLog,
		release.PublicReason,
		release.PublicMessage,
		nullInt64Value(release.PreviousReleaseID),
		nullInt64Value(release.RequestedByUserID),
		nullInt64Value(release.ActivatedByUserID),
		release.CreatedAt,
		release.UpdatedAt,
		nullTimeValue(release.ReadyAt),
		nullTimeValue(release.ActivationStartedAt),
		nullTimeValue(release.ActivatedAt),
		nullTimeValue(release.CompletedAt),
	}
}

func nullInt64Value(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func nullTimeValue(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func assignFakeScanValue(destination any, value any) error {
	destinationValue := reflect.ValueOf(destination)
	if destinationValue.Kind() != reflect.Pointer || destinationValue.IsNil() {
		return errors.New("destination is not a pointer")
	}
	target := destinationValue.Elem()
	if value == nil {
		target.Set(reflect.Zero(target.Type()))
		return nil
	}
	source := reflect.ValueOf(value)
	if !source.Type().AssignableTo(target.Type()) {
		return fmt.Errorf("cannot assign %s to %s", source.Type(), target.Type())
	}
	target.Set(source)
	return nil
}

func normalizeSQL(query string) string {
	return strings.ToLower(strings.Join(strings.Fields(query), " "))
}

func assertCallOrder(t *testing.T, calls []fakeSQLCall, expected ...string) {
	t.Helper()
	if len(calls) != len(expected) {
		t.Fatalf("unexpected call count: got=%d want=%d calls=%#v", len(calls), len(expected), calls)
	}
	for index, fragment := range expected {
		actual := calls[index].kind + ":" + normalizeSQL(calls[index].query)
		if !strings.Contains(actual, fragment) {
			t.Fatalf("call %d mismatch: want fragment %q, got %q", index, fragment, actual)
		}
	}
}

func hasSQLCall(calls []fakeSQLCall, fragment string) bool {
	fragment = strings.ToLower(fragment)
	for _, call := range calls {
		if strings.Contains(normalizeSQL(call.query), fragment) {
			return true
		}
	}
	return false
}

func findSQLCall(t *testing.T, calls []fakeSQLCall, fragment string) fakeSQLCall {
	t.Helper()
	return firstSQLCall(t, calls, "", fragment)
}

func firstSQLCall(t *testing.T, calls []fakeSQLCall, kind string, fragment string) fakeSQLCall {
	t.Helper()
	fragment = strings.ToLower(fragment)
	for _, call := range calls {
		if (kind == "" || call.kind == kind) && strings.Contains(normalizeSQL(call.query), fragment) {
			return call
		}
	}
	t.Fatalf("SQL call %s containing %q was not found in %#v", kind, fragment, calls)
	return fakeSQLCall{}
}

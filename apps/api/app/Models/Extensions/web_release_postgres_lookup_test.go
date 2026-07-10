package extensions

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPostgresWebReleaseStoreActiveLookupUsesCallerTransaction(t *testing.T) {
	active := webReleaseFixture(9, 4, WebReleaseActive, time.Now())
	runner := newFakeWebReleaseTx()
	runner.rows = append(runner.rows, rowWithValues(releaseRowValues(active)))
	tx := &fakePGXWebReleaseTx{runner: runner}
	store := &PostgresWebReleaseStore{}

	result, err := store.ActiveWebReleaseTx(context.Background(), tx)
	if err != nil {
		t.Fatalf("load active web release: %v", err)
	}
	if result.ID != active.ID || tx.committed || tx.rolledBack {
		t.Fatalf("active lookup changed caller transaction: result=%#v tx=%#v", result, tx)
	}
	firstSQLCall(t, runner.calls, "query-row", "where status = 'active'")
}

func TestPostgresWebReleaseStoreActiveLookupMapsMissingRelease(t *testing.T) {
	runner := newFakeWebReleaseTx()
	runner.rows = append(runner.rows, fakeWebReleaseRow{err: errors.New("sentinel")})
	store := &PostgresWebReleaseStore{}
	_, err := store.ActiveWebReleaseTx(context.Background(), &fakePGXWebReleaseTx{runner: runner})
	if err == nil || errors.Is(err, ErrWebReleaseNotFound) {
		t.Fatalf("non-pgx error was not preserved: %v", err)
	}
}

func TestPostgresWebReleaseStoreListsLiveIntentEffectsAfterClosingReleaseRows(t *testing.T) {
	first := webReleaseFixture(12, 6, WebReleaseBuilding, time.Now())
	second := webReleaseFixture(11, 5, WebReleaseReady, time.Now())
	releaseRows := &fakeWebReleaseRows{values: [][]any{releaseRowValues(first), releaseRowValues(second)}}
	runner := newFakeWebReleaseTx()
	runner.queryResults = append(runner.queryResults,
		releaseRows,
		&fakeWebReleaseRows{},
		&fakeWebReleaseRows{},
	)
	store := &PostgresWebReleaseStore{}

	items, err := store.LiveWebReleasesByCompositionTx(
		context.Background(),
		&fakePGXWebReleaseTx{runner: runner},
		first.CompositionHash,
	)
	if err != nil {
		t.Fatalf("list live releases by composition: %v", err)
	}
	if !releaseRows.closed || len(items) != 2 || items[0].ID != first.ID || items[1].ID != second.ID {
		t.Fatalf("unexpected live intent results: closed=%t items=%#v", releaseRows.closed, items)
	}
	firstSQLCall(t, runner.calls, "query", "order by desired_generation desc, id desc")
	if countSQLCalls(runner.calls, "order by extension_id") != 2 {
		t.Fatalf("expected effects for each live release, calls=%#v", runner.calls)
	}
}

func countSQLCalls(calls []fakeSQLCall, fragment string) int {
	count := 0
	for _, call := range calls {
		if containsNormalizedSQL(call.query, fragment) {
			count++
		}
	}
	return count
}

func containsNormalizedSQL(query string, fragment string) bool {
	return len(fragment) == 0 || hasSQLCall([]fakeSQLCall{{query: query}}, fragment)
}

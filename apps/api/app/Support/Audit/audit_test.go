package audit

import "testing"

func TestEnsureNoop(t *testing.T) {
	w := Ensure(nil)
	if err := w.Append(nil, Event{Action: ActionSettingsUpdate}); err != nil {
		t.Fatalf("noop append: %v", err)
	}
	idWriter, ok := w.(IDWriter)
	if !ok {
		t.Fatal("noop writer does not preserve the optional IDWriter contract")
	}
	if id, err := idWriter.AppendReturningID(nil, Event{Action: ActionSettingsUpdate}); err != nil || id != 0 {
		t.Fatalf("noop durable append id=%d err=%v", id, err)
	}
}

func TestPostgresWriterReturningIDFailsClosedWhenUnconfigured(t *testing.T) {
	var writer *PostgresWriter
	if id, err := writer.AppendReturningID(nil, Event{Action: ActionSettingsUpdate}); err == nil || id != 0 {
		t.Fatalf("nil writer durable append id=%d err=%v", id, err)
	}
	writer = &PostgresWriter{}
	if id, err := writer.AppendReturningID(nil, Event{Action: ActionSettingsUpdate}); err == nil || id != 0 {
		t.Fatalf("unconfigured writer durable append id=%d err=%v", id, err)
	}
}

func TestRecommendedRetentionDays(t *testing.T) {
	if RecommendedRetentionDays < 30 {
		t.Fatalf("retention too short: %d", RecommendedRetentionDays)
	}
}

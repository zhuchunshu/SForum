package audit

import "testing"

func TestEnsureNoop(t *testing.T) {
	w := Ensure(nil)
	if err := w.Append(nil, Event{Action: ActionSettingsUpdate}); err != nil {
		t.Fatalf("noop append: %v", err)
	}
}

func TestRecommendedRetentionDays(t *testing.T) {
	if RecommendedRetentionDays < 30 {
		t.Fatalf("retention too short: %d", RecommendedRetentionDays)
	}
}

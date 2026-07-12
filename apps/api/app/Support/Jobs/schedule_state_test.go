package jobs

import "testing"

func TestParseScheduleEnabled(t *testing.T) {
	if !ParseScheduleEnabled("", false) {
		t.Fatal("missing should default enabled")
	}
	if !ParseScheduleEnabled("true", true) {
		t.Fatal("true should enable")
	}
	if ParseScheduleEnabled("false", true) {
		t.Fatal("false should disable")
	}
	if ParseScheduleEnabled("0", true) {
		t.Fatal("0 should disable")
	}
	if !ParseScheduleEnabled("weird", true) {
		t.Fatal("unknown should stay enabled")
	}
}

func TestScheduleEnabledOptionName(t *testing.T) {
	got := ScheduleEnabledOptionName("identity.cleanup_sessions")
	want := "jobs.schedule.identity.cleanup_sessions.enabled"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestValidateScheduleID(t *testing.T) {
	if err := ValidateScheduleID("identity.cleanup_sessions"); err != nil {
		t.Fatalf("valid: %v", err)
	}
	if err := ValidateScheduleID(""); err == nil {
		t.Fatal("expected empty error")
	}
	if err := ValidateScheduleID("bad id"); err == nil {
		t.Fatal("expected whitespace error")
	}
}

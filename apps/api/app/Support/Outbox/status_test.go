package outbox

import "testing"

func TestNormalizeAliases(t *testing.T) {
	cases := map[string]string{
		"queued":    StatusQueued,
		"RUNNING":   StatusSending,
		"succeeded": StatusSent,
		"failed":    StatusFailed,
		"  Skipped ": StatusSkipped,
		"dead":      StatusDead,
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Fatalf("Normalize(%q)=%q want %q", in, got, want)
		}
	}
}

func TestIsTerminalAndReplay(t *testing.T) {
	if !IsTerminal(StatusSent) || !IsTerminal("succeeded") {
		t.Fatal("sent/succeeded should be terminal")
	}
	if IsTerminal(StatusQueued) || IsTerminal(StatusSending) {
		t.Fatal("queued/sending should not be terminal")
	}
	if !CanReplay(StatusFailed) || !CanReplay(StatusDead) {
		t.Fatal("failed/dead should be replayable")
	}
	if CanReplay(StatusSkipped) || CanReplay(StatusSent) {
		t.Fatal("skipped/sent should not be replayable by default")
	}
}

func TestCanTransition(t *testing.T) {
	if !CanTransition("", StatusQueued) {
		t.Fatal("new record may enter queued")
	}
	if !CanTransition(StatusQueued, StatusSending) {
		t.Fatal("queued→sending")
	}
	if !CanTransition(StatusSending, StatusSent) {
		t.Fatal("sending→sent")
	}
	if !CanTransition(StatusSending, StatusFailed) {
		t.Fatal("sending→failed")
	}
	if !CanTransition(StatusFailed, StatusQueued) {
		t.Fatal("failed→queued replay")
	}
	if CanTransition(StatusSent, StatusQueued) {
		t.Fatal("sent should not replay")
	}
	if CanTransition(StatusSkipped, StatusSending) {
		t.Fatal("skipped should not re-enter sending")
	}
}

func TestExhaustedAttempts(t *testing.T) {
	if ExhaustedAttempts(3, 0) {
		t.Fatal("maxAttempts 0 disables budget")
	}
	if !ExhaustedAttempts(5, 5) {
		t.Fatal("attempt==max should exhaust")
	}
	if ExhaustedAttempts(4, 5) {
		t.Fatal("under budget")
	}
}

func TestShouldSetCompletedAt(t *testing.T) {
	if !ShouldSetCompletedAt(StatusFailed) || ShouldSetCompletedAt(StatusSending) {
		t.Fatal("completed_at only on terminal")
	}
}

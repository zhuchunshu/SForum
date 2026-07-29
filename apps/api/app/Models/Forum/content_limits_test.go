package forum

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidateTopicTitleAndContentLimits(t *testing.T) {
	settings := defaultForumSettings()
	if err := validateTopicTitle("好", settings); !errors.Is(err, ErrTitleTooShort) {
		t.Fatalf("expected title too short, got %v", err)
	}
	if err := validateTopicTitle(strings.Repeat("标", settings.TopicTitleMaxRunes+1), settings); !errors.Is(err, ErrTitleTooLong) {
		t.Fatalf("expected title too long, got %v", err)
	}
	if err := validateTopicTitle("正常标题", settings); err != nil {
		t.Fatalf("expected valid title, got %v", err)
	}

	settings.TopicContentMinRunes = 5
	if err := validateTopicContent("短", settings); !errors.Is(err, ErrContentTooShort) {
		t.Fatalf("expected content too short, got %v", err)
	}
	settings.TopicContentMaxRunes = 10
	if err := validateTopicContent(strings.Repeat("正", 11), settings); !errors.Is(err, ErrContentTooLong) {
		t.Fatalf("expected content too long, got %v", err)
	}
}

func TestValidateCommentNestingAndTagMin(t *testing.T) {
	settings := defaultForumSettings()
	settings.CommentMaxNestingDepth = 1
	if err := validateCommentNesting(1, settings); !errors.Is(err, ErrCommentNestingDeep) {
		t.Fatalf("expected nesting error, got %v", err)
	}
	if err := validateCommentNesting(0, settings); err != nil {
		t.Fatalf("depth 1 should be allowed, got %v", err)
	}

	settings.TagMinPerTopic = 2
	settings.TagMaxPerTopic = 5
	if err := validateTagCount(1, settings); !errors.Is(err, ErrTagMinRequired) {
		t.Fatalf("expected tag min required, got %v", err)
	}
	if err := validateTagCount(2, settings); err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
}

func TestEditWindowAndCooldownHelpers(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	created := now.Add(-30 * time.Minute)
	if withinEditWindow(created, 0, now) != true {
		t.Fatal("0 window should be unlimited")
	}
	if withinEditWindow(created, 10, now) {
		t.Fatal("expired window should fail")
	}
	if !withinEditWindow(created, 60, now) {
		t.Fatal("open window should pass")
	}
	if cooldownElapsed(created, true, 0, now) != true {
		t.Fatal("0 cooldown always elapsed")
	}
	if cooldownElapsed(now.Add(-10*time.Second), true, 30, now) {
		t.Fatal("cooldown should still apply")
	}
	if !cooldownElapsed(now.Add(-40*time.Second), true, 30, now) {
		t.Fatal("cooldown should have elapsed")
	}
}

func TestCooldownErrorPreservesCauseAndRetryAt(t *testing.T) {
	lastAt := time.Date(2026, 7, 30, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	err := newCooldownError(ErrCommentCooldown, lastAt, 45)
	if !errors.Is(err, ErrCommentCooldown) {
		t.Fatalf("expected comment cooldown cause, got %v", err)
	}
	var cooldown *CooldownError
	if !errors.As(err, &cooldown) {
		t.Fatalf("expected CooldownError, got %T", err)
	}
	want := time.Date(2026, 7, 30, 4, 0, 45, 0, time.UTC)
	if !cooldown.RetryAt.Equal(want) || cooldown.RetryAt.Location() != time.UTC {
		t.Fatalf("retryAt = %s, want UTC %s", cooldown.RetryAt, want)
	}
}

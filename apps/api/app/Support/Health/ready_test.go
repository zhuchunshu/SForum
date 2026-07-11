package health

import (
	"context"
	"errors"
	"testing"
)

func TestEvaluateAllOK(t *testing.T) {
	report := Evaluate(context.Background(), []Checker{
		FuncChecker{ComponentName: "postgres", IsRequired: true, Fn: func(context.Context) error { return nil }},
		FuncChecker{ComponentName: "redis", IsRequired: false, Fn: func(context.Context) error { return nil }},
	})
	if !report.Ready || report.Status != "ready" {
		t.Fatalf("expected ready, got %#v", report)
	}
	if len(report.Components) != 2 {
		t.Fatalf("components=%d", len(report.Components))
	}
}

func TestEvaluateRequiredFailureNotReady(t *testing.T) {
	report := Evaluate(context.Background(), []Checker{
		FuncChecker{ComponentName: "postgres", IsRequired: true, Fn: func(context.Context) error {
			return errors.New("down")
		}},
		FuncChecker{ComponentName: "redis", IsRequired: false, Fn: func(context.Context) error { return nil }},
	})
	if report.Ready || report.Status != "not_ready" {
		t.Fatalf("expected not_ready, got %#v", report)
	}
	if report.Components[0].Status != StatusError || report.Components[0].Error == "" {
		t.Fatalf("postgres result: %#v", report.Components[0])
	}
}

func TestEvaluateOptionalFailureDegradedReady(t *testing.T) {
	// F1 默认：Meili/Redis 失败仍 ready（degraded）。
	report := Evaluate(context.Background(), []Checker{
		FuncChecker{ComponentName: "postgres", IsRequired: true, Fn: func(context.Context) error { return nil }},
		FuncChecker{ComponentName: "meilisearch", IsRequired: false, Fn: func(context.Context) error {
			return errors.New("meili down")
		}},
	})
	if !report.Ready {
		t.Fatalf("expected ready despite meili failure, got %#v", report)
	}
	if report.Status != StatusDegraded {
		t.Fatalf("expected degraded status, got %q", report.Status)
	}
}

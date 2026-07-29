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
	// F1 默认：可选依赖失败仍 ready（degraded）。
	report := Evaluate(context.Background(), []Checker{
		FuncChecker{ComponentName: "postgres", IsRequired: true, Fn: func(context.Context) error { return nil }},
		FuncChecker{ComponentName: "redis", IsRequired: false, Fn: func(context.Context) error {
			return errors.New("redis down")
		}},
	})
	if !report.Ready {
		t.Fatalf("expected ready despite optional failure, got %#v", report)
	}
	if report.Status != StatusDegraded {
		t.Fatalf("expected degraded status, got %q", report.Status)
	}
}

func TestApplyRecoveryRequirementForcesNotReady(t *testing.T) {
	requirement := &RecoveryRequirement{
		Code: PluginRuntimeRecoveryCode, Component: "plugin_runtime", Message: "stage failed",
		PublicationRevision: 935,
		Artifacts:           []RecoveryArtifact{{ExtensionID: "sforum.auth-github", Version: "1.0.0", Digest: "digest"}},
	}
	report := ApplyRecoveryRequirement(Evaluate(context.Background(), nil), requirement)
	if report.Ready || report.Status != "not_ready" || report.Recovery != requirement {
		t.Fatalf("recovery readiness = %#v", report)
	}
	if len(report.Components) != 1 || report.Components[0].Name != "plugin_runtime" ||
		!report.Components[0].Required || report.Components[0].Status != StatusError {
		t.Fatalf("recovery component = %#v", report.Components)
	}
}

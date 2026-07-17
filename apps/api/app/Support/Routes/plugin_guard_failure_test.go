package routes

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestPluginGuardFailureCarriesOnlyStableHostEvidence(t *testing.T) {
	tests := []struct {
		kind     PluginGuardFailureKind
		observed bool
		compat   error
	}{
		{PluginGuardFailureDenied, true, ErrCoreGuardPermissionDenied},
		{PluginGuardFailureUnavailable, false, ErrCoreGuardEvaluatorUnavailable},
		{PluginGuardFailureCrash, true, ErrCoreGuardEvaluatorUnavailable},
		{PluginGuardFailureTimeout, true, context.DeadlineExceeded},
		{PluginGuardFailureProtocol, true, ErrCoreGuardEvaluatorUnavailable},
		{PluginGuardFailureCanceled, true, context.Canceled},
	}
	for _, test := range tests {
		t.Run(string(test.kind), func(t *testing.T) {
			failure := NewPluginGuardFailure(test.kind, test.observed)
			wrapped := fmt.Errorf("guard evaluation: %w", failure)
			var evidence *PluginGuardFailure
			if !errors.As(wrapped, &evidence) || evidence.Kind() != test.kind ||
				evidence.RuntimeExecutionObserved() != test.observed || !errors.Is(wrapped, test.compat) {
				t.Fatalf("failure=%#v wrapped=%v", evidence, wrapped)
			}
			for _, secret := range []string{"plugin reason", "runtime-instance", strings.Repeat("a", 64)} {
				if strings.Contains(wrapped.Error(), secret) {
					t.Fatalf("stable error disclosed %q: %s", secret, wrapped)
				}
			}
		})
	}

	failureType := reflect.TypeOf(PluginGuardFailure{})
	if failureType.NumField() != 2 || failureType.Field(0).Name != "kind" ||
		failureType.Field(1).Name != "runtimeExecutionObserved" {
		t.Fatalf("plugin guard evidence gained identity or payload fields: %#v", failureType)
	}
}

func TestPluginGuardFailurePreservesUnavailableCompatibility(t *testing.T) {
	for _, kind := range []PluginGuardFailureKind{
		PluginGuardFailureUnavailable,
		PluginGuardFailureCrash,
		PluginGuardFailureTimeout,
		PluginGuardFailureProtocol,
		PluginGuardFailureCanceled,
	} {
		if err := NewPluginGuardFailure(kind, kind != PluginGuardFailureUnavailable); !errors.Is(err, ErrCoreGuardEvaluatorUnavailable) {
			t.Fatalf("kind %q lost evaluator-unavailable compatibility: %v", kind, err)
		}
	}
}

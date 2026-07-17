package routes

import (
	"context"
	"errors"
)

type PluginGuardFailureKind string

const (
	PluginGuardFailureDenied      PluginGuardFailureKind = "denied"
	PluginGuardFailureUnavailable PluginGuardFailureKind = "unavailable"
	PluginGuardFailureCrash       PluginGuardFailureKind = "crash"
	PluginGuardFailureTimeout     PluginGuardFailureKind = "timeout"
	PluginGuardFailureProtocol    PluginGuardFailureKind = "protocol"
	PluginGuardFailureCanceled    PluginGuardFailureKind = "canceled"
)

// PluginGuardFailure carries only Host-observed classification. Exact route,
// artifact, stage, and raw plugin error data remain owned by the Dispatcher.
type PluginGuardFailure struct {
	kind                     PluginGuardFailureKind
	runtimeExecutionObserved bool
}

func NewPluginGuardFailure(kind PluginGuardFailureKind, runtimeExecutionObserved bool) *PluginGuardFailure {
	return &PluginGuardFailure{kind: kind, runtimeExecutionObserved: runtimeExecutionObserved}
}

func (e *PluginGuardFailure) Kind() PluginGuardFailureKind {
	if e == nil {
		return ""
	}
	return e.kind
}

func (e *PluginGuardFailure) RuntimeExecutionObserved() bool {
	return e != nil && e.runtimeExecutionObserved
}

func (e *PluginGuardFailure) Error() string {
	if e == nil {
		return ""
	}
	switch e.kind {
	case PluginGuardFailureDenied:
		return "routes: plugin guard denied the request"
	case PluginGuardFailureUnavailable:
		return "routes: plugin guard is unavailable"
	case PluginGuardFailureCrash:
		return "routes: plugin guard runtime failed"
	case PluginGuardFailureTimeout:
		return "routes: plugin guard timed out"
	case PluginGuardFailureProtocol:
		return "routes: plugin guard returned an invalid response"
	case PluginGuardFailureCanceled:
		return "routes: plugin guard was canceled"
	default:
		return "routes: plugin guard failed"
	}
}

func (e *PluginGuardFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	switch e.kind {
	case PluginGuardFailureDenied:
		return ErrCoreGuardPermissionDenied
	case PluginGuardFailureTimeout:
		return errors.Join(ErrCoreGuardEvaluatorUnavailable, context.DeadlineExceeded)
	case PluginGuardFailureCanceled:
		return errors.Join(ErrCoreGuardEvaluatorUnavailable, context.Canceled)
	default:
		return ErrCoreGuardEvaluatorUnavailable
	}
}

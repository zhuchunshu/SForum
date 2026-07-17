package routes

import (
	"context"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

// RouteStreamFailureClass is Host-observed provenance only. Raw transport
// errors, request metadata, and payload bytes must never enter durable evidence.
type RouteStreamFailureClass string

const (
	RouteStreamFailureRuntimeTransport RouteStreamFailureClass = "runtime_transport"
	RouteStreamFailureHostBudget       RouteStreamFailureClass = "host_budget"
	RouteStreamFailureInvalidPreflight RouteStreamFailureClass = "invalid_preflight"
	RouteStreamFailureMissingTerminal  RouteStreamFailureClass = "missing_terminal"
)

// RouteStreamFailure pins one stream incident to the exact route snapshot and
// executable artifact. It deliberately contains no raw error or request data.
type RouteStreamFailure struct {
	Revision                 uint64
	StepIndex                int
	Phase                    RouteExecutionPhase
	InvocationStage          InvocationStage `json:"invocationStage"`
	Action                   string
	Mode                     string
	RouteID                  string
	ContractVersion          string
	Method                   string
	PathSignature            string
	FailureCode              RouteFailureCode
	CauseClass               RouteStreamFailureClass
	RuntimeExecutionObserved bool
	ActorID                  int64
	ResponseStatus           int
	CommitState              RouteExecutionCommitState
	Artifact                 PluginArtifact
}

func ValidRouteStreamFailureClass(class RouteStreamFailureClass) bool {
	switch class {
	case RouteStreamFailureRuntimeTransport, RouteStreamFailureHostBudget,
		RouteStreamFailureInvalidPreflight, RouteStreamFailureMissingTerminal:
		return true
	default:
		return false
	}
}

func ValidRouteStreamFailure(event RouteStreamFailure) bool {
	validMode := event.Mode == extensionmanifest.RouteModeMultipart ||
		event.Mode == extensionmanifest.RouteModeStream || event.Mode == extensionmanifest.RouteModeSSE ||
		event.Mode == extensionmanifest.RouteModeWebSocket
	return event.Revision > 0 && event.StepIndex >= 0 &&
		event.Phase == RoutePhaseHandler && event.InvocationStage == InvocationStageHandler &&
		(event.Action == extensionmanifest.RouteActionAdd || event.Action == extensionmanifest.RouteActionReplace) &&
		validMode && event.FailureCode == RouteFailureTransportFailed &&
		ValidRouteStreamFailureClass(event.CauseClass) && event.RuntimeExecutionObserved &&
		(event.CommitState == RouteCommitSideEffectStarted || event.CommitState == RouteCommitResponseStarted) &&
		event.RouteID != "" && event.ContractVersion != "" && event.Method != "" && event.PathSignature != "" &&
		event.Artifact.ExtensionID != "" && event.Artifact.ExtensionVersion != "" &&
		event.Artifact.PackageDigest != "" && event.Artifact.RuntimeInstanceID != ""
}

// RouteStreamFailureSink must return promptly. Production first persists
// payload-free evidence, then closes exact local admission and resolves the
// local quarantine result.
type RouteStreamFailureSink interface {
	RecordStreamFailure(context.Context, RouteStreamFailure)
}

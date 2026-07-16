package routes

import "context"

type RouteFailureCode string

const (
	RouteFailureGuardDenied            RouteFailureCode = "guard_denied"
	RouteFailureRequestSchemaRejected  RouteFailureCode = "request_schema_rejected"
	RouteFailureTransportFailed        RouteFailureCode = "transport_failed"
	RouteFailureResponseSchemaRejected RouteFailureCode = "response_schema_rejected"
)

// RouteCommittedAfterFailure contains only stable execution identity. Request
// headers, query, body, raw errors, and response payloads must never enter it.
type RouteCommittedAfterFailure struct {
	Revision                 uint64
	StepIndex                int
	Phase                    RouteExecutionPhase
	Action                   string
	RouteID                  string
	ContractVersion          string
	Method                   string
	PathSignature            string
	FailureCode              RouteFailureCode
	RuntimeExecutionObserved bool
	ActorID                  int64
	ResponseStatus           int
	CommitState              RouteExecutionCommitState
	Artifact                 PluginArtifact
}

// RouteFailureSink must return promptly. Production implementations close exact
// admission synchronously and enqueue only bounded audit work.
type RouteFailureSink interface {
	RecordCommittedAfterFailure(context.Context, RouteCommittedAfterFailure)
}

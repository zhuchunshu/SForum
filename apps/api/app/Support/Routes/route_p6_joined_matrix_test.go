package routes

import "testing"

// TestP6JoinedRouteMatrix is the Routes-side named gate for the P6 plan-book
// Tests row. It composes existing Registry/Dispatcher matrices and stream
// lifetime / guard-failure suites without a parallel harness.
//
// Pair with apps/api/app/Http TestP6JoinedBehaviorMatrix for the full joined gate.
func TestP6JoinedRouteMatrix(t *testing.T) {
	t.Run("action_terminals", TestP6RouteActionMatrixTerminals)
	t.Run("modifier_and_global_chain", TestP6RouteModifierAndGlobalChainSelection)
	t.Run("same_priority_order_permutations", TestP6RouteSamePriorityOrderingPermutations)
	t.Run("priority_tie_break", TestP6RoutePriorityTieBreakOrder)
	t.Run("priority_and_conflict_selection", TestP6RoutePriorityAndConflictSelection)
	t.Run("safe_mode_bypass", TestP6RouteSafeModeBypassesPluginContributions)
	t.Run("unsafe_no_second_writer", TestP6RouteUnsafeNoSecondWriterMatrix)
	t.Run("timeout_where_harness_supports", TestP6RouteTimeoutWhereHarnessSupports)
	t.Run("plugin_guard_request_failure_matrix", TestDispatcherPluginGuardRequestFailureMatrix)
	t.Run("plugin_guard_request_caller_cancel_no_incident", TestDispatcherPluginGuardRequestCallerCancellationDoesNotRecordIncident)
	t.Run("plugin_guard_unsafe_response_failure_matrix", TestDispatcherPluginGuardUnsafeResponseFailureMatrix)
	t.Run("plugin_guard_unsafe_response_caller_cancel_no_incident", TestDispatcherPluginGuardUnsafeResponseCallerCancellationDoesNotRecordIncident)
	t.Run("plugin_guard_replay_reauthorization", TestDispatcherPluginGuardReplayReauthorizationFailureClassification)
	t.Run("stream_dispatcher_guard_failures", TestStreamDispatcherClassifiesPluginGuardFailures)
	t.Run("stream_default_budget_zero_timeout", TestStreamDispatcherUsesDefaultBudgetWhenTimeoutIsZero)
	t.Run("stream_host_budget_covers_lifecycle", TestStreamDispatcherHostBudgetCoversGuardPreflightOpenAndStream)
	t.Run("stream_host_budget_timeout", TestStreamDispatcherHostBudgetTimeoutFailsClosed)
	t.Run("stream_detach_caller", TestStreamLifetimeDetachCallerStopsRequestCancel)
	t.Run("stream_inner_completion_releases_resources", TestStreamLifetimeInnerCompletionReleasesResourcesBeforeAdapterCancel)
	t.Run("stream_exact_host_cancel_cause", TestStreamLifetimePropagatesExactHostCauseToInner)
	t.Run("stream_caller_cancel_before_open", TestStreamLifetimeCallerCancelBeforeOpenHasNoInvoker)
	t.Run("stream_terminal_wins_over_cancel", TestStreamLifetimeTerminalWinsOverConcurrentCancel)
	t.Run("stream_outer_preserves_inner_cause", TestStreamLifetimeOuterDoesNotEraseInnerTypedCause)
	t.Run("stream_recv_does_not_close_done_before_cancel", TestStreamLifetimeRecvDoesNotCloseDoneBeforeCancel)
	t.Run("stream_raw_request_stamp", TestStreamDispatcherPreservesAuthorizedRawRequestStamp)
	t.Run("stream_caller_cancel_no_failure_evidence", TestStreamDispatcherCallerCancellationHasNoFailureEvidence)
	t.Run("stream_observed_cancel_fails_closed", TestStreamDispatcherCallerCancellationAfterObservedExecutionFailsClosed)
}

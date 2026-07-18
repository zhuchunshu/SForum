package http

import "testing"

// TestP6JoinedBehaviorMatrix is the Http-side named gate for the P6 plan-book
// Tests row. It composes existing production-path suites instead of inventing a
// parallel harness. Routes-side action/priority/conflict cells live in
// Support/Routes TestP6JoinedRouteMatrix.
//
// Run (with Routes sibling):
//
//	go test ./app/Http -run '^TestP6JoinedBehaviorMatrix$' -count=1
//	go test ./app/Support/Routes -run '^TestP6JoinedRouteMatrix$' -count=1
//	source ../../.env
//	test -n "$DATABASE_URL"
//	SFORUM_TEST_DATABASE_URL="$DATABASE_URL" go test -v ./app/Http -run '^TestP6JoinedPostgresBehaviorMatrix$' -count=1
//
// Race/count discipline matches stream and custom/raw production gates.
func TestP6JoinedBehaviorMatrix(t *testing.T) {
	t.Run("permission_csrf_locale_query_body", TestP6RoutePermissionCSRFLocaleQueryAndBodyMatrix)
	t.Run("custom_guard_fake_runtime", TestP6RouteCustomGuardOwnsPolicyAndFailsClosed)
	t.Run("raw_request_trust_and_credentials", TestP6RouteRawRequestDeclarationRequiresExactTrustAndForwardsCredentials)
	t.Run("protocol_v2_crash_and_timeout", TestP6RouteProtocolV2CrashAndTimeoutMatrix)
	t.Run("custom_raw_production_chain", TestRouteCustomAndRawGuardsAcrossFiberManagerAndRealProtocolV2Process)
	t.Run("legacy_host_guard_cannot_mint_raw", TestRouteHostRouteGuardAuthorizerCannotMintRawOnFiber)
	t.Run("websocket_custom_guard_open_only", TestRouteWebSocketCustomGuardRunsOnlyAtOpenPreflight)
	t.Run("stream_multipart_sse_websocket_disconnect", TestRouteStreamAcrossFiberManagerAndRealProtocolV2Process)
	t.Run("websocket_invalid_preflight_fail_before_cancel", TestRouteDispatcherInvalidWebSocketPreflightFailsBeforeLifetimeDone)
	t.Run("sse_fiber_commit", TestRouteDispatcherStreamsSSEThroughFiberAndCommitsTrace)
	t.Run("websocket_bridge_and_disconnect", TestRouteDispatcherBridgesWebSocketMessagesAndDisconnect)
	t.Run("stream_request_adapter_provenance", TestPumpRouteStreamRequestClassifiesCallerAndRuntimeFailures)
	t.Run("stream_response_runtime_incident", TestStreamRouteResponseCancelsOnRuntimeFailure)
	t.Run("stream_response_missing_terminal", TestStreamRouteResponseClassifiesMissingTerminal)
	t.Run("stream_host_writer_zero_incidents", TestStreamRouteResponseHostWriterFailuresDoNotRecordIncidents)
	t.Run("sse_invalid_preflight_incident", TestRouteDispatcherClassifiesInvalidSSEPreflight)
	t.Run("stream_terminal_trace_before_done", TestStreamRouteResponsePublishesCommitTraceBeforeLifetimeDone)
	t.Run("stream_force_cancel_before_lease_release", TestRouteV2StreamSessionCapturesForceCancelCauseBeforeLeaseRelease)
	t.Run("stream_host_winner_over_wire_teardown", TestRouteV2StreamSessionHostWinnerOverridesWireTeardownFailure)
	t.Run("websocket_runtime_failure_dispositions", TestRouteWebSocketRuntimeFailureDispositionMatrix)
	t.Run("websocket_client_zero_incidents", TestRouteWebSocketClientFailuresDoNotRecordIncidents)
	t.Run("websocket_response_pump_dispositions", TestPumpWebSocketResponsesFailureDisposition)
	t.Run("websocket_detach_disposition", TestClassifyRouteWebSocketDetachFailure)
	t.Run("websocket_grace_timeout_incident", TestAwaitRouteWebSocketTerminalClassifiesGraceTimeout)
	t.Run("websocket_queued_terminal_arbitration", TestAwaitRouteWebSocketTerminalPrefersQueuedRuntimeAndResponse)
	t.Run("websocket_normal_terminal_zero_incidents", TestRouteDispatcherCommitsWebSocketAfterRequestCloseAndResponseTerminal)
	t.Run("stream_producer_to_recorder", TestRouteStreamRuntimeIncidentReachesRecorder)
	t.Run("stream_recorder_persist_quarantine_resolve", TestRouteFailureRecorderPersistsStreamIncidentBeforeQuarantineAndResolution)
}

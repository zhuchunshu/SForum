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
}

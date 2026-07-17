package routes

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRouteFailureInvocationStageIsRequiredAndJSONStable(t *testing.T) {
	for _, stage := range []InvocationStage{InvocationStageRequest, InvocationStageHandler, InvocationStageResponse} {
		if !ValidInvocationStage(stage) {
			t.Fatalf("valid invocation stage %q rejected", stage)
		}
	}
	for _, stage := range []InvocationStage{"", "forged"} {
		if ValidInvocationStage(stage) {
			t.Fatalf("invalid invocation stage %q accepted", stage)
		}
	}

	body, err := json.Marshal(RouteCommittedAfterFailure{InvocationStage: InvocationStageResponse})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"invocationStage":"response"`) {
		t.Fatalf("failure JSON omits invocation stage: %s", body)
	}

	valid := []struct {
		phase  RouteExecutionPhase
		action string
		stage  InvocationStage
	}{
		{RoutePhaseGlobal, "global_middleware", InvocationStageRequest},
		{RoutePhaseBefore, "before", InvocationStageRequest},
		{RoutePhaseFilter, "filter", InvocationStageRequest},
		{RoutePhaseFilter, "filter", InvocationStageResponse},
		{RoutePhaseWrap, "wrap", InvocationStageRequest},
		{RoutePhaseWrap, "wrap", InvocationStageResponse},
		{RoutePhaseHandler, "add", InvocationStageHandler},
		{RoutePhaseAfter, "after", InvocationStageResponse},
	}
	for _, item := range valid {
		if !ValidInvocationStageForStep(item.phase, item.action, item.stage) {
			t.Fatalf("valid stage tuple rejected: %#v", item)
		}
	}
	for _, item := range []struct {
		phase  RouteExecutionPhase
		action string
		stage  InvocationStage
	}{
		{RoutePhaseHandler, "add", InvocationStageRequest},
		{RoutePhaseBefore, "before", InvocationStageResponse},
		{RoutePhaseAfter, "after", InvocationStageHandler},
		{RoutePhaseAfter, "wrap", InvocationStageResponse},
	} {
		if ValidInvocationStageForStep(item.phase, item.action, item.stage) {
			t.Fatalf("invalid stage tuple accepted: %#v", item)
		}
	}
}

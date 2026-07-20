package main

import (
	"context"
	"errors"
	"log"
	"strings"

	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

// P13 commerce-workflow reference: hooks + services + jobs for a trusted vertical.

const (
	orderEvaluateHookID = "sforum.commerce-workflow.hook.order-evaluate"
	orderCreatedHookID  = "sforum.commerce-workflow.hook.order-created"
	orderEvaluateName   = "sforum.commerce-workflow.order.evaluate"
	orderCreatedName    = "sforum.commerce-workflow.order.created"
	orderEvaluateInput  = "sforum.commerce-workflow.hook.order-evaluate.input@1"
	orderEvaluateResult = "sforum.commerce-workflow.hook.order-evaluate.result@1"
	orderEvaluatePatch  = "sforum.commerce-workflow.hook.order-evaluate.result.patch@1"
	orderCreatedInput   = "sforum.commerce-workflow.hook.order-created.input@1"

	serviceID       = "sforum.commerce-workflow.service.orders"
	serviceVersion  = "1.0.0"
	serviceRequest  = "sforum.commerce-workflow.service.orders.request@1"
	serviceResponse = "sforum.commerce-workflow.service.orders.response@1"

	jobID      = "sforum.commerce-workflow.job.settle"
	jobKind    = "sforum.commerce-workflow.settle"
	jobHandler = "sforum.commerce-workflow.job.settle"
	jobPayload = "sforum.commerce-workflow.job.settle.payload@1"
)

func main() {
	hooks, err := pluginv2.NewHookRegistry(
		pluginv2.HookDefinition{
			ID: orderEvaluateHookID, Name: orderEvaluateName, Kind: "filter",
			ContractVersion: orderEvaluateHookID + "@1",
			Handler:         "sforum.commerce-workflow.hook.order-evaluate",
			InputSchema:     orderEvaluateInput, ResultSchema: orderEvaluateResult,
			Execution: "sync", FailurePolicy: "fail_closed", TimeoutMS: 1000,
			MutableFields: []string{"status"}, Execute: evaluateOrder,
		},
		pluginv2.HookDefinition{
			ID: orderCreatedHookID, Name: orderCreatedName, Kind: "observe",
			ContractVersion: orderCreatedHookID + "@1",
			Handler:         "sforum.commerce-workflow.hook.order-created",
			InputSchema:     orderCreatedInput,
			Execution:       "async", FailurePolicy: "fail_open", TimeoutMS: 2000,
			Execute: observeOrderCreated,
		},
	)
	if err != nil {
		log.Fatalf("configure commerce hooks: %v", err)
	}
	services, err := pluginv2.NewServiceRegistry(pluginv2.ServiceDefinition{
		ServiceID: serviceID, Version: serviceVersion,
		RequestSchemaID: serviceRequest, ResponseSchemaID: serviceResponse,
		Operations: []pluginv2.ServiceOperation{{Name: "lookup", Unary: lookupOrder}},
	})
	if err != nil {
		log.Fatalf("configure commerce services: %v", err)
	}
	jobs, err := pluginv2.NewJobRegistry(pluginv2.JobDefinition{
		ID: jobID, ContractVersion: jobID + "@1", Name: jobKind,
		Handler: jobHandler, PayloadSchema: jobPayload,
		RetryPolicy: "exponential", MaxAttempts: 5, ConcurrencyLimit: 1,
		Execute: settleOrderJob,
	})
	if err != nil {
		log.Fatalf("configure commerce jobs: %v", err)
	}
	pluginv2.Serve(pluginv2.NewServer().
		WithHookRegistry(hooks).
		WithServiceRegistry(services).
		WithJobRegistry(jobs),
	)
}

func evaluateOrder(ctx context.Context, call *pluginv2.HookCall) (*pluginv2.HookResult, error) {
	values := pluginv2.TypedDocumentValues(call.Payload)
	status, _ := values["status"].(string)
	// 测试触发：status=fail 失败；status=timeout 等待取消。
	switch strings.TrimSpace(status) {
	case "fail":
		return nil, errors.New("reference commerce evaluate failure")
	case "timeout":
		<-ctx.Done()
		return nil, context.Cause(ctx)
	}
	result, err := pluginv2.NewTypedDocument(orderEvaluateResult, map[string]any{
		"accepted": true,
		"status":   "approved",
	})
	if err != nil {
		return nil, err
	}
	patch, err := pluginv2.NewTypedDocument(orderEvaluatePatch, map[string]any{
		"status": "approved",
	})
	if err != nil {
		return nil, err
	}
	return &pluginv2.HookResult{Accepted: true, Result: result, Patch: patch}, nil
}

func observeOrderCreated(_ context.Context, call *pluginv2.HookCall) (*pluginv2.HookResult, error) {
	// observe 只确认投递；不返回 patch。
	_ = pluginv2.TypedDocumentValues(call.Payload)
	return &pluginv2.HookResult{Accepted: true}, nil
}

func lookupOrder(_ context.Context, call *pluginv2.ServiceCall) (*protocolwire.TypedDocument, error) {
	values := pluginv2.TypedDocumentValues(call.Input)
	orderID, _ := values["orderId"].(string)
	if strings.TrimSpace(orderID) == "" {
		return nil, &pluginv2.ServiceError{
			Code: protocolwire.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Reason:  "commerce.order_id_required",
			Message: "orderId is required",
		}
	}
	if orderID == "fail" {
		return nil, errors.New("reference commerce service failure")
	}
	return pluginv2.NewTypedDocument(serviceResponse, map[string]any{
		"orderId": orderID,
		"status":  "open",
		"total":   "19.90",
	})
}

func settleOrderJob(ctx context.Context, call *pluginv2.JobCall) error {
	if call == nil || call.Progress == nil {
		return errors.New("missing job progress stream")
	}
	values := pluginv2.TypedDocumentValues(call.Payload)
	if orderID, _ := values["orderId"].(string); orderID == "fail" {
		return errors.New("reference commerce settle failure")
	}
	if err := call.Progress.Send(&protocolwire.ProgressUpdate{
		StepId: call.JobID, State: protocolwire.ProgressState_PROGRESS_STATE_RUNNING,
		CompletedUnits: 1, TotalUnits: 2, Checkpoint: "settling",
	}); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
	}
	return call.Progress.Send(&protocolwire.ProgressUpdate{
		StepId: call.JobID, State: protocolwire.ProgressState_PROGRESS_STATE_SUCCEEDED,
		CompletedUnits: 2, TotalUnits: 2, Checkpoint: "settled",
	})
}

package main

import (
	"context"
	"errors"
	"log"
	"strings"

	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

// P13 media-optimize reference: MIME/transform publication + background job.
// Transform execution remains Host-final; failure mode is fallback_original.

const (
	jobID      = "sforum.media-optimize.job.variants"
	jobKind    = "sforum.media-optimize.variants"
	jobHandler = "sforum.media-optimize.job.variants"
	jobPayload = "sforum.media-optimize.job.variants.payload@1"
)

func main() {
	jobs, err := pluginv2.NewJobRegistry(pluginv2.JobDefinition{
		ID: jobID, ContractVersion: jobID + "@1", Name: jobKind,
		Handler: jobHandler, PayloadSchema: jobPayload,
		RetryPolicy: "bounded", MaxAttempts: 3, RetryDelaySeconds: 5, ConcurrencyLimit: 2,
		Execute: runOptimizeJob,
	})
	if err != nil {
		log.Fatalf("configure media-optimize jobs: %v", err)
	}
	pluginv2.Serve(pluginv2.NewServer().WithJobRegistry(jobs))
}

func runOptimizeJob(ctx context.Context, call *pluginv2.JobCall) error {
	if call == nil || call.Progress == nil {
		return errors.New("missing job progress stream")
	}
	values := pluginv2.TypedDocumentValues(call.Payload)
	source, _ := values["sourceDigest"].(string)
	// 测试触发：payload 标记 fail/timeout 时失败，Host 应保留 original。
	switch strings.TrimSpace(source) {
	case "reference:fail":
		return errors.New("reference media optimize failure")
	case "reference:timeout":
		<-ctx.Done()
		return context.Cause(ctx)
	}
	if err := call.Progress.Send(&protocolwire.ProgressUpdate{
		StepId: call.JobID, State: protocolwire.ProgressState_PROGRESS_STATE_RUNNING,
		CompletedUnits: 1, TotalUnits: 2, Checkpoint: "variants",
	}); err != nil {
		return err
	}
	return call.Progress.Send(&protocolwire.ProgressUpdate{
		StepId: call.JobID, State: protocolwire.ProgressState_PROGRESS_STATE_SUCCEEDED,
		CompletedUnits: 2, TotalUnits: 2, Checkpoint: "done",
	})
}

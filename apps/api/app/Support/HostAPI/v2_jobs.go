package hostapi

import (
	"context"
	"fmt"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	hostv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/host/v2"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/grpc"
)

type protocolV2JobServer struct {
	hostv2.UnimplementedJobServiceServer
	core *protocolV2Core
}

func (s *protocolV2JobServer) Enqueue(ctx context.Context, request *hostv2.JobEnqueueRequest) (*hostv2.JobEnqueueResponse, error) {
	response := &hostv2.JobEnqueueResponse{Context: protocolV2ResponseContext(request.GetContext())}
	if request.GetIdempotencyKey() != "" || request.GetDelay().AsDuration() != 0 {
		response.Error = protocolV2Unsupported("host.job_options_unsupported", "The v1 compatibility queue cannot honor idempotency keys or delayed delivery.")
		return response, nil
	}
	if request.GetPayload() == nil || request.GetPayload().GetSchemaId() == "" || request.GetPayloadVersion() == "" {
		response.Error = &protocolv2.ErrorDetail{
			Code: protocolv2.ErrorCode_ERROR_CODE_INVALID_ARGUMENT, Reason: "host.job_payload_contract_required", Message: "Job payload schema id and version are required.",
		}
		return response, nil
	}
	if request.GetPayload().GetSchemaVersion() != request.GetPayloadVersion() {
		response.Error = protocolV2Unsupported("host.job_payload_version_mismatch", "Job payload versions in the request and typed document must match.")
		return response, nil
	}
	result := s.enqueueVersioned(ctx, request)
	if !result.OK {
		response.Error = protocolV2Failure(result.Reason, result.Message)
	}
	return response, nil
}

func (s *protocolV2JobServer) enqueueVersioned(ctx context.Context, request *hostv2.JobEnqueueRequest) Response {
	if s == nil || s.core == nil || s.core.service == nil {
		return fail("host.unavailable", "Host API is not configured.")
	}
	identity := request.GetContext().GetExtension()
	extensionID := identity.GetExtensionId()
	caps, err := s.core.service.loadCaps(ctx, extensionID)
	if err != nil {
		return fail("host.extension_unavailable", err.Error())
	}
	if err := caps.Require(capabilities.JobsEnqueue); err != nil {
		return denied(capabilities.JobsEnqueue)
	}
	source, ok := s.core.service.capabilities.(PluginJobContractSource)
	if !ok {
		return fail("host.job_contract_unavailable", "The authoritative plugin job contract source is not configured.")
	}
	contract, err := source.PluginJobContract(ctx, extensionID, request.GetJobKind())
	if err != nil {
		return fail("host.job_contract_failed", err.Error())
	}
	if contract.ExtensionVersion != identity.GetExtensionVersion() || contract.ArtifactDigest != identity.GetArtifactDigest() {
		return fail("host.job_runtime_stale", "The runtime artifact no longer matches the installed plugin job contract.")
	}
	if contract.PayloadSchemaID != request.GetPayload().GetSchemaId() || contract.PayloadSchemaVersion != request.GetPayloadVersion() {
		return fail("host.job_payload_contract_mismatch", "The payload schema does not match the declared plugin job contract.")
	}
	enqueuer, ok := s.core.service.jobs.(VersionedPluginJobEnqueuer)
	if !ok {
		return fail("host.job_versioned_queue_unavailable", "The versioned plugin job queue is not configured.")
	}
	if err := enqueuer.EnqueueVersionedPluginJob(
		ctx, contract, identity.GetTrustGrantId(), protocolV2DocumentValues(request.GetPayload()),
	); err != nil {
		return fail("host.enqueue_failed", fmt.Sprintf("enqueue versioned plugin job: %v", err))
	}
	return success(map[string]any{
		"kind": request.GetJobKind(), "contractVersion": contract.JobContract,
		"payloadSchemaId": contract.PayloadSchemaID, "payloadVersion": contract.PayloadSchemaVersion,
		"enqueued": true,
	})
}

func (s *protocolV2JobServer) Cancel(_ context.Context, request *hostv2.JobCancelRequest) (*hostv2.JobCancelResponse, error) {
	return &hostv2.JobCancelResponse{
		Context: protocolV2ResponseContext(request.GetContext()),
		Error:   protocolV2Unsupported("host.job_cancel_unavailable", "Job cancellation has no v1 compatibility adapter."),
	}, nil
}

func (s *protocolV2JobServer) Watch(request *hostv2.JobWatchRequest, stream grpc.ServerStreamingServer[protocolv2.ProgressUpdate]) error {
	return stream.Send(&protocolv2.ProgressUpdate{
		Context: protocolV2ResponseContext(request.GetContext()),
		Error:   protocolV2Unsupported("host.job_watch_unavailable", "Job progress has no v1 compatibility adapter."),
	})
}

package hostapi

import (
	"context"

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
	result := s.core.call(ctx, request.GetContext(), MethodEnqueueOwnJob, map[string]any{
		"kind": request.GetJobKind(), "payload": protocolV2DocumentValues(request.GetPayload()),
	})
	if !result.OK {
		response.Error = protocolV2Failure(result.Reason, result.Message)
	}
	return response, nil
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

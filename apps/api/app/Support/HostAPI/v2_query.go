package hostapi

import (
	"context"

	hostv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/host/v2"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/grpc"
)

type protocolV2QueryServer struct {
	hostv2.UnimplementedHostQueryServiceServer
	core *protocolV2Core
}

func (s *protocolV2QueryServer) Execute(ctx context.Context, request *hostv2.QueryRequest) (*hostv2.QueryResponse, error) {
	if request.GetQueryId() != QueryOwnSettingsID {
		if isStableProtocolV2QueryID(request.GetQueryId()) {
			if s == nil || s.core == nil || s.core.queries == nil {
				response := &hostv2.QueryResponse{Context: protocolV2ResponseContext(request.GetContext()), Page: &protocolv2.PageInfo{}}
				response.Error = queryError(protocolv2.ErrorCode_ERROR_CODE_UNAVAILABLE, "host.query_backend_unavailable", "Stable Host Queries are not configured.", true)
				return response, nil
			}
			return s.core.queries.execute(ctx, request), nil
		}
		response := &hostv2.QueryResponse{Context: protocolV2ResponseContext(request.GetContext())}
		response.Error = protocolV2Unsupported("host.query_unsupported", "The query id or plan version is not supported.")
		return response, nil
	}
	response := &hostv2.QueryResponse{Context: protocolV2ResponseContext(request.GetContext())}
	if request.GetPlanVersion() != QueryOwnSettingsVersion {
		response.Error = protocolV2Unsupported("host.query_unsupported", "The query id or plan version is not supported.")
		return response, nil
	}
	if (request.GetResultSchemaId() != "" && request.GetResultSchemaId() != QueryOwnSettingsSchemaID) ||
		(request.GetResultSchemaVersion() != "" && request.GetResultSchemaVersion() != QueryOwnSettingsSchemaV1) {
		response.Error = protocolV2Unsupported("host.query_schema_mismatch", "The requested result schema is not supported.")
		return response, nil
	}
	if len(request.GetFilters()) > 0 || len(request.GetSorts()) > 0 || request.GetPage().GetCursor() != "" {
		response.Error = protocolV2Unsupported("host.query_shape_unsupported", "Own-settings compatibility query does not support filters, sorts, or cursors.")
		return response, nil
	}
	result := s.core.call(ctx, request.GetContext(), MethodGetSettings, nil)
	if !result.OK {
		response.Error = protocolV2Failure(result.Reason, result.Message)
		return response, nil
	}
	settings, _ := result.Data["settings"].(map[string]any)
	if len(request.GetFields()) > 0 {
		filtered := make(map[string]any, len(request.GetFields()))
		for _, field := range request.GetFields() {
			if value, ok := settings[field]; ok {
				filtered[field] = value
			}
		}
		settings = filtered
	}
	document, err := protocolV2Document(QueryOwnSettingsSchemaID, QueryOwnSettingsSchemaV1, map[string]any{"settings": settings})
	if err != nil {
		response.Error = protocolV2Failure("host.query_encode_failed", err.Error())
		return response, nil
	}
	response.Rows = []*protocolv2.TypedDocument{document}
	response.Page = &protocolv2.PageInfo{}
	return response, nil
}

func (s *protocolV2QueryServer) Stream(request *hostv2.QueryRequest, stream grpc.ServerStreamingServer[hostv2.QueryRow]) error {
	response, err := s.Execute(stream.Context(), request)
	if err != nil {
		return err
	}
	if response.GetError() != nil {
		return stream.Send(&hostv2.QueryRow{Context: response.GetContext(), Error: response.GetError()})
	}
	for index, row := range response.GetRows() {
		if err := stream.Send(&hostv2.QueryRow{Context: response.GetContext(), Sequence: uint64(index + 1), Value: row}); err != nil {
			return err
		}
	}
	return nil
}

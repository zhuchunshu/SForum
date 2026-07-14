package extensionsruntime_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	pluginv2sdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
	pluginwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const protocolV2RouteHelperEnv = "protocol-v2-route-e2e"

func TestProtocolV2RouteAcrossRealSubprocessAndManagerAdmission(t *testing.T) {
	extension := protocolV2RouteExtension(t)
	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust: staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{TrustGrantID: "41", ImpactDigest: "impact-41"}},
	})
	manager := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{Starter: starter})
	if err := manager.Start(context.Background(), extension); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Stop(context.Background(), extension) })
	snapshot, lease, err := manager.AcquireActiveRuntimeCall(context.Background(), extension.ID, extensionsruntime.RuntimeCallRoute)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Target.BaseURL != "" || snapshot.Identity.InstanceID == "" || snapshot.Admission.ActiveTotal != 1 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	response, err := manager.InvokeRouteInstance(lease.Context, snapshot.Identity, protocolV2RouteE2ERequest())
	lease.Release()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusCreated || response.Body["instance"] != snapshot.Identity.InstanceID ||
		response.Body["actor"] != float64(42) || response.Body["title"] != "hello" {
		t.Fatalf("response = %#v", response)
	}
	if after, err := manager.InspectRuntimeInstance(snapshot.Identity); err != nil || after.Admission.ActiveTotal != 0 {
		t.Fatalf("admission after call = %#v, %v", after, err)
	}
}

func TestProtocolV2RouteExactRetainedInstanceNeverFallsBackActive(t *testing.T) {
	extension := protocolV2RouteExtension(t)
	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust: staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{TrustGrantID: "41", ImpactDigest: "impact-41"}},
	})
	first, err := starter.Start(context.Background(), extension)
	if err != nil {
		t.Fatal(err)
	}
	second, err := starter.Start(context.Background(), extension)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = starter.StopInstance(context.Background(), extensionsruntime.RuntimeInstanceIdentity{ExtensionID: extension.ID, InstanceID: first.InstanceID})
		_ = starter.Stop(context.Background(), extension)
	})
	if first.InstanceID == second.InstanceID {
		t.Fatal("replacement reused runtime identity")
	}
	response, err := starter.InvokeRouteInstance(context.Background(), extensionsruntime.RuntimeInstanceIdentity{
		ExtensionID: extension.ID, InstanceID: first.InstanceID,
	}, protocolV2RouteE2ERequest())
	if err != nil {
		t.Fatal(err)
	}
	if response.Body["instance"] != first.InstanceID {
		t.Fatalf("retained call fell back to active %q: %#v", second.InstanceID, response.Body)
	}
	_, err = starter.InvokeRouteInstance(context.Background(), extensionsruntime.RuntimeInstanceIdentity{
		ExtensionID: extension.ID, InstanceID: "missing-instance",
	}, protocolV2RouteE2ERequest())
	if !errors.Is(err, extensionsruntime.ErrRuntimeInstanceNotFound) {
		t.Fatalf("missing exact instance error = %v", err)
	}
}

func TestProtocolV2RouteHelperProcess(t *testing.T) {
	if os.Getenv("SFORUM_PLUGIN_HELPER") != protocolV2RouteHelperEnv {
		return
	}
	pluginv2sdk.Serve(&protocolV2RouteE2EServer{Server: pluginv2sdk.NewServer()})
	os.Exit(0)
}

type protocolV2RouteE2EServer struct{ *pluginv2sdk.Server }

func (s *protocolV2RouteE2EServer) InvokeRoute(_ context.Context, request *pluginwire.RouteRequest) (*pluginwire.RouteResponse, error) {
	ctx := request.GetContext()
	if request.GetRouteId() != "runtime.v2.route.echo" || request.GetContractVersion() != "runtime.v2.route.echo@1" ||
		request.GetMethod() != http.MethodPost || request.GetPath() != "/runtime/41" ||
		request.GetPathParameters()["id"] != "41" || request.GetQueryParameters()["page"] != "2" ||
		request.GetBody().GetSchemaId() != "runtime.v2.route.request" || request.GetBody().GetSchemaVersion() != "1" ||
		ctx.GetActor().GetUserId() != 42 {
		return &pluginwire.RouteResponse{
			Context: protocolV2RouteResponseContext(ctx),
			Error:   &protocolwire.ErrorDetail{Code: protocolwire.ErrorCode_ERROR_CODE_INVALID_ARGUMENT, Reason: "route.fixture_invalid"},
		}, nil
	}
	input := request.GetBody().GetValue().AsMap()
	value, err := structpb.NewStruct(map[string]any{
		"instance": ctx.GetExtension().GetInstanceId(), "actor": ctx.GetActor().GetUserId(), "title": input["title"],
	})
	if err != nil {
		return nil, err
	}
	return &pluginwire.RouteResponse{
		Context: protocolV2RouteResponseContext(ctx), StatusCode: http.StatusCreated,
		Headers: []*protocolwire.Header{{Name: "X-Route-E2E", Values: []string{"ok"}}},
		Body:    &protocolwire.TypedDocument{SchemaId: "runtime.v2.route.response", SchemaVersion: "1", Value: value},
	}, nil
}

func protocolV2RouteResponseContext(request *protocolwire.RequestContext) *protocolwire.ResponseContext {
	return &protocolwire.ResponseContext{
		RequestId: request.GetRequestId(), Trace: proto.Clone(request.GetTrace()).(*protocolwire.TraceContext),
		Extension: proto.Clone(request.GetExtension()).(*protocolwire.ExtensionIdentity), ServerTime: timestamppb.Now(),
	}
}

func protocolV2RouteExtension(t *testing.T) extensions.Extension {
	t.Helper()
	extension := protocolV2TestExtension(t, "v2")
	launcher := "#!/bin/sh\nSFORUM_PLUGIN_HELPER=" + protocolV2RouteHelperEnv + " exec " + shellQuote(os.Args[0]) +
		" -test.run=TestProtocolV2RouteHelperProcess -- \"$@\"\n"
	if err := os.WriteFile(filepath.Join(extension.PackagePath, "backend", "plugin"), []byte(launcher), 0o755); err != nil {
		t.Fatal(err)
	}
	extension.Manifest.Events = nil
	extension.Manifest.Services = nil
	extension.Manifest.Routes = []extensions.ManifestRoute{{
		ID: "runtime.v2.route.echo", ContractVersion: "runtime.v2.route.echo@1", Action: "add",
		Path: "/runtime/{id}", Methods: []string{http.MethodPost}, Handler: "route.echo",
		RequestSchema: "runtime.v2.route.request@1", ResponseSchema: "runtime.v2.route.response@1",
	}}
	return extension
}

func protocolV2RouteE2ERequest() extensionsruntime.ProtocolV2RouteRequest {
	return extensionsruntime.ProtocolV2RouteRequest{
		RouteID: "runtime.v2.route.echo", ContractVersion: "runtime.v2.route.echo@1",
		Method: http.MethodPost, Path: "/runtime/41", PathParameters: map[string]string{"id": "41"},
		QueryParameters: map[string]string{"page": "2"}, RequestSchema: "runtime.v2.route.request@1",
		ResponseSchema: "runtime.v2.route.response@1", Body: map[string]any{"title": "hello"}, BodyPresent: true,
		Actor: extensionsruntime.NewProtocolV2RouteActor(42, true, map[string]bool{"topics.write": true}), Timeout: 3 * time.Second,
	}
}

var _ pluginwire.PluginRuntimeServiceServer = (*protocolV2RouteE2EServer)(nil)

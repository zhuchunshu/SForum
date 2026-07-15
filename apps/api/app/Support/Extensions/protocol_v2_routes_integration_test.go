package extensionsruntime_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	moderation "github.com/zhuchunshu/sforum/apps/api/app/Models/Moderation"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
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

func TestProtocolV2RouteDeliversHostSignedDelegationAcrossRealBroker(t *testing.T) {
	pool := new(pgxpool.Pool)
	authority, err := hostapi.NewProtocolV2ActorDelegationAuthority()
	if err != nil {
		t.Fatal(err)
	}
	commandRuntime, err := hostapi.NewPostgresProtocolV2CommandRuntime(hostapi.PostgresProtocolV2CommandRuntimeConfig{
		Pool: pool, ActorDelegations: authority, Jobs: new(supportjobs.Dispatcher),
		Moderation: moderation.NewPostgresStore(pool), AttachmentStatuses: protocolV2RouteAttachmentMutator{},
	})
	if err != nil {
		t.Fatal(err)
	}
	gateway := hostapi.NewGateway(hostapi.New(hostapi.Config{}))
	if err := gateway.BindProtocolV2CommandRuntime(commandRuntime); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gateway.Close() })

	extension := protocolV2RouteExtension(t)
	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		HostAPI: gateway,
		Trust:   staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{TrustGrantID: "41", ImpactDigest: "impact-41"}},
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
	request := protocolV2RouteE2ERequest()
	request.Actor = extensionsruntime.NewProtocolV2RouteActor(42, true, map[string]bool{"user.manage": true})
	request.IdempotencyKey = "real-route-request-42"
	response, err := manager.InvokeRouteInstance(lease.Context, snapshot.Identity, request)
	lease.Release()
	if err != nil {
		t.Fatal(err)
	}
	if response.Body["delegatedPlan"] != true || response.Body["delegationCount"] != float64(1) {
		t.Fatalf("delegated response = %#v", response)
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

func TestProtocolV2CustomGuardAcrossRealSubprocessAndManagerAdmission(t *testing.T) {
	extension := protocolV2RouteExtension(t)
	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust: staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{TrustGrantID: "41", ImpactDigest: "impact-41"}},
	})
	manager := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{Starter: starter})
	if err := manager.Start(context.Background(), extension); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Stop(context.Background(), extension) })
	snapshot, lease, err := manager.AcquireActiveRuntimeCall(context.Background(), extension.ID, extensionsruntime.RuntimeCallGuard)
	if err != nil {
		t.Fatal(err)
	}
	request := extensionsruntime.ProtocolV2GuardRequest{
		GuardID: "runtime.v2.guard.owner", GuardContractVersion: "runtime.v2.guard.owner@1",
		RouteID: "runtime.v2.route.echo", RouteContractVersion: "runtime.v2.route.echo@1",
		Method: http.MethodPost, Path: "/runtime/41", PathParameters: map[string]string{"id": "41"},
		RequestSchema: "runtime.v2.route.request@1", Body: map[string]any{"title": "guarded"}, BodyPresent: true,
		Actor: extensionsruntime.NewProtocolV2RouteActor(42, true, map[string]bool{"topics.write": true}), Timeout: 3 * time.Second,
	}
	err = manager.InvokeGuardInstance(lease.Context, snapshot.Identity, request)
	lease.Release()
	if err != nil {
		t.Fatal(err)
	}
	request.QueryParameters = map[string]string{"deny": "1"}
	snapshot, lease, err = manager.AcquireActiveRuntimeCall(context.Background(), extension.ID, extensionsruntime.RuntimeCallGuard)
	if err != nil {
		t.Fatal(err)
	}
	err = manager.InvokeGuardInstance(lease.Context, snapshot.Identity, request)
	lease.Release()
	if !errors.Is(err, extensionsruntime.ErrProtocolV2GuardDenied) {
		t.Fatalf("denied guard error = %v", err)
	}
	if after, err := manager.InspectRuntimeInstance(snapshot.Identity); err != nil || after.Admission.ActiveTotal != 0 {
		t.Fatalf("guard admission after call = %#v, %v", after, err)
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

func (s *protocolV2RouteE2EServer) InvokeRoute(callCtx context.Context, request *pluginwire.RouteRequest) (*pluginwire.RouteResponse, error) {
	ctx := request.GetContext()
	if request.GetRouteId() == "runtime.v2.guard.owner" {
		status := http.StatusNoContent
		if request.GetQueryParameters()["deny"] == "1" {
			status = http.StatusForbidden
		}
		if request.GetContractVersion() != "runtime.v2.guard.owner@1" || request.GetMethod() != http.MethodPost ||
			request.GetPath() != "/runtime/41" || request.GetPathParameters()["id"] != "41" ||
			request.GetContext().GetActor().GetUserId() != 42 ||
			request.GetBody().GetSchemaId() != "runtime.v2.route.request" {
			status = http.StatusForbidden
		}
		return &pluginwire.RouteResponse{Context: protocolV2RouteResponseContext(ctx), StatusCode: uint32(status)}, nil
	}
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
	delegatedPlan := false
	if len(ctx.GetHostCommandDelegations()) > 0 {
		host, err := s.Host()
		if err != nil {
			return nil, err
		}
		value, err := structpb.NewStruct(map[string]any{"userId": "99", "status": "disabled"})
		if err != nil {
			return nil, err
		}
		command, err := host.DelegatedCommandRequest(ctx,
			hostapi.CommandIdentityUserStatusSetID, hostapi.CommandIdentityUserStatusSetVersion,
			&protocolwire.TypedDocument{
				SchemaId:      hostapi.CommandIdentityUserStatusInputSchemaID,
				SchemaVersion: hostapi.CommandIdentityUserStatusSchemaVersion, Value: value,
			},
		)
		if err != nil {
			return nil, err
		}
		command.ExpectedRevision = "0"
		plan, err := host.Commands.Plan(callCtx, command)
		if err != nil {
			return nil, err
		}
		if plan.GetError() != nil || plan.GetPlanId() == "" {
			return nil, errors.New("Host rejected the real actor delegation")
		}
		delegatedPlan = true
	}
	value, err := structpb.NewStruct(map[string]any{
		"instance": ctx.GetExtension().GetInstanceId(), "actor": ctx.GetActor().GetUserId(), "title": input["title"],
		"delegatedPlan": delegatedPlan, "delegationCount": len(ctx.GetHostCommandDelegations()),
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

type protocolV2RouteAttachmentMutator struct{}

func (protocolV2RouteAttachmentMutator) MutateProtocolV2AttachmentStatus(
	context.Context,
	pgx.Tx,
	int64,
	string,
) (hostapi.ProtocolV2AttachmentStatusResult, error) {
	return hostapi.ProtocolV2AttachmentStatusResult{}, errors.New("route delegation fixture does not execute attachment commands")
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
	extension.Manifest.Database = &extensions.ManifestDatabase{
		ContractVersion: extension.ID + ".database@1",
		Grants:          []string{extensionmanifest.DatabaseGrantHostCommands},
		Retention: extensionmanifest.ManifestRetention{
			OnDisable: "retain", OnUninstall: "retain",
		},
	}
	extension.Manifest.Guards = []extensions.ManifestGuard{{
		ID: "runtime.v2.guard.owner", ContractVersion: "runtime.v2.guard.owner@1", Kind: "custom",
		Entry: "backend/guard", Digest: strings.Repeat("c", 64),
	}}
	extension.Manifest.Routes = []extensions.ManifestRoute{{
		ID: "runtime.v2.route.echo", ContractVersion: "runtime.v2.route.echo@1", Action: "add",
		Path: "/runtime/{id}", Methods: []string{http.MethodPost}, Guard: "runtime.v2.guard.owner", Handler: "route.echo",
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

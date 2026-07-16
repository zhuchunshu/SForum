package pluginv2

import (
	"sort"
	"strings"
	"testing"

	_ "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/host/v2"
	_ "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	_ "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestGeneratedServiceCatalog(t *testing.T) {
	want := []string{
		"sforum.host.v2.AdminSurfaceService",
		"sforum.host.v2.AuditService",
		"sforum.host.v2.CacheService",
		"sforum.host.v2.DatabaseService",
		"sforum.host.v2.FileService",
		"sforum.host.v2.HostCommandService",
		"sforum.host.v2.HostQueryService",
		"sforum.host.v2.HttpService",
		"sforum.host.v2.IdentityService",
		"sforum.host.v2.JobService",
		"sforum.host.v2.MediaService",
		"sforum.host.v2.NavigationService",
		"sforum.host.v2.PermissionService",
		"sforum.host.v2.ScheduleService",
		"sforum.host.v2.SecretService",
		"sforum.host.v2.ServiceDiscoveryService",
		"sforum.host.v2.TracingService",
		"sforum.plugin.v2.PluginRuntimeService",
	}

	var got []string
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if !strings.HasPrefix(string(file.Package()), "sforum.") {
			return true
		}
		services := file.Services()
		for i := 0; i < services.Len(); i++ {
			got = append(got, string(services.Get(i).FullName()))
		}
		return true
	})
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("generated service catalog drifted\nwant:\n%s\ngot:\n%s", strings.Join(want, "\n"), strings.Join(got, "\n"))
	}
}

func TestRequiredEnvelopeAndCommandFields(t *testing.T) {
	assertFields(t, "sforum.protocol.v2.RequestContext",
		"request_id", "trace", "actor", "locale", "deadline", "extension", "granted_authority", "idempotency_key", "host_command_delegations", "host_query_delegations")
	assertFields(t, "sforum.protocol.v2.HostCommandDelegation",
		"command_id", "command_version", "idempotency_key", "token")
	assertFields(t, "sforum.protocol.v2.HostQueryDelegation",
		"query_id", "contract_version", "plan_version", "result_schema_id", "result_schema_version", "scope", "token")
	assertFields(t, "sforum.protocol.v2.ExtensionIdentity",
		"extension_id", "extension_version", "artifact_digest", "trust_grant_id", "runtime_epoch", "instance_id")
	assertFields(t, "sforum.protocol.v2.TypedDocument", "schema_id", "schema_version", "value")
	assertFields(t, "sforum.protocol.v2.HandshakeRequest",
		"context", "host_protocols", "host_features", "limits", "host_broker_id", "host_api_version", "runtime_token")
	assertFields(t, "sforum.protocol.v2.LifecycleRequest",
		"context", "action", "plan_version", "step_id", "checkpoint", "input", "dry_run", "forced")
	assertFields(t, "sforum.host.v2.CommandRequest",
		"context", "command_id", "command_version", "idempotency_key", "dry_run", "expected_revision", "input", "actor_delegation")
	assertFields(t, "sforum.host.v2.QueryRequest",
		"context", "query_id", "plan_version", "fields", "filters", "sorts", "page", "result_schema_id", "result_schema_version", "contract_version", "relations", "scope", "actor_delegation", "offset")
	assertFields(t, "sforum.host.v2.QueryResponse", "context", "rows", "page", "error", "next_offset")
	assertFields(t, "sforum.host.v2.QueryRow", "context", "sequence", "value", "error", "page", "next_offset", "final")
	assertFields(t, "sforum.host.v2.CommandResult",
		"context", "state", "transaction_id", "audit_event_id", "committed_revision", "output", "error")
	assertFields(t, "sforum.plugin.v2.CommandInvocationRequest",
		"context", "command_id", "contract_version", "handler", "input")
	assertFields(t, "sforum.plugin.v2.CommandInvocationResponse", "context", "result", "error")
}

func TestStreamingModesRemainExplicit(t *testing.T) {
	tests := []struct {
		method          string
		clientStreaming bool
		serverStreaming bool
	}{
		{"sforum.plugin.v2.PluginRuntimeService.RunLifecycle", false, true},
		{"sforum.plugin.v2.PluginRuntimeService.StreamRoute", true, true},
		{"sforum.plugin.v2.PluginRuntimeService.ExecuteJob", false, true},
		{"sforum.plugin.v2.PluginRuntimeService.TransferFile", true, true},
		{"sforum.plugin.v2.PluginRuntimeService.StreamService", true, true},
		{"sforum.host.v2.HostQueryService.Stream", false, true},
		{"sforum.host.v2.DatabaseService.StreamQuery", false, true},
		{"sforum.host.v2.JobService.Watch", false, true},
		{"sforum.host.v2.ServiceDiscoveryService.Stream", true, true},
		{"sforum.host.v2.FileService.Read", false, true},
		{"sforum.host.v2.FileService.Write", true, false},
		{"sforum.host.v2.HttpService.Stream", true, true},
	}
	for _, test := range tests {
		method := methodDescriptor(t, protoreflect.FullName(test.method))
		if method.IsStreamingClient() != test.clientStreaming || method.IsStreamingServer() != test.serverStreaming {
			t.Errorf("%s streaming mode = (%v,%v), want (%v,%v)", test.method,
				method.IsStreamingClient(), method.IsStreamingServer(), test.clientStreaming, test.serverStreaming)
		}
	}
}

func assertFields(t *testing.T, messageName string, names ...protoreflect.Name) {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(messageName))
	if err != nil {
		t.Fatalf("find message %s: %v", messageName, err)
	}
	message, ok := descriptor.(protoreflect.MessageDescriptor)
	if !ok {
		t.Fatalf("%s is %T, want message", messageName, descriptor)
	}
	for _, name := range names {
		if message.Fields().ByName(name) == nil {
			t.Errorf("%s missing field %s", messageName, name)
		}
	}
}

func methodDescriptor(t *testing.T, fullName protoreflect.FullName) protoreflect.MethodDescriptor {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(fullName)
	if err != nil {
		t.Fatalf("find method %s: %v", fullName, err)
	}
	method, ok := descriptor.(protoreflect.MethodDescriptor)
	if !ok {
		t.Fatalf("%s is %T, want method", fullName, descriptor)
	}
	return method
}

package extensionsruntime_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	capabilities "github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin"
	pluginv2sdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
	pluginwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestProtocolV2NegotiatesGRPCAndInvokesTypedHook(t *testing.T) {
	extension := protocolV2TestExtension(t, "v2")
	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust: staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{TrustGrantID: "41", ImpactDigest: "impact-41"}},
	})
	target, err := starter.Start(context.Background(), extension)
	if err != nil {
		t.Fatalf("start protocol v2 plugin: %v", err)
	}
	defer starter.Stop(context.Background(), extension)
	if target.BaseURL != "" {
		t.Fatalf("v2 route target must use gRPC registry, got %#v", target)
	}

	result := starter.InvokeHook(context.Background(), extension, extensionsruntime.HookInput{
		Name: "topic.before_create", Kind: "filter", DeliveryID: 41,
		CorrelationID: "trace-41", Timeout: 2 * time.Second,
		Payload: map[string]any{"title": "before"}, PatchFields: []string{"title"},
	})
	if !result.OK || result.Patch["title"] != "after-v2" {
		t.Fatalf("unexpected v2 hook result: %#v", result)
	}
}

func TestProtocolSelectionNeverSilentlyDowngrades(t *testing.T) {
	tests := []struct {
		name            string
		manifestVersion int
		helperMode      string
		hostAPIVersion  string
	}{
		{"v2 manifest with v1 binary", 2, "v1", "sforum.host/v2"},
		{"v1 manifest with v2 binary", 1, "v2", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			extension := protocolV2TestExtension(t, test.helperMode)
			extension.Manifest.Backend.ProtocolVersion = test.manifestVersion
			extension.Manifest.Backend.HostAPIVersion = test.hostAPIVersion
			starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
				Trust: staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{TrustGrantID: "41", ImpactDigest: "impact-41"}},
			})
			if _, err := starter.Start(context.Background(), extension); err == nil {
				t.Fatal("expected exact protocol negotiation failure")
			}
		})
	}
}

func TestProtocolV2RejectsUploadedArtifactWithoutLiveGrant(t *testing.T) {
	extension := protocolV2TestExtension(t, "v2")
	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{})
	_, err := starter.Start(context.Background(), extension)
	if !errors.Is(err, extensions.ErrTrustGrantNotFound) {
		t.Fatalf("missing live grant error = %v", err)
	}
}

func TestProtocolV2HelperProcess(t *testing.T) {
	switch os.Getenv("SFORUM_PLUGIN_HELPER") {
	case "protocol-v2":
		pluginv2sdk.Serve(&protocolV2Helper{Server: pluginv2sdk.NewServer()})
		os.Exit(0)
	case "protocol-v1":
		pluginsdk.Serve(protocolV1Helper{})
		os.Exit(0)
	}
}

type protocolV2Helper struct {
	*pluginv2sdk.Server
}

func (s *protocolV2Helper) InvokeHook(_ context.Context, request *pluginwire.HookRequest) (*pluginwire.HookResponse, error) {
	ctx := request.GetContext()
	identity := ctx.GetExtension()
	if identity.GetExtensionId() != "runtime.v2" || identity.GetArtifactDigest() == "" ||
		identity.GetTrustGrantId() != "41" || ctx.GetLocale() != "und" || ctx.GetDeadline() == nil {
		return &pluginwire.HookResponse{Error: &protocolwire.ErrorDetail{
			Code: protocolwire.ErrorCode_ERROR_CODE_INVALID_ARGUMENT, Reason: "fixture.context_invalid", Message: "typed runtime context is incomplete",
		}}, nil
	}
	if request.GetPayload().GetSchemaId() != "sforum.hook.topic.before_create" || len(ctx.GetGrantedAuthority()) != 1 ||
		ctx.GetGrantedAuthority()[0].GetKey() != capabilities.HostAPI {
		return &pluginwire.HookResponse{Error: &protocolwire.ErrorDetail{
			Code: protocolwire.ErrorCode_ERROR_CODE_PERMISSION_DENIED, Reason: "fixture.authority_invalid", Message: "authority or payload contract is incomplete",
		}}, nil
	}
	patch, err := structpb.NewStruct(map[string]any{"title": "after-v2"})
	if err != nil {
		return nil, err
	}
	return &pluginwire.HookResponse{
		Context:  &protocolwire.ResponseContext{RequestId: ctx.GetRequestId(), Extension: identity},
		Accepted: true,
		Patch:    &protocolwire.TypedDocument{SchemaId: "sforum.hook.topic.before_create.patch", SchemaVersion: "1", Value: patch},
	}, nil
}

type protocolV1Helper struct{ pluginsdk.Noop }

type staticRuntimeTrust struct {
	identity extensions.RuntimeTrustIdentity
}

func (s staticRuntimeTrust) RuntimeIdentity(context.Context, extensions.Extension) (extensions.RuntimeTrustIdentity, error) {
	return s.identity, nil
}

func protocolV2TestExtension(t *testing.T, helperMode string) extensions.Extension {
	t.Helper()
	packageRoot := filepath.Join(t.TempDir(), "runtime.v2", "1.0.0")
	filesRoot := filepath.Join(packageRoot, "backend")
	if err := os.MkdirAll(filesRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	launcher := "#!/bin/sh\nSFORUM_PLUGIN_HELPER=protocol-" + helperMode + " exec " + shellQuote(os.Args[0]) + " -test.run=TestProtocolV2HelperProcess -- \"$@\"\n"
	if err := os.WriteFile(filepath.Join(filesRoot, "plugin"), []byte(launcher), 0o755); err != nil {
		t.Fatal(err)
	}
	return extensions.Extension{
		ID: "runtime.v2", Name: "Runtime V2", Version: "1.0.0", Type: extensions.TypePlugin,
		Status: extensions.StatusEnabled, Source: extensions.SourceUploaded,
		PackageDigest: strings.Repeat("a", 64), PackagePath: packageRoot,
		CapabilityGrants: []extensions.CapabilityGrant{{Key: capabilities.HostAPI, Risk: capabilities.RiskLow}},
		Manifest: extensions.Manifest{
			ManifestVersion: 3, ID: "runtime.v2", Version: "1.0.0", Type: extensions.TypePlugin,
			Backend: extensions.ManifestBackend{
				Entry: "backend/plugin", RPC: "hashicorp-go-plugin", ProtocolVersion: 2, HostAPIVersion: "sforum.host/v2",
			},
		},
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

package hostapi

import (
	"context"
	"testing"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	hostv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/host/v2"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestExtensionPluginDisableRequiresExtensionsManageCapability(t *testing.T) {
	authority, err := NewProtocolV2ActorDelegationAuthority()
	if err != nil {
		t.Fatal(err)
	}
	definition := newProtocolV2ExtensionPluginDisableCommandDefinition()
	backend := newFakeProtocolV2CommandBackend()
	engine, err := newProtocolV2CommandEngineWithActorDelegation(backend, authority, definition)
	if err != nil {
		t.Fatal(err)
	}
	// 无能力源：拒绝。
	request := testProtocolV2ExtensionsManageDisableRequest(t, "target.plugin")
	plan, err := engine.plan(context.Background(), request)
	if err != nil || plan.GetError().GetReason() != "host.command_capability_denied" {
		t.Fatalf("plan without capability source = %#v err=%v", plan.GetError(), err)
	}

	// 有能力源但缺少 extensions.manage：拒绝。
	engine.BindCapabilitySource(fakeCaps{set: capabilities.NewSet([]string{capabilities.HostAPI})})
	plan, err = engine.plan(context.Background(), request)
	if err != nil || plan.GetError().GetReason() != "host.command_capability_denied" {
		t.Fatalf("plan without extensions.manage = %#v err=%v", plan.GetError(), err)
	}

	// 授予 extensions.manage 后，仍因缺少 actor delegation 失败（证明能力门禁已通过）。
	engine.BindCapabilitySource(fakeCaps{set: capabilities.NewSet([]string{capabilities.ExtensionsManage})})
	plan, err = engine.plan(context.Background(), request)
	if err != nil || plan.GetError().GetReason() != "host.command_actor_delegation_required" {
		t.Fatalf("plan after capability grant = %#v err=%v", plan.GetError(), err)
	}
}

func TestExtensionPluginDisableRejectsSelfTarget(t *testing.T) {
	mutation, err := protocolV2ExtensionPluginDisableMutationFromRequest(
		testProtocolV2ExtensionsManageDisableRequest(t, "demo.plugin"),
	)
	if err != nil || mutation.targetExtensionID != "demo.plugin" {
		t.Fatalf("mutation=%#v err=%v", mutation, err)
	}
	// 空目标失败关闭。
	if _, err := protocolV2ExtensionPluginDisableMutationFromRequest(&hostv2.CommandRequest{
		CommandId: CommandExtensionPluginDisableID, CommandVersion: CommandExtensionPluginDisableVersion,
		Input: &protocolv2.TypedDocument{SchemaId: CommandExtensionPluginDisableInputSchema, SchemaVersion: CommandExtensionPluginDisableSchemaV1},
	}); err == nil {
		t.Fatal("empty target accepted")
	}
}

func testProtocolV2ExtensionsManageDisableRequest(t *testing.T, target string) *hostv2.CommandRequest {
	t.Helper()
	value, err := structpb.NewStruct(map[string]any{"targetExtensionId": target})
	if err != nil {
		t.Fatal(err)
	}
	requestContext := testProtocolV2RequestContext()
	// 插件发起的 Host Command 不得自带 actor；委托令牌另附。
	requestContext.Actor = nil
	requestContext.IdempotencyKey = "extensions-manage-disable-1"
	return &hostv2.CommandRequest{
		Context:        requestContext,
		CommandId:      CommandExtensionPluginDisableID,
		CommandVersion: CommandExtensionPluginDisableVersion,
		IdempotencyKey: "extensions-manage-disable-1",
		Input: &protocolv2.TypedDocument{
			SchemaId: CommandExtensionPluginDisableInputSchema, SchemaVersion: CommandExtensionPluginDisableSchemaV1,
			Value: value,
		},
	}
}

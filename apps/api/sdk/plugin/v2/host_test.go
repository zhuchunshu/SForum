package pluginv2

import (
	"testing"
	"time"

	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestHostRequestContextRebindsRuntimeOwnedFields(t *testing.T) {
	identity := &protocolwire.ExtensionIdentity{
		ExtensionId: "demo.v2", ExtensionVersion: "1.0.0", ArtifactDigest: "artifact",
		TrustGrantId: "grant", RuntimeEpoch: 8, InstanceId: "instance-8",
	}
	authority := []*protocolwire.AuthorityGrant{{Key: "settings.own", ContractVersion: HostAPIVersion}}
	host := &Host{identity: identity, authority: authority, instance: identity.InstanceId}
	parent := &protocolwire.RequestContext{
		RequestId: "plugin-call", Locale: "zh-CN", Deadline: timestamppb.New(time.Now().Add(time.Hour)),
		Trace: &protocolwire.TraceContext{TraceId: "trace-1"}, Actor: &protocolwire.Actor{UserId: 42},
		Extension:        &protocolwire.ExtensionIdentity{ExtensionId: "forged"},
		GrantedAuthority: []*protocolwire.AuthorityGrant{{Key: "raw.database"}},
	}

	result := host.RequestContext(parent)
	if result.GetRequestId() != "instance-8-host-1" || !proto.Equal(result.GetExtension(), identity) ||
		!equalAuthority(result.GetGrantedAuthority(), authority) {
		t.Fatalf("runtime binding = %#v", result)
	}
	if result.GetActor().GetUserId() != 42 || result.GetLocale() != "zh-CN" || result.GetTrace().GetTraceId() != "trace-1" {
		t.Fatalf("request context was not propagated: %#v", result)
	}
	if remaining := time.Until(result.GetDeadline().AsTime()); remaining <= 0 || remaining > 6*time.Second {
		t.Fatalf("deadline was not bounded: %s", remaining)
	}
	if parent.GetExtension().GetExtensionId() != "forged" || parent.GetGrantedAuthority()[0].GetKey() != "raw.database" {
		t.Fatalf("parent context was mutated: %#v", parent)
	}
}

func equalAuthority(left, right []*protocolwire.AuthorityGrant) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !proto.Equal(left[index], right[index]) {
			return false
		}
	}
	return true
}

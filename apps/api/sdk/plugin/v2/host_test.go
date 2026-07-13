package pluginv2

import (
	"context"
	"io"
	"testing"
	"time"

	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/grpc/metadata"
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

func TestHostClientStreamKeepsReceiveSideAliveAfterCloseSend(t *testing.T) {
	cancelled := false
	stream := &hostClientStream{
		ClientStream: fakeHostClientStream{},
		cancel:       func() { cancelled = true },
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatal(err)
	}
	if cancelled {
		t.Fatal("CloseSend cancelled the server-streaming receive side")
	}
	if err := stream.RecvMsg(new(any)); err != io.EOF {
		t.Fatalf("RecvMsg error = %v", err)
	}
	if !cancelled {
		t.Fatal("terminal RecvMsg did not release the stream context")
	}
}

type fakeHostClientStream struct{}

func (fakeHostClientStream) Header() (metadata.MD, error) { return nil, nil }
func (fakeHostClientStream) Trailer() metadata.MD         { return nil }
func (fakeHostClientStream) CloseSend() error             { return nil }
func (fakeHostClientStream) Context() context.Context     { return context.Background() }
func (fakeHostClientStream) SendMsg(any) error            { return nil }
func (fakeHostClientStream) RecvMsg(any) error            { return io.EOF }

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

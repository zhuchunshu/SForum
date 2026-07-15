package pluginv2

import (
	"context"
	"errors"
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
		HostCommandDelegations: []*protocolwire.HostCommandDelegation{{
			CommandId: "sforum.demo", CommandVersion: "1", IdempotencyKey: "request-42", Token: "secret-token",
		}},
	}

	result := host.RequestContext(parent)
	if result.GetRequestId() != "instance-8-host-1" || !proto.Equal(result.GetExtension(), identity) ||
		!equalAuthority(result.GetGrantedAuthority(), authority) {
		t.Fatalf("runtime binding = %#v", result)
	}
	if result.GetActor() != nil || len(result.GetHostCommandDelegations()) != 0 ||
		result.GetLocale() != "zh-CN" || result.GetTrace().GetTraceId() != "trace-1" {
		t.Fatalf("request context was not propagated: %#v", result)
	}
	if remaining := time.Until(result.GetDeadline().AsTime()); remaining <= 0 || remaining > 6*time.Second {
		t.Fatalf("deadline was not bounded: %s", remaining)
	}
	if parent.GetActor().GetUserId() != 42 || parent.GetExtension().GetExtensionId() != "forged" || parent.GetGrantedAuthority()[0].GetKey() != "raw.database" {
		t.Fatalf("parent context was mutated: %#v", parent)
	}
}

func TestHostDelegatedCommandRequestSelectsOneTokenWithoutMutatingParent(t *testing.T) {
	identity := &protocolwire.ExtensionIdentity{
		ExtensionId: "demo.v2", ExtensionVersion: "1.0.0", ArtifactDigest: "artifact",
		TrustGrantId: "grant", RuntimeEpoch: 8, InstanceId: "instance-8",
	}
	host := &Host{identity: identity, instance: identity.InstanceId}
	parent := &protocolwire.RequestContext{
		IdempotencyKey: "request-42", Trace: &protocolwire.TraceContext{TraceId: "trace-1"},
		HostCommandDelegations: []*protocolwire.HostCommandDelegation{
			{CommandId: "sforum.other", CommandVersion: "1", IdempotencyKey: "request-42", Token: "other-token"},
			{CommandId: "sforum.demo", CommandVersion: "1", IdempotencyKey: "request-42", Token: "demo-token"},
		},
	}
	input := &protocolwire.TypedDocument{SchemaId: "demo.input", SchemaVersion: "1"}
	request, err := host.DelegatedCommandRequest(parent, " sforum.demo ", "1", input)
	if err != nil {
		t.Fatal(err)
	}
	if request.GetCommandId() != "sforum.demo" || request.GetCommandVersion() != "1" ||
		request.GetIdempotencyKey() != "request-42" || request.GetActorDelegation() != "demo-token" ||
		request.GetContext().GetIdempotencyKey() != "request-42" || len(request.GetContext().GetHostCommandDelegations()) != 0 ||
		request.GetContext().GetActor() != nil || request.GetContext().GetExtension().GetInstanceId() != identity.InstanceId {
		t.Fatalf("command request = %#v", request)
	}
	request.Input.SchemaId = "changed"
	if input.GetSchemaId() != "demo.input" || len(parent.GetHostCommandDelegations()) != 2 {
		t.Fatal("delegated command request mutated caller-owned values")
	}
	for _, command := range []string{"", "sforum.missing"} {
		if _, err := host.DelegatedCommandRequest(parent, command, "1", nil); !errors.Is(err, ErrHostActorDelegationUnavailable) {
			t.Fatalf("command %q error = %v", command, err)
		}
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

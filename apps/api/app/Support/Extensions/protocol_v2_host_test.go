package extensionsruntime

import (
	"context"
	"testing"
	"time"

	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProtocolV2HostUnaryAuthBindsExactRuntime(t *testing.T) {
	token := []byte("01234567890123456789012345678901")
	identity := &protocolv2.ExtensionIdentity{
		ExtensionId: "demo.v2", ExtensionVersion: "1.0.0", ArtifactDigest: "sha256:artifact",
		TrustGrantId: "grant-1", RuntimeEpoch: 7, InstanceId: "instance-1",
	}
	authority := []*protocolv2.AuthorityGrant{{Key: "settings.own", ContractVersion: hostAPIV2Version}}
	binding := newProtocolV2HostBinding(protocolV2ClientConfig{identity: identity, authority: authority, token: token})
	valid := func() *protocolv2.HealthRequest {
		return &protocolv2.HealthRequest{Context: &protocolv2.RequestContext{
			RequestId: "request-1", Deadline: timestamppb.New(time.Now().Add(time.Minute)),
			Extension: cloneV2Identity(identity), GrantedAuthority: cloneV2Authority(authority),
		}}
	}
	call := func(rawToken []byte, request *protocolv2.HealthRequest) error {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(ProtocolV2RuntimeTokenMetadataKey, string(rawToken)))
		_, err := protocolV2HostUnaryAuthInterceptor(binding)(ctx, request, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
			return &protocolv2.HealthResponse{Healthy: true}, nil
		})
		return err
	}

	if err := call(token, valid()); err != nil {
		t.Fatalf("valid runtime rejected: %v", err)
	}
	tests := []struct {
		name   string
		token  []byte
		mutate func(*protocolv2.HealthRequest)
		code   codes.Code
	}{
		{name: "stale token", token: []byte("11234567890123456789012345678901"), code: codes.Unauthenticated},
		{name: "stale identity", token: token, mutate: func(request *protocolv2.HealthRequest) {
			request.Context.Extension.RuntimeEpoch++
		}, code: codes.FailedPrecondition},
		{name: "forged authority", token: token, mutate: func(request *protocolv2.HealthRequest) {
			request.Context.GrantedAuthority[0].Key = "raw.database"
		}, code: codes.PermissionDenied},
		{name: "expired deadline", token: token, mutate: func(request *protocolv2.HealthRequest) {
			request.Context.Deadline = timestamppb.New(time.Now().Add(-time.Second))
		}, code: codes.DeadlineExceeded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := valid()
			if test.mutate != nil {
				test.mutate(request)
			}
			if code := status.Code(call(test.token, request)); code != test.code {
				t.Fatalf("code = %s, want %s", code, test.code)
			}
		})
	}
}

func TestProtocolV2HostBindingOwnsClonedIdentity(t *testing.T) {
	identity := &protocolv2.ExtensionIdentity{ExtensionId: "demo", RuntimeEpoch: 1}
	binding := newProtocolV2HostBinding(protocolV2ClientConfig{identity: identity, token: []byte("01234567890123456789012345678901")})
	identity.RuntimeEpoch = 2
	if proto.Equal(binding.identity, identity) || binding.identity.GetRuntimeEpoch() != 1 {
		t.Fatalf("binding identity changed with caller: %#v", binding.identity)
	}
}

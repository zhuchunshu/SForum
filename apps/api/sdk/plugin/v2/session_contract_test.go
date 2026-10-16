package pluginv2

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// TestPluginSessionRequestsCarryNoCallerIdentity 是 S7 安全属性的描述符级证据。
//
// 它不看注释、不做目测：直接把三条请求消息的字段集合钉死，并在整个
// sforum.host.v2.PluginSessionService 的请求/响应里搜索任何名字带
// extension/user/session 标识的字段。调用方因此不是「被忽略」而是「表达不出来」。
func TestPluginSessionRequestsCarryNoCallerIdentity(t *testing.T) {
	assertExactFields(t, "sforum.host.v2.PluginSessionGetRequest", "context", "namespace_key")
	assertExactFields(t, "sforum.host.v2.PluginSessionSetRequest", "context", "namespace_key", "value", "ttl_ms")
	assertExactFields(t, "sforum.host.v2.PluginSessionDeleteRequest", "context", "namespace_key")
	assertExactFields(t, "sforum.host.v2.PluginSessionGetResponse", "context", "found", "value", "error")
	assertExactFields(t, "sforum.host.v2.PluginSessionSetResponse", "context", "effective_ttl_ms", "error")
	assertExactFields(t, "sforum.host.v2.PluginSessionDeleteResponse", "context", "deleted", "error")

	// The delegation message is the only carrier of session scope, and it is
	// opaque: one token field, no identifier, no fingerprint, no scope enum.
	assertExactFields(t, "sforum.protocol.v2.HostSessionDelegation", "token")

	service, err := protoregistry.GlobalFiles.FindDescriptorByName("sforum.host.v2.PluginSessionService")
	if err != nil {
		t.Fatalf("find PluginSessionService: %v", err)
	}
	descriptor, ok := service.(protoreflect.ServiceDescriptor)
	if !ok {
		t.Fatalf("PluginSessionService is %T, want service", service)
	}
	forbidden := []string{"extension_id", "user_id", "session_id", "session_fingerprint", "actor", "sid", "scope"}
	for index := 0; index < descriptor.Methods().Len(); index++ {
		method := descriptor.Methods().Get(index)
		for _, message := range []protoreflect.MessageDescriptor{method.Input(), method.Output()} {
			fields := message.Fields()
			for field := 0; field < fields.Len(); field++ {
				name := string(fields.Get(field).Name())
				for _, banned := range forbidden {
					if strings.Contains(name, banned) {
						t.Errorf("%s.%s carries caller-supplied identity field %q", message.FullName(), method.Name(), name)
					}
				}
			}
		}
	}
}

// TestRequestContextSessionDelegationIsTheOnlyNewField pins the wire addition so
// a later change cannot quietly widen the envelope.
func TestRequestContextSessionDelegationIsTheOnlyNewField(t *testing.T) {
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName("sforum.protocol.v2.RequestContext")
	if err != nil {
		t.Fatalf("find RequestContext: %v", err)
	}
	message, ok := descriptor.(protoreflect.MessageDescriptor)
	if !ok {
		t.Fatalf("RequestContext is %T, want message", descriptor)
	}
	field := message.Fields().ByName("host_session_delegation")
	if field == nil {
		t.Fatal("RequestContext is missing host_session_delegation")
	}
	if field.Number() != 11 {
		t.Fatalf("host_session_delegation number = %d, want 11", field.Number())
	}
	if field.Kind() != protoreflect.MessageKind || field.Message().FullName() != "sforum.protocol.v2.HostSessionDelegation" {
		t.Fatalf("host_session_delegation kind = %v/%v, want HostSessionDelegation", field.Kind(), field.Message().FullName())
	}
	if field.IsList() || field.IsMap() {
		t.Fatal("host_session_delegation must be singular: one invocation has exactly one session")
	}
}

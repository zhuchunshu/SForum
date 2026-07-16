package pluginv2

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestCacheIncrementProtocolContract(t *testing.T) {
	method := methodDescriptor(t, protoreflect.FullName("sforum.host.v2.CacheService.Increment"))
	if method.IsStreamingClient() || method.IsStreamingServer() {
		t.Fatal("cache increment unexpectedly became a streaming RPC")
	}
	assertFields(t, "sforum.host.v2.CacheIncrementRequest", "context", "namespace", "key", "delta", "ttl")
	assertFields(t, "sforum.host.v2.CacheIncrementResponse", "context", "value", "error")
}

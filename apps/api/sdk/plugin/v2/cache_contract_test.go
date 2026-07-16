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

func TestCacheGetProtocolContract(t *testing.T) {
	assertFields(t, "sforum.host.v2.CacheGetResponse", "context", "found", "value", "error", "revision")
}

func TestCacheLockProtocolContract(t *testing.T) {
	for _, methodName := range []protoreflect.FullName{
		"sforum.host.v2.CacheService.AcquireLock",
		"sforum.host.v2.CacheService.RenewLock",
		"sforum.host.v2.CacheService.ReleaseLock",
		"sforum.host.v2.CacheService.SetAndReleaseLock",
	} {
		method := methodDescriptor(t, methodName)
		if method.IsStreamingClient() || method.IsStreamingServer() {
			t.Fatalf("%s unexpectedly became a streaming RPC", methodName)
		}
	}
	assertFields(t, "sforum.host.v2.CacheLockAcquireRequest", "context", "namespace", "key", "ttl")
	assertFields(t, "sforum.host.v2.CacheLockAcquireResponse", "context", "acquired", "lease_token", "expires_at", "error")
	assertFields(t, "sforum.host.v2.CacheLockRenewRequest", "context", "namespace", "key", "lease_token", "ttl")
	assertFields(t, "sforum.host.v2.CacheLockRenewResponse", "context", "renewed", "expires_at", "error")
	assertFields(t, "sforum.host.v2.CacheLockReleaseRequest", "context", "namespace", "key", "lease_token")
	assertFields(t, "sforum.host.v2.CacheLockReleaseResponse", "context", "released", "error")
	assertFields(t, "sforum.host.v2.CacheSetAndReleaseLockRequest",
		"context", "namespace", "key", "lease_token", "value", "ttl", "tags", "expected_revision")
	assertFields(t, "sforum.host.v2.CacheSetAndReleaseLockResponse", "context", "revision", "error")
}

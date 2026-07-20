package extensionsruntime

import (
	"sync"
	"testing"
)

type recordingShimTelemetry struct {
	mu    sync.Mutex
	calls []string
}

func (r *recordingShimTelemetry) RecordShimCall(contractID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, contractID)
}

func (r *recordingShimTelemetry) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.calls))
	copy(out, r.calls)
	return out
}

func TestProtocolV1CallRecordsAPILTSShimTelemetry(t *testing.T) {
	shim := &recordingShimTelemetry{}
	starter := NewProtocolStarter(ProtocolStarterConfig{ShimTelemetry: shim})

	starter.mu.Lock()
	starter.telemetry["sforum.smtp"] = &protocolTelemetry{
		version: 1, transport: "net/rpc", deprecated: true,
	}
	starter.recordProtocolCallLocked("sforum.smtp")
	starter.recordProtocolCallLocked("sforum.smtp")
	starter.mu.Unlock()

	calls := shim.snapshot()
	if len(calls) != 2 {
		t.Fatalf("shim calls = %d, want 2: %#v", len(calls), calls)
	}
	for _, id := range calls {
		if id != protocolV1ShimContractID {
			t.Fatalf("contract id = %q, want %q", id, protocolV1ShimContractID)
		}
	}
	// Per-extension counters must stay independent of APILTS.
	if got := starter.ProtocolTelemetry("sforum.smtp"); got.CallCount != 2 || got.ProtocolVersion != 1 {
		t.Fatalf("protocol telemetry = %#v", got)
	}
}

func TestProtocolV2CallDoesNotRecordProtocolV1Shim(t *testing.T) {
	shim := &recordingShimTelemetry{}
	starter := NewProtocolStarter(ProtocolStarterConfig{ShimTelemetry: shim})

	starter.mu.Lock()
	starter.telemetry["sforum.content-policy"] = &protocolTelemetry{
		version: 2, transport: "grpc", deprecated: false,
	}
	starter.recordProtocolCallLocked("sforum.content-policy")
	starter.mu.Unlock()

	if calls := shim.snapshot(); len(calls) != 0 {
		t.Fatalf("v2 must not record protocol.v1 shim: %#v", calls)
	}
	if got := starter.ProtocolTelemetry("sforum.content-policy"); got.CallCount != 1 {
		t.Fatalf("call count = %d", got.CallCount)
	}
}

func TestProtocolV1StartRecordsAPILTSShimTelemetry(t *testing.T) {
	shim := &recordingShimTelemetry{}
	starter := NewProtocolStarter(ProtocolStarterConfig{ShimTelemetry: shim})

	starter.mu.Lock()
	starter.recordProtocolStartLocked("sforum.storage-fs", 1)
	starter.mu.Unlock()

	calls := shim.snapshot()
	if len(calls) != 1 || calls[0] != protocolV1ShimContractID {
		t.Fatalf("start shim = %#v", calls)
	}
	if got := starter.ProtocolTelemetry("sforum.storage-fs"); got.StartCount != 1 || !got.Deprecated {
		t.Fatalf("start telemetry = %#v", got)
	}
}

func TestProtocolShimTelemetryNilIsSafe(t *testing.T) {
	starter := NewProtocolStarter(ProtocolStarterConfig{})
	starter.mu.Lock()
	starter.telemetry["ext"] = &protocolTelemetry{version: 1, transport: "net/rpc", deprecated: true}
	// 不得 panic。
	starter.recordProtocolCallLocked("ext")
	starter.recordProtocolStartLocked("ext", 1)
	starter.mu.Unlock()
}

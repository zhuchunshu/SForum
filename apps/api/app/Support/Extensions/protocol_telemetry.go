package extensionsruntime

import "time"

// protocolV1ShimContractID 与 apilts.ProtocolV1ContractID 对齐；此处常量避免
// Extensions 包对 APILTS 的硬依赖，便于单测注入 mock。
const protocolV1ShimContractID = "sforum.protocol.v1"

// ProtocolTelemetrySnapshot is the operator-visible transport/deprecation
// metric for one extension runtime.
type ProtocolTelemetrySnapshot struct {
	ProtocolVersion int
	Transport       string
	Deprecated      bool
	StartCount      uint64
	CallCount       uint64
	LastCallAt      *time.Time
}

// ProtocolTelemetrySource lets Manager merge transport metrics into status.
type ProtocolTelemetrySource interface {
	ProtocolTelemetry(extensionID string) ProtocolTelemetrySnapshot
}

type protocolTelemetry struct {
	version    int
	transport  string
	deprecated bool
	starts     uint64
	calls      uint64
	lastCallAt *time.Time
}

func (s *ProtocolStarter) ProtocolTelemetry(extensionID string) ProtocolTelemetrySnapshot {
	if s == nil {
		return ProtocolTelemetrySnapshot{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	metric := s.telemetry[extensionID]
	if metric == nil {
		return ProtocolTelemetrySnapshot{}
	}
	return ProtocolTelemetrySnapshot{
		ProtocolVersion: metric.version,
		Transport:       metric.transport,
		Deprecated:      metric.deprecated,
		StartCount:      metric.starts,
		CallCount:       metric.calls,
		LastCallAt:      cloneTelemetryTime(metric.lastCallAt),
	}
}

func (s *ProtocolStarter) recordProtocolStartLocked(extensionID string, version int) {
	transport := "net/rpc"
	if version == 2 {
		transport = "grpc"
	}
	metric := s.telemetry[extensionID]
	if metric == nil || metric.version != version {
		metric = &protocolTelemetry{version: version, transport: transport, deprecated: version == 1}
		s.telemetry[extensionID] = metric
	}
	metric.starts++
	// V1 冷启动也计入 LTS shim 使用，证明兼容窗口仍被占用。
	if version == 1 {
		s.recordProtocolShimLocked(protocolV1ShimContractID)
	}
}

func (s *ProtocolStarter) recordProtocolCallLocked(extensionID string) {
	metric := s.telemetry[extensionID]
	if metric == nil {
		return
	}
	now := time.Now().UTC()
	metric.calls++
	metric.lastCallAt = &now
	// 仅 V1 net/rpc 计入 APILTS；V2 gRPC 不得抬高弃用遥测。
	if metric.version == 1 {
		s.recordProtocolShimLocked(protocolV1ShimContractID)
	}
}

// recordProtocolShimLocked 写入可选 process-wide LTS 计数；调用方须已持有 s.mu
// 或保证 starter 单线程（record* 路径在持锁处调用）。
func (s *ProtocolStarter) recordProtocolShimLocked(contractID string) {
	if s == nil || s.shimTelemetry == nil {
		return
	}
	s.shimTelemetry.RecordShimCall(contractID)
}

func cloneTelemetryTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

var _ ProtocolTelemetrySource = (*ProtocolStarter)(nil)

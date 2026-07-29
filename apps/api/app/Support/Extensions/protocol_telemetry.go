package extensionsruntime

import "time"

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
	metric := s.telemetry[extensionID]
	if metric == nil || metric.version != version {
		metric = &protocolTelemetry{version: version, transport: "grpc"}
		s.telemetry[extensionID] = metric
	}
	metric.starts++
}

func (s *ProtocolStarter) recordProtocolCallLocked(extensionID string) {
	metric := s.telemetry[extensionID]
	if metric == nil {
		return
	}
	now := time.Now().UTC()
	metric.calls++
	metric.lastCallAt = &now
}

func cloneTelemetryTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

var _ ProtocolTelemetrySource = (*ProtocolStarter)(nil)

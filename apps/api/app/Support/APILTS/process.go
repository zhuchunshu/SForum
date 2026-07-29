package apilts

import "sync"

// processRegistry is the API/worker process-local LTS + shim telemetry store.
// Each process has its own counters (API vs standalone worker).
var (
	processMu       sync.Mutex
	processRegistry *Registry
)

// Process returns the process-local LTS registry, creating it on first use.
func Process() *Registry {
	processMu.Lock()
	defer processMu.Unlock()
	if processRegistry == nil {
		processRegistry = New()
	}
	return processRegistry
}

// ResetProcessForTest replaces the process registry (tests only).
func ResetProcessForTest(reg *Registry) {
	processMu.Lock()
	processRegistry = reg
	processMu.Unlock()
}

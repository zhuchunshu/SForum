// Package pluginv2 is the stable handwritten entry point for the generated
// SForum Host API and plugin runtime protocol v2 packages.
//
// Generated wire types live below gen/sforum. Runtime helpers in this package
// must preserve exact-artifact identity, host-owned authority, and gRPC
// deadlines; generated messages alone do not authorize an operation.
//
// P7 author surfaces for the six frozen families:
//
//   - hooks: HookRegistry + Server.WithHookRegistry (Host → plugin InvokeHook by id)
//   - services: ServiceRegistry + Host.List/Resolve/InvokeService
//   - providers: ProviderRegistry + Host.InvokeProvider (invoke only; no probe/send)
//   - jobs: JobRegistry + Host.EnqueueJob (Cancel/Watch remain wire-only unavailable)
//   - schedules: Manifest schedules[] + CoreSchedules catalog only; no ScheduleService helper
//   - commands: CommandRegistry (Host CLI → plugin InvokeCommand; no peer call RPC)
//
// Schema refs use id@version identity binding. Full JSON Schema value validation is
// host-owned when a schema registry is present; this package does not invent one.
package pluginv2

const (
	// HostAPIVersion is the first typed Host API contract.
	HostAPIVersion = "sforum.host/v2"
	// ProtocolMajor is the HashiCorp go-plugin protocol selected by V2 manifests.
	ProtocolMajor = 2
)

// Package pluginv2 is the stable handwritten entry point for the generated
// SForum Host API and plugin runtime protocol v2 packages.
//
// Generated wire types live below gen/sforum. Runtime helpers in this package
// must preserve exact-artifact identity, host-owned authority, and gRPC
// deadlines; generated messages alone do not authorize an operation.
package pluginv2

const (
	// HostAPIVersion is the first typed Host API contract.
	HostAPIVersion = "sforum.host/v2"
	// ProtocolMajor is the HashiCorp go-plugin protocol selected by V2 manifests.
	ProtocolMajor = 2
)

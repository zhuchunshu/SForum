// Package pluginbootstrap owns the process bootstrap ABI shared by the Host
// and executable plugin SDK. It is deliberately independent from the SForum
// application protocol and Host API versions negotiated after process start.
package pluginbootstrap

import (
	"strings"

	"github.com/hashicorp/go-plugin"
)

const (
	HashicorpGoPluginRPC = "hashicorp-go-plugin"

	// BootstrapABIV1 is the stable HashiCorp go-plugin process-control contract.
	// The historical "v1" cookie does not indicate support for SForum Protocol V1.
	BootstrapABIV1 = 1

	MagicCookieKeyV1   = "SFORUM_PLUGIN"
	MagicCookieValueV1 = "sforum-plugin-v1"

	ApplicationProtocolV2     = 2
	ApplicationProtocolV2Name = "sforum-plugin-v2"
)

// HandshakeV1 returns a copy so callers cannot mutate process-wide contract
// state. HashiCorp core protocol 1 negotiates SForum application protocol 2.
func HandshakeV1() plugin.HandshakeConfig {
	return plugin.HandshakeConfig{
		ProtocolVersion:  BootstrapABIV1,
		MagicCookieKey:   MagicCookieKeyV1,
		MagicCookieValue: MagicCookieValueV1,
	}
}

func SupportsProtocolV2(rpc string, protocolVersion int) bool {
	return strings.TrimSpace(rpc) == HashicorpGoPluginRPC && protocolVersion == ApplicationProtocolV2
}

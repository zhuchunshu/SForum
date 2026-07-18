package extensionsruntime_test

import "testing"

// TestP7JoinedProtocolV2ServiceProviderMatrix joins the real subprocess
// transport evidence for typed providers and plugin-to-plugin services.
func TestP7JoinedProtocolV2ServiceProviderMatrix(t *testing.T) {
	t.Run("typed_provider_declaration", TestProtocolV2InvokesVersionedProviderByExactTypedDeclaration)
	t.Run("plugin_b_host_plugin_a_provider", TestProtocolV2ProviderBrokerPluginBConsumesPluginA)
	t.Run("plugin_to_plugin_service", TestProtocolV2ServiceDiscoveryAcrossRealPluginProcesses)
}

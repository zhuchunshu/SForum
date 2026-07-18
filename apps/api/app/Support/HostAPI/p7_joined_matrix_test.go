package hostapi

import "testing"

// TestP7JoinedHostAPIServiceProviderMatrix joins Host-owned caller attestation
// and exact service dependency behavior without introducing a parallel harness.
func TestP7JoinedHostAPIServiceProviderMatrix(t *testing.T) {
	t.Run("service_exact_caller_and_provider_upgrade", TestServiceRuntimeDependencyTracksExactCallerAndProviderUpgrades)
	t.Run("service_capability_missing_and_ambiguous", TestServiceRuntimeCapabilityDependencyFailsClosedOnMissingAndAmbiguity)
	t.Run("service_dependency_denial_all_modes", TestProtocolV2ServiceDependencyDenialIsConsistentAcrossCallModes)
	t.Run("provider_attested_caller", TestProtocolV2ProviderBrokerUsesOnlyAttestedCallerIdentity)
	t.Run("provider_rejects_unattested_caller", TestProtocolV2ProviderBrokerRejectsUnattestedCaller)
	t.Run("provider_errors_are_redacted", TestProtocolV2ProviderBrokerDoesNotLeakInternalErrors)
}

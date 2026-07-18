package extensionmanifest

import "testing"

// TestP7JoinedManifestPolicyDependencyMatrix aggregates the frozen declaration
// and package-graph evidence used by the P7 execution-policy test row.
func TestP7JoinedManifestPolicyDependencyMatrix(t *testing.T) {
	t.Run("hook_defaults_and_failure_policy", TestManifestV3NormalizesHookExecutionAndFailureDefaults)
	t.Run("hook_timeout_bound", TestManifestV3RejectsHookTimeoutAboveHostDeadline)
	t.Run("provider_defaults", TestManifestV3NormalizesProviderSlotDefaults)
	t.Run("provider_timeout_and_fallback_bounds", TestManifestV3ProviderSlotRejectsUntypedAndUnboundedContracts)
	t.Run("dependency_order", TestResolvePackageGraphDeterministicOrder)
	t.Run("optional_dependency_fallback", TestResolvePackageGraphOptionalDependencyCanBeAbsentOrIncompatible)
	t.Run("version_conflict_and_cycle_failures", TestResolvePackageGraphFailures)
}

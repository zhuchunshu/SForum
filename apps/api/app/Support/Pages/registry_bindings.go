package pages

// 绑定（ProviderBinding）的纯函数集合：克隆与精确制品比对。
// 从 registry.go 抽出，保持注册表本体的方法集合聚焦在发布/切换语义上。

func cloneBindings(input map[string]ProviderBinding) map[string]ProviderBinding {
	result := make(map[string]ProviderBinding, len(input))
	for pageID, binding := range input {
		result[pageID] = cloneBinding(binding)
	}
	return result
}

func sameProviderArtifact(current, exact ProviderBinding) bool {
	return current.PageID == exact.PageID && current.ExtensionID == exact.ExtensionID &&
		current.ContributionID == exact.ContributionID && current.Version == exact.Version &&
		current.PackageDigest == exact.PackageDigest && current.TemplatePath == exact.TemplatePath &&
		current.ContractVersion == exact.ContractVersion
}

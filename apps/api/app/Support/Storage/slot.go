package storage

// Host provider slot for attachment object storage（F3.5）。
//
// v1 决策：具体驱动（local / OSS / COS / FTP / SFTP）仍由 core
// Support/Storage 适配器实现，运营通过 web_options.attachment.provider
// 选择；不强制拆成插件。slot 名称保留给未来插件覆盖路径与目录文档。
const ProviderSlot = "attachment.storage.provider"

// DriverCatalog 返回 core 内置驱动 id（与 options 校验一致）。
func DriverCatalog() []string {
	return []string{
		ProviderLocal,
		ProviderAliyunOSS,
		ProviderTencentCOS,
		ProviderFTP,
		ProviderSFTP,
	}
}

// IsKnownDriver 判断 provider 是否为 core 内置驱动。
func IsKnownDriver(provider string) bool {
	switch NormalizeProvider(provider) {
	case ProviderLocal, ProviderAliyunOSS, ProviderTencentCOS, ProviderFTP, ProviderSFTP:
		return true
	default:
		return false
	}
}

// NormalizeProvider 空白视为 local。
func NormalizeProvider(provider string) string {
	switch provider {
	case "", ProviderLocal:
		return ProviderLocal
	default:
		return provider
	}
}

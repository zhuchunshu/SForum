package storage

import "strings"

// Host provider slot for attachment object storage。
//
// E6：
//   - 槽位达到 mail 同级 L4–L6：插件可注册为候选，选中后 host 经 RPC 转发
//     Put/Open/Delete/Probe；密钥进 extension_settings。
//   - core 至少保留 local 作为 zero-config 与 restore 默认。
//   - 选择值：core 驱动 id，或 plugin:<extensionId>（见 selection.go）。
//   - 业务层仍只依赖 Adapter；插件细节封装在 Extensions.PluginStorageAdapter（E6.2）。
//
// NewAdapter 只接受 core 驱动 id；plugin: 选择由 Attachments.adapterForSettings 解析。
const ProviderSlot = "attachment.storage.provider"

// DriverCatalog 返回 core 内置驱动 id。Core 只保留 local 作为零配置回退；
// 远程存储必须经 attachment.storage.provider 插件槽实现。
func DriverCatalog() []string {
	return []string{ProviderLocal}
}

// IsKnownDriver 判断 provider 是否为 core 内置驱动。
// 不含 plugin: 选择；插件合法性在 E6.1 结合扩展目录校验。
func IsKnownDriver(provider string) bool {
	if IsPluginSelection(provider) {
		return false
	}
	return NormalizeProvider(provider) == ProviderLocal
}

// NormalizeProvider 空白视为 local。
// plugin: 前缀原样返回（插件选择应走 ParseSelection，勿当 core 驱动 id）。
func NormalizeProvider(provider string) string {
	provider = strings.TrimSpace(provider)
	if IsPluginSelection(provider) {
		return provider
	}
	switch provider {
	case "", ProviderLocal:
		return ProviderLocal
	default:
		return provider
	}
}

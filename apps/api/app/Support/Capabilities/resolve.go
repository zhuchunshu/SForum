package capabilities

import (
	"strings"
)

// ResolveInput 从 manifest 解析有效能力时的输入。
// 使用最小字段，避免与 ExtensionManifest 包循环依赖。
type ResolveInput struct {
	// Explicit 为 manifest.capabilities 显式声明。
	Explicit []string
	// HasJobs 是否声明了 jobs。
	HasJobs bool
	// HasSettings 是否声明了 settings。
	HasSettings bool
	// ProviderSlots 已声明的 provider slot。
	ProviderSlots []string
	// HasBackend 有 backend 入口时默认推断 host.api。
	HasBackend bool
}

// Resolve 合并显式声明与宿主推断，返回有效 key 与 implied 标记。
func Resolve(input ResolveInput) (keys []string, implied map[string]bool) {
	implied = map[string]bool{}
	set := NewSet(input.Explicit)

	// 显式声明优先；推断仅补充未写出的项。
	imply := func(key string) {
		if set.Has(key) {
			return
		}
		set[key] = struct{}{}
		implied[key] = true
	}

	if input.HasBackend {
		imply(HostAPI)
	}
	if input.HasSettings {
		imply(SettingsOwn)
	}
	if input.HasJobs {
		imply(JobsEnqueue)
	}
	for _, slot := range input.ProviderSlots {
		switch strings.TrimSpace(slot) {
		case "mail.provider":
			// SMTP 等邮件投递需要出站网络。
			imply(NetOutbound)
		}
	}

	return set.Keys(), implied
}

// GrantsFor 返回启用审查用的 Grant 列表。
func GrantsFor(input ResolveInput) []Grant {
	keys, implied := Resolve(input)
	return GrantsFromKeys(keys, implied)
}

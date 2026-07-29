package storage

import "strings"

// Candidate 是 attachment.storage.provider 的可选条目（E6.1）。
// Core 驱动与插件提供方共用此形状，便于 Admin 下拉与 OpenAPI。
type Candidate struct {
	// Value 写入 attachment.provider 的规范值（core id 或 plugin:<extensionId>）。
	Value string `json:"value"`
	// Kind 为 core 或 plugin。
	Kind SelectionKind `json:"kind"`
	// Label 运营可见名称。
	Label string `json:"label"`
	// ExtensionID 仅 plugin 候选非空。
	ExtensionID string `json:"extensionId,omitempty"`
	// SettingsPath 插件设置页相对路径提示（管理端 deep-link，可选）。
	SettingsPath string `json:"settingsPath,omitempty"`
	// Available false 表示已声明但当前不可用；E6.1 候选列表通常只含可用项。
	Available bool `json:"available"`
}

// CoreCandidates 返回内置驱动候选（始终 available）。
func CoreCandidates() []Candidate {
	// 标签与历史 Admin i18n 对齐；前端仍可再本地化。
	labels := map[string]string{
		ProviderLocal: "Local filesystem",
	}
	out := make([]Candidate, 0, len(DriverCatalog()))
	for _, id := range DriverCatalog() {
		label := labels[id]
		if label == "" {
			label = id
		}
		out = append(out, Candidate{
			Value:     id,
			Kind:      SelectionKindCore,
			Label:     label,
			Available: true,
		})
	}
	return out
}

// PluginCandidate 由扩展目录构造一条插件候选。
func PluginCandidate(extensionID, label, settingsPath string) Candidate {
	id := strings.TrimSpace(extensionID)
	if label == "" {
		label = id
	}
	return Candidate{
		Value:        FormatPluginSelection(id),
		Kind:         SelectionKindPlugin,
		Label:        label,
		ExtensionID:  id,
		SettingsPath: settingsPath,
		Available:    true,
	}
}

// MergeCandidates core 在前，插件按调用方顺序追加。
func MergeCandidates(core []Candidate, plugins []Candidate) []Candidate {
	out := make([]Candidate, 0, len(core)+len(plugins))
	out = append(out, core...)
	out = append(out, plugins...)
	return out
}

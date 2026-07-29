package storage

import (
	"strings"
)

// 选择值编码（E6.0 决策）：
//   - core 驱动：local（空白 → local）
//   - 插件：plugin:<extensionId>
// 见 knowledge/decisions/2026-07-12-attachment-storage-plugin-provider.md

// PluginSelectionPrefix 是 attachment.provider 中插件选择的固定前缀。
// 强制前缀避免扩展 id 与 core 驱动 id 碰撞。
const PluginSelectionPrefix = "plugin:"

// SelectionKind 区分 core 驱动与插件提供方。
type SelectionKind string

const (
	SelectionKindCore   SelectionKind = "core"
	SelectionKindPlugin SelectionKind = "plugin"
)

// Selection 是解析后的 attachment.provider 选择（E6.0 草图；E6.1 接线）。
type Selection struct {
	Kind SelectionKind
	// Driver 在 Kind=core 时为 NormalizeProvider 后的驱动 id。
	Driver string
	// ExtensionID 在 Kind=plugin 时为扩展 id（不含 plugin: 前缀）。
	ExtensionID string
	// Raw 为写入 web_options 前的规范字符串。
	Raw string
}

// FormatPluginSelection 将扩展 id 编码为 attachment.provider 可选值。
func FormatPluginSelection(extensionID string) string {
	id := strings.TrimSpace(extensionID)
	if id == "" {
		return ""
	}
	return PluginSelectionPrefix + id
}

// IsPluginSelection 判断原始 provider 字符串是否为插件选择。
func IsPluginSelection(raw string) bool {
	return strings.HasPrefix(strings.TrimSpace(raw), PluginSelectionPrefix)
}

// ParseSelection 解析 attachment.provider。
// 空白或未知非 plugin 值按 core 处理：NormalizeProvider 后交给 IsKnownDriver 校验（E6.1）。
func ParseSelection(raw string) Selection {
	raw = strings.TrimSpace(raw)
	if IsPluginSelection(raw) {
		extID := strings.TrimSpace(strings.TrimPrefix(raw, PluginSelectionPrefix))
		return Selection{
			Kind:        SelectionKindPlugin,
			ExtensionID: extID,
			Raw:         FormatPluginSelection(extID),
		}
	}
	driver := NormalizeProvider(raw)
	return Selection{
		Kind:   SelectionKindCore,
		Driver: driver,
		Raw:    driver,
	}
}

// IsCoreDriverSelection 为 true 表示应走 storage.NewAdapter 路径（非插件 RPC）。
func (s Selection) IsCoreDriverSelection() bool {
	return s.Kind == SelectionKindCore
}

// IsValidPluginSelection 插件选择至少需要非空 extension id（启用/槽位校验在 E6.1）。
func (s Selection) IsValidPluginSelection() bool {
	return s.Kind == SelectionKindPlugin && s.ExtensionID != ""
}

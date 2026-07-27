package extensions

import (
	"context"
	"strings"
)

// AuthProviderSettingsConfigured 判断扩展的必需设置是否齐全。
//
// 规则（Host 聚合 / 公开可用性门控共用）：
//   - 无 settings 字段：视为已配置（纯协议/无凭证插件）；
//   - 每个 manifest 字段必须有非空存储值（secret 含在内；API 掩码不影响本检查）；
//   - 读取失败向上返回，调用方 fail closed。
//
// 不在此处解析 Identity Registry：调用方传入已解析的 owner extension id。
func (s *SettingsService) AuthProviderSettingsConfigured(ctx context.Context, extensionID string) (bool, error) {
	if s == nil {
		return false, nil
	}
	extensionID = strings.TrimSpace(extensionID)
	if extensionID == "" {
		return false, nil
	}
	extension, err := s.store.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return false, err
	}
	if len(extension.Manifest.Settings) == 0 {
		return true, nil
	}
	values, err := s.listDecryptedSettings(ctx, extension)
	if err != nil {
		return false, err
	}
	for _, setting := range extension.Manifest.Settings {
		key := strings.TrimSpace(setting.Key)
		if key == "" {
			continue
		}
		if strings.TrimSpace(values[key]) == "" {
			return false, nil
		}
	}
	return true, nil
}

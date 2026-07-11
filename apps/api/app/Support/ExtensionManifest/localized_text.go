package extensionmanifest

import (
	"encoding/json"
	"strings"
)

// LocalizedText 支持插件多语言文案。
// JSON 可写纯字符串（默认文案），或 locale map：{"zh-CN":"...","en-US":"..."}。
type LocalizedText struct {
	Default  string            `json:"-"`
	ByLocale map[string]string `json:"-"`
}

func (t LocalizedText) IsEmpty() bool {
	if strings.TrimSpace(t.Default) != "" {
		return false
	}
	for _, value := range t.ByLocale {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func (t LocalizedText) Resolve(locale string) string {
	for _, candidate := range localeLookupCandidates(locale) {
		if value := strings.TrimSpace(t.ByLocale[candidate]); value != "" {
			return value
		}
	}
	return strings.TrimSpace(t.Default)
}

func (t LocalizedText) MarshalJSON() ([]byte, error) {
	if len(t.ByLocale) == 0 {
		return json.Marshal(t.Default)
	}
	out := make(map[string]string, len(t.ByLocale))
	for key, value := range t.ByLocale {
		out[key] = value
	}
	return json.Marshal(out)
}

func (t *LocalizedText) UnmarshalJSON(data []byte) error {
	if t == nil {
		return nil
	}
	*t = LocalizedText{}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		t.Default = strings.TrimSpace(asString)
		return nil
	}
	var asMap map[string]string
	if err := json.Unmarshal(data, &asMap); err != nil {
		return err
	}
	t.ByLocale = make(map[string]string, len(asMap))
	for key, value := range asMap {
		code := normalizeLocaleKey(key)
		value = strings.TrimSpace(value)
		if code == "" || value == "" {
			continue
		}
		t.ByLocale[code] = value
	}
	t.Default = defaultFromLocaleMap(t.ByLocale)
	if len(t.ByLocale) == 0 {
		t.ByLocale = nil
	}
	return nil
}

func (t LocalizedText) normalized() LocalizedText {
	defaultValue := strings.TrimSpace(t.Default)
	if len(t.ByLocale) == 0 {
		return LocalizedText{Default: defaultValue}
	}
	byLocale := make(map[string]string, len(t.ByLocale))
	for key, value := range t.ByLocale {
		code := normalizeLocaleKey(key)
		value = strings.TrimSpace(value)
		if code == "" || value == "" {
			continue
		}
		byLocale[code] = value
	}
	if defaultValue == "" {
		defaultValue = defaultFromLocaleMap(byLocale)
	}
	if len(byLocale) == 0 {
		return LocalizedText{Default: defaultValue}
	}
	return LocalizedText{Default: defaultValue, ByLocale: byLocale}
}

func defaultFromLocaleMap(byLocale map[string]string) string {
	for _, key := range []string{"en-US", "en"} {
		if value := strings.TrimSpace(byLocale[key]); value != "" {
			return value
		}
	}
	for _, value := range byLocale {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

// ResolvedSettingPresentation 是按 locale 解析后的设置展示字段。
type ResolvedSettingPresentation struct {
	Label       string
	Description string
	Placeholder string
	Group       string
	Options     []ResolvedSettingOption
}

type ResolvedSettingOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// ResolveSettingPresentation 将 manifest 设置声明解析为当前 locale 的展示字段。
func ResolveSettingPresentation(setting ManifestSetting, locale string) ResolvedSettingPresentation {
	setting.Label = setting.Label.normalized()
	setting.Description = setting.Description.normalized()
	setting.Placeholder = setting.Placeholder.normalized()
	setting.Group = setting.Group.normalized()
	options := make([]ResolvedSettingOption, 0, len(setting.Options))
	for _, option := range setting.Options {
		options = append(options, ResolvedSettingOption{
			Value:       strings.TrimSpace(option.Value),
			Label:       option.Label.normalized().Resolve(locale),
			Description: option.Description.normalized().Resolve(locale),
		})
	}
	return ResolvedSettingPresentation{
		Label:       setting.Label.Resolve(locale),
		Description: setting.Description.Resolve(locale),
		Placeholder: setting.Placeholder.Resolve(locale),
		Group:       setting.Group.Resolve(locale),
		Options:     options,
	}
}

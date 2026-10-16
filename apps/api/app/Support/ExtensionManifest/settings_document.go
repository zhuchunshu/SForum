package extensionmanifest

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const SettingsSchemaVersion = 1

const (
	SettingsUIModeSchema        = "schema"
	SettingsUIModeComponent     = "component"
	SettingsLayoutForm          = "form"
	SettingsLayoutTabs          = "tabs"
	SettingsActionProviderProbe = "provider_probe"
	// SettingsActionAIToolProbe 让插件在设置页提供一个「测试」按钮：宿主用草稿值
	// 起一个短命 runtime，再按 aiTools[] 声明调用插件自己的一个 AI 工具。
	SettingsActionAIToolProbe = "ai_tool_probe"
)

type ManifestSetting struct {
	Key              string        `json:"key"`
	Label            LocalizedText `json:"label"`
	Description      LocalizedText `json:"description,omitempty"`
	Type             string        `json:"type"`
	Default          string        `json:"default,omitempty"`
	Placeholder      LocalizedText `json:"placeholder,omitempty"`
	RecommendedValue string        `json:"recommendedValue,omitempty"`
	Required         bool          `json:"required,omitempty"`
	// Width 控制 Schema UI 控件横向占位：default（受限宽度）或 full（占满可用列宽）。
	// 省略时等价于 default。
	Width   string                  `json:"width,omitempty"`
	Group   LocalizedText           `json:"group,omitempty"`
	GroupID string                  `json:"groupId,omitempty"`
	Column  int                     `json:"column,omitempty"`
	Options []ManifestSettingOption `json:"options,omitempty"`
}

type ManifestSettingOption struct {
	Value       string        `json:"value"`
	Label       LocalizedText `json:"label"`
	Description LocalizedText `json:"description,omitempty"`
}

// SettingsDocument 是插件与主题共用的版本化设置展示契约。Fields 仍是存储语义的唯一来源。
type SettingsDocument struct {
	SchemaVersion int               `json:"schemaVersion"`
	UI            SettingsUI        `json:"ui"`
	Fields        []ManifestSetting `json:"fields"`
	Actions       []SettingsAction  `json:"actions,omitempty"`
	Explicit      bool              `json:"-"`
}

type SettingsUI struct {
	Mode      string             `json:"mode"`
	Layout    string             `json:"layout"`
	Tabs      []SettingsTab      `json:"tabs,omitempty"`
	Groups    []SettingsGroup    `json:"groups,omitempty"`
	Callouts  []SettingsCallout  `json:"callouts,omitempty"`
	Component *SettingsComponent `json:"component,omitempty"`
}

type SettingsTab struct {
	ID          string        `json:"id"`
	Label       LocalizedText `json:"label"`
	Description LocalizedText `json:"description,omitempty"`
	Groups      []string      `json:"groups,omitempty"`
}

type SettingsGroup struct {
	ID          string        `json:"id"`
	Label       LocalizedText `json:"label"`
	Description LocalizedText `json:"description,omitempty"`
	Columns     int           `json:"columns,omitempty"`
}

type SettingsCallout struct {
	ID        string        `json:"id"`
	Tone      string        `json:"tone,omitempty"`
	Title     LocalizedText `json:"title"`
	Body      LocalizedText `json:"body,omitempty"`
	LinkLabel LocalizedText `json:"linkLabel,omitempty"`
	LinkURL   string        `json:"linkUrl,omitempty"`
	Tab       string        `json:"tab,omitempty"`
	Group     string        `json:"group,omitempty"`
}

// SettingsComponent remains as a source-compatible name for the original
// settings-only contract. The artifact shape is now shared with admin pages.
type SettingsComponent = AdminComponent

type SettingsAction struct {
	ID             string        `json:"id"`
	Kind           string        `json:"kind"`
	Label          LocalizedText `json:"label"`
	Description    LocalizedText `json:"description,omitempty"`
	Placement      string        `json:"placement,omitempty"`
	UseDraftValues bool          `json:"useDraftValues,omitempty"`
	Fields         []string      `json:"fields,omitempty"`
	// Tool 是探测要派发到的 AI 工具声明 id（aiTools[].id），只被 kind
	// ai_tool_probe 使用：provider_probe 走 provider slot，不需要工具。
	Tool string `json:"tool,omitempty"`
}

func defaultSettingsDocument(fields []ManifestSetting) SettingsDocument {
	return SettingsDocument{
		SchemaVersion: SettingsSchemaVersion,
		UI:            SettingsUI{Mode: SettingsUIModeSchema, Layout: SettingsLayoutForm},
		Fields:        append([]ManifestSetting(nil), fields...),
	}
}

func decodeSettingsDocument(raw json.RawMessage) (SettingsDocument, error) {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return defaultSettingsDocument(nil), nil
	}
	if raw[0] == '[' {
		var fields []ManifestSetting
		if err := json.Unmarshal(raw, &fields); err != nil {
			return SettingsDocument{}, err
		}
		return defaultSettingsDocument(fields), nil
	}
	var document SettingsDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return SettingsDocument{}, err
	}
	document.Explicit = true
	return document, nil
}

func normalizeSettingsDocument(manifest *Manifest) {
	document := manifest.SettingsDocument
	if document.SchemaVersion == 0 {
		document = defaultSettingsDocument(manifest.Settings)
	}
	if document.SchemaVersion == 0 {
		document.SchemaVersion = SettingsSchemaVersion
	}
	document.UI.Mode = strings.ToLower(strings.TrimSpace(document.UI.Mode))
	if document.UI.Mode == "" {
		document.UI.Mode = SettingsUIModeSchema
	}
	document.UI.Layout = strings.ToLower(strings.TrimSpace(document.UI.Layout))
	if document.UI.Layout == "" {
		document.UI.Layout = SettingsLayoutForm
	}
	for index := range document.UI.Tabs {
		tab := &document.UI.Tabs[index]
		tab.ID = NormalizeID(tab.ID)
		tab.Label = tab.Label.normalized()
		tab.Description = tab.Description.normalized()
		for groupIndex := range tab.Groups {
			tab.Groups[groupIndex] = NormalizeID(tab.Groups[groupIndex])
		}
	}
	for index := range document.UI.Groups {
		group := &document.UI.Groups[index]
		group.ID = NormalizeID(group.ID)
		group.Label = group.Label.normalized()
		group.Description = group.Description.normalized()
	}
	for fieldIndex := range manifest.Settings {
		field := &manifest.Settings[fieldIndex]
		if field.GroupID != "" || field.Group.IsEmpty() {
			continue
		}
		for _, group := range document.UI.Groups {
			if localizedTextMatches(field.Group, group.Label) {
				field.GroupID = group.ID
				break
			}
		}
	}
	for index := range document.UI.Callouts {
		callout := &document.UI.Callouts[index]
		callout.ID = NormalizeID(callout.ID)
		callout.Tone = strings.ToLower(strings.TrimSpace(callout.Tone))
		callout.Title = callout.Title.normalized()
		callout.Body = callout.Body.normalized()
		callout.LinkLabel = callout.LinkLabel.normalized()
		callout.LinkURL = strings.TrimSpace(callout.LinkURL)
		callout.Tab = NormalizeID(callout.Tab)
		callout.Group = NormalizeID(callout.Group)
	}
	if document.UI.Component != nil {
		normalizeAdminComponent(document.UI.Component)
	}
	for index := range document.Actions {
		action := &document.Actions[index]
		action.ID = NormalizeID(action.ID)
		action.Kind = strings.ToLower(strings.TrimSpace(action.Kind))
		action.Label = action.Label.normalized()
		action.Description = action.Description.normalized()
		action.Placement = strings.ToLower(strings.TrimSpace(action.Placement))
		if action.Placement == "" {
			action.Placement = "footer"
		}
		for fieldIndex := range action.Fields {
			action.Fields[fieldIndex] = strings.TrimSpace(action.Fields[fieldIndex])
		}
		sort.Strings(action.Fields)
		action.Tool = strings.TrimSpace(action.Tool)
	}
	document.Fields = append([]ManifestSetting(nil), manifest.Settings...)
	manifest.SettingsDocument = document
}

func localizedTextMatches(left, right LocalizedText) bool {
	left = left.normalized()
	right = right.normalized()
	if left.Default != "" && left.Default == right.Default {
		return true
	}
	for locale, value := range left.ByLocale {
		if value != "" && value == right.ByLocale[locale] {
			return true
		}
	}
	return false
}

func validateSettingsDocument(manifest Manifest) error {
	document := manifest.SettingsDocument
	if document.SchemaVersion != SettingsSchemaVersion {
		return ErrInvalidManifest
	}
	if document.UI.Mode != SettingsUIModeSchema && document.UI.Mode != SettingsUIModeComponent {
		return ErrInvalidManifest
	}
	if document.UI.Layout != SettingsLayoutForm && document.UI.Layout != SettingsLayoutTabs {
		return ErrInvalidManifest
	}
	groups := map[string]struct{}{}
	for _, group := range document.UI.Groups {
		if !adminComponentIDPattern.MatchString(group.ID) || group.Label.IsEmpty() || (group.Columns != 0 && group.Columns != 1 && group.Columns != 2) {
			return ErrInvalidManifest
		}
		if _, exists := groups[group.ID]; exists {
			return ErrInvalidManifest
		}
		groups[group.ID] = struct{}{}
	}
	tabIDs := map[string]struct{}{}
	assignedGroups := map[string]string{}
	for _, tab := range document.UI.Tabs {
		if !adminComponentIDPattern.MatchString(tab.ID) || tab.Label.IsEmpty() {
			return ErrInvalidManifest
		}
		if _, exists := tabIDs[tab.ID]; exists {
			return ErrInvalidManifest
		}
		tabIDs[tab.ID] = struct{}{}
		for _, group := range tab.Groups {
			if _, exists := groups[group]; !exists || assignedGroups[group] != "" {
				return ErrInvalidManifest
			}
			assignedGroups[group] = tab.ID
		}
	}
	if document.UI.Layout == SettingsLayoutTabs && len(document.UI.Tabs) == 0 {
		return ErrInvalidManifest
	}
	for _, callout := range document.UI.Callouts {
		if !adminComponentIDPattern.MatchString(callout.ID) || callout.Title.IsEmpty() {
			return ErrInvalidManifest
		}
		if (callout.LinkURL == "") != callout.LinkLabel.IsEmpty() || (callout.LinkURL != "" && !validHTTPURL(callout.LinkURL)) {
			return ErrInvalidManifest
		}
		if callout.Tab != "" {
			if _, exists := tabIDs[callout.Tab]; !exists {
				return ErrInvalidManifest
			}
		}
		if callout.Group != "" {
			if _, exists := groups[callout.Group]; !exists {
				return ErrInvalidManifest
			}
		}
	}
	fieldKeys := map[string]struct{}{}
	for _, field := range document.Fields {
		fieldKeys[field.Key] = struct{}{}
		if field.GroupID != "" {
			if _, exists := groups[field.GroupID]; !exists {
				return ErrInvalidManifest
			}
		}
		if field.Column < 0 || field.Column > 2 {
			return ErrInvalidManifest
		}
	}
	actionIDs := map[string]struct{}{}
	for _, action := range document.Actions {
		if !adminComponentIDPattern.MatchString(action.ID) || action.Label.IsEmpty() ||
			(action.Kind != SettingsActionProviderProbe && action.Kind != SettingsActionAIToolProbe) ||
			(action.Placement != "header" && action.Placement != "footer") {
			return ErrInvalidManifest
		}
		if _, exists := actionIDs[action.ID]; exists {
			return ErrInvalidManifest
		}
		actionIDs[action.ID] = struct{}{}
		for _, key := range action.Fields {
			if _, exists := fieldKeys[key]; !exists {
				return ErrInvalidManifest
			}
		}
		// AI 工具探测必须点名一个属于自己的已声明工具，并至少声明一个草稿字段：
		// 没有字段就没有草稿值可测，探测也就失去意义。
		if action.Kind == SettingsActionAIToolProbe && !settingsDocumentDeclaresAITool(manifest, action.Tool) {
			return ErrInvalidManifest
		}
		if action.Kind == SettingsActionAIToolProbe && len(action.Fields) == 0 {
			return ErrInvalidManifest
		}
	}
	// 任何设置页动作都需要可执行的插件后端；provider slot 只有 provider_probe
	// 才要求：AI 工具探测需要的是一个已声明的工具，而不是 provider 槽位。
	if len(document.Actions) > 0 && (manifest.Type != TypePlugin || manifest.Backend.Entry == "") {
		return ErrInvalidManifest
	}
	if len(document.Actions) > 0 && len(manifest.Providers) == 0 && settingsDocumentHasProviderProbe(document.Actions) {
		return ErrInvalidManifest
	}
	if document.UI.Mode == SettingsUIModeComponent {
		component := document.UI.Component
		if !validAdminComponent(component) || len(document.Fields) == 0 {
			return ErrInvalidManifest
		}
	} else if document.UI.Component != nil {
		return ErrInvalidManifest
	}
	return nil
}

// settingsDocumentDeclaresAITool 判断 tool 是否为本插件声明的 AI 工具 id：
// 必须落在插件命名空间下，且逐字匹配 manifest.aiTools[].id。
func settingsDocumentDeclaresAITool(manifest Manifest, tool string) bool {
	tool = strings.TrimSpace(tool)
	if !IsOwnedID(manifest.ID, tool) {
		return false
	}
	for _, declaration := range manifest.AITools {
		if declaration.ID == tool {
			return true
		}
	}
	return false
}

func settingsDocumentHasProviderProbe(actions []SettingsAction) bool {
	for _, action := range actions {
		if action.Kind == SettingsActionProviderProbe {
			return true
		}
	}
	return false
}

func (document SettingsDocument) canonicalValue(fields []ManifestSetting) any {
	if !document.Explicit {
		return fields
	}
	copy := document
	copy.Fields = fields
	return copy
}

func mergeSettingsDocuments(documents []SettingsDocument) (SettingsDocument, error) {
	result := defaultSettingsDocument(nil)
	explicit := false
	fields := make([]ManifestSetting, 0)
	for _, document := range documents {
		fields = append(fields, document.Fields...)
		if document.Explicit {
			if explicit {
				return SettingsDocument{}, fmt.Errorf("%w: multiple settings documents", ErrInvalidManifest)
			}
			explicit = true
			result = document
		}
	}
	result.Fields = fields
	result.Explicit = explicit
	return result, nil
}

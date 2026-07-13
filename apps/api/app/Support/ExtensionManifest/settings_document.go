package extensionmanifest

import (
	"encoding/json"
	"fmt"
	"path"
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
)

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
	ID    string        `json:"id"`
	Tone  string        `json:"tone,omitempty"`
	Title LocalizedText `json:"title"`
	Body  LocalizedText `json:"body,omitempty"`
	Tab   string        `json:"tab,omitempty"`
	Group string        `json:"group,omitempty"`
}

type SettingsComponent struct {
	ID         string `json:"id"`
	APIVersion int    `json:"apiVersion"`
	Entry      string `json:"entry,omitempty"`
	CSS        string `json:"css,omitempty"`
}

type SettingsAction struct {
	ID             string        `json:"id"`
	Kind           string        `json:"kind"`
	Label          LocalizedText `json:"label"`
	Description    LocalizedText `json:"description,omitempty"`
	Placement      string        `json:"placement,omitempty"`
	UseDraftValues bool          `json:"useDraftValues,omitempty"`
	Fields         []string      `json:"fields,omitempty"`
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
		callout.Tab = NormalizeID(callout.Tab)
		callout.Group = NormalizeID(callout.Group)
	}
	if document.UI.Component != nil {
		component := document.UI.Component
		component.ID = NormalizeID(component.ID)
		component.Entry = normalizeAdminRelativePath(component.Entry)
		component.CSS = normalizeAdminRelativePath(component.CSS)
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
		if !adminComponentIDPattern.MatchString(action.ID) || action.Label.IsEmpty() || action.Kind != SettingsActionProviderProbe || (action.Placement != "header" && action.Placement != "footer") {
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
	}
	if len(document.Actions) > 0 && (manifest.Type != TypePlugin || manifest.Backend.Entry == "" || len(manifest.Providers) == 0) {
		return ErrInvalidManifest
	}
	if document.UI.Mode == SettingsUIModeComponent {
		component := document.UI.Component
		if component == nil || !adminComponentIDPattern.MatchString(component.ID) || component.APIVersion != AdminMicroFrontendAPIVersion || len(document.Fields) == 0 {
			return ErrInvalidManifest
		}
		if component.Entry != "" {
			if !safeAdminRelativePath(component.Entry) || path.Ext(component.Entry) != ".mjs" || !strings.HasPrefix(component.Entry, "frontend/admin/dist/") {
				return ErrInvalidManifest
			}
			if component.CSS != "" && (!safeAdminRelativePath(component.CSS) || path.Ext(component.CSS) != ".css" || !strings.HasPrefix(component.CSS, "frontend/admin/dist/")) {
				return ErrInvalidManifest
			}
		}
	} else if document.UI.Component != nil {
		return ErrInvalidManifest
	}
	return nil
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

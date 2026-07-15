package themecompiler

import (
	"fmt"
	htmltemplate "html/template"
	"strings"
)

func cloneBindings(input Bindings) Bindings {
	output := Bindings{
		BindingRevision: strings.Clone(input.BindingRevision),
		SiteName:        strings.Clone(input.SiteName),
		Assets:          cloneStringMap(input.Assets),
		Routes:          cloneStringMap(input.Routes),
		Translations:    make(map[string]map[string]string, len(input.Translations)),
		PageViewModels:  make(map[string]PageTemplateBinding, len(input.PageViewModels)),
		Islands:         make(map[string]IslandBinding, len(input.Islands)),
	}
	for locale, messages := range input.Translations {
		output.Translations[strings.Clone(locale)] = cloneStringMap(messages)
	}
	for name, binding := range input.PageViewModels {
		output.PageViewModels[strings.Clone(name)] = PageTemplateBinding{
			PageID: strings.Clone(binding.PageID), SchemaVersion: strings.Clone(binding.SchemaVersion),
		}
	}
	for tag, binding := range input.Islands {
		output.Islands[strings.Clone(tag)] = IslandBinding{
			ComponentID:   strings.Clone(binding.ComponentID),
			Props:         append([]IslandPropContract(nil), binding.Props...),
			AllowFallback: binding.AllowFallback,
		}
	}
	return output
}

func cloneStringMap(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[strings.Clone(key)] = strings.Clone(value)
	}
	return output
}

func restrictedFuncMap(bindings Bindings) htmltemplate.FuncMap {
	return htmltemplate.FuncMap{
		"safeHTML": renderSafeHTML,
		"siteName": func() string { return bindings.SiteName },
		"asset": func(id string) (string, error) {
			return boundValue(bindings.Assets, "asset", id)
		},
		"route": func(id string) (string, error) {
			return boundValue(bindings.Routes, "route", id)
		},
		"i18n": func(locale, key string) (string, error) {
			messages, ok := bindings.Translations[locale]
			if !ok {
				return "", fmt.Errorf("%w: locale %q", ErrHelperValueMissing, locale)
			}
			return boundValue(messages, "translation", key)
		},
	}
}

func boundValue(values map[string]string, kind, key string) (string, error) {
	value, ok := values[key]
	if !ok {
		return "", fmt.Errorf("%w: %s %q", ErrHelperValueMissing, kind, key)
	}
	return value, nil
}

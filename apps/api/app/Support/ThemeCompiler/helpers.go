package themecompiler

import (
	"fmt"
	htmltemplate "html/template"
	"regexp"
	"strings"
)

// hostIconNamePattern 是宿主认可的图标名语法，必须与 apps/web 的
// HOST_ICON_NAME_PATTERN、贡献点校验以及管理端 surface 校验保持一致。
var hostIconNamePattern = regexp.MustCompile(`^i-(lucide|tabler)-[a-z0-9]+(?:-[a-z0-9]+)*$`)

// iconNameEchoPattern 只用于把非法名称回显进注释，替换掉一切可能提前闭合注释的字符。
var iconNameEchoPattern = regexp.MustCompile(`[^A-Za-z0-9._:-]`)

const iconNameEchoLimit = 80

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
		"icon": renderHostIcon,
	}
}

// renderHostIcon 输出一个由宿主客户端运行时填充的图标占位元素。
//
// 刻意不在服务端内联 SVG：那会把整个 lucide/tabler 图标集编进 Go 二进制。
// 图标是装饰性的（aria-hidden），因此禁用 JavaScript 时只丢失装饰，不影响
// 页面内容、可访问性与爬虫渲染。名称非法时输出一条 HTML 注释而不是让整页
// 渲染失败——模板里一个图标拼错不该变成 500，但作者仍需在源码里看到它。
func renderHostIcon(name string) htmltemplate.HTML {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	if !hostIconNamePattern.MatchString(trimmed) {
		echo := iconNameEchoPattern.ReplaceAllString(trimmed, "?")
		if len(echo) > iconNameEchoLimit {
			echo = echo[:iconNameEchoLimit]
		}
		return htmltemplate.HTML(`<!-- sforum.icon: unsupported name "` + echo + `" -->`)
	}
	// 名称已被语法限定为 `i-(lucide|tabler)-[a-z0-9-]+`，可直接进属性。
	return htmltemplate.HTML(`<span class="sf-host-icon" data-sf-icon="` + trimmed + `" aria-hidden="true"></span>`)
}

func boundValue(values map[string]string, kind, key string) (string, error) {
	value, ok := values[key]
	if !ok {
		return "", fmt.Errorf("%w: %s %q", ErrHelperValueMissing, kind, key)
	}
	return value, nil
}

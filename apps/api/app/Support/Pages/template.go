package pages

import (
	"fmt"
	"html"
	"os"
	"regexp"
)

// 极简 L1 模板：允许 HTML + {{var}} + <sf-*> 宿主岛标签。
// 禁止 script/iframe/on* 事件与 javascript: URL。

var (
	tplScript   = regexp.MustCompile(`(?is)<script\b`)
	tplIframe   = regexp.MustCompile(`(?is)<iframe\b`)
	tplObject   = regexp.MustCompile(`(?is)<object\b`)
	tplEmbed    = regexp.MustCompile(`(?is)<embed\b`)
	tplOnAttr   = regexp.MustCompile(`(?i)\son[a-z]+\s*=`)
	tplJSHref   = regexp.MustCompile(`(?i)(href|src)\s*=\s*['"]\s*javascript:`)
	tplMustache = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)
)

// ValidateTemplate 沙箱检查 L1 模板源。
func ValidateTemplate(src string) error {
	if tplScript.MatchString(src) || tplIframe.MatchString(src) || tplObject.MatchString(src) || tplEmbed.MatchString(src) {
		return fmt.Errorf("pages: template rejects script/iframe/object/embed")
	}
	if tplOnAttr.MatchString(src) {
		return fmt.Errorf("pages: template rejects inline event handlers")
	}
	if tplJSHref.MatchString(src) {
		return fmt.Errorf("pages: template rejects javascript: urls")
	}
	return nil
}

// LoadTemplate 从主题包读取并校验模板。
func LoadTemplate(packageRoot, rel string) (string, error) {
	full, err := ResolveThemeAsset(packageRoot, rel)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	src := string(raw)
	if err := ValidateTemplate(src); err != nil {
		return "", err
	}
	return src, nil
}

// RenderTemplate 替换 {{key}} 为转义后的字符串值；未知 key 置空。
// <sf-*> 标签原样保留，由前端岛注册表挂载。
func RenderTemplate(src string, vars map[string]string) (string, error) {
	if err := ValidateTemplate(src); err != nil {
		return "", err
	}
	return tplMustache.ReplaceAllStringFunc(src, func(m string) string {
		sub := tplMustache.FindStringSubmatch(m)
		if len(sub) < 2 {
			return ""
		}
		key := sub[1]
		if vars == nil {
			return ""
		}
		return html.EscapeString(vars[key])
	}), nil
}

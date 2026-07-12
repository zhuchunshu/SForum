package pages

import (
	"fmt"
	"regexp"
	"strings"
)

// 拒绝危险 CSS 构造（L0 skin 校验）。非完整 CSS 解析器，覆盖常见攻击面。
var (
	cssExpression = regexp.MustCompile(`(?i)expression\s*\(`)
	cssJSURL      = regexp.MustCompile(`(?i)url\s*\(\s*['"]?\s*javascript:`)
	cssImportURL  = regexp.MustCompile(`(?i)@import\s+(url\s*\()?\s*['"]?\s*https?:`)
	cssBehavior   = regexp.MustCompile(`(?i)behavior\s*:`)
	cssMozBinding = regexp.MustCompile(`(?i)-moz-binding\s*:`)
)

// ValidateCSS 校验皮肤 CSS 是否可接受；失败返回错误原因。
func ValidateCSS(css string) error {
	if cssExpression.MatchString(css) {
		return fmt.Errorf("pages: css rejects expression()")
	}
	if cssJSURL.MatchString(css) {
		return fmt.Errorf("pages: css rejects javascript: urls")
	}
	if cssImportURL.MatchString(css) {
		return fmt.Errorf("pages: css rejects external @import")
	}
	if cssBehavior.MatchString(css) {
		return fmt.Errorf("pages: css rejects behavior")
	}
	if cssMozBinding.MatchString(css) {
		return fmt.Errorf("pages: css rejects -moz-binding")
	}
	// 拒绝 <script 混入
	if strings.Contains(strings.ToLower(css), "<script") {
		return fmt.Errorf("pages: css rejects embedded script tags")
	}
	return nil
}

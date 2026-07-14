package themecompiler

import (
	"fmt"
	htmltemplate "html/template"
	"strings"
)

// SafeHTML is an opaque value produced by a Host-owned ViewModel assembler.
// It is intentionally not a string alias, so template data and plugin JSON
// cannot cast ordinary text into trusted markup.
type SafeHTML struct {
	value string
}

// NewSafeHTMLFromSanitized marks HTML that has already passed a core-owned
// sanitizer. This function does not sanitize; callers must remain Host-owned.
func NewSafeHTMLFromSanitized(value string) SafeHTML {
	return SafeHTML{value: strings.Clone(value)}
}

func renderSafeHTML(value any) (htmltemplate.HTML, error) {
	trusted, ok := value.(SafeHTML)
	if !ok {
		return "", fmt.Errorf("%w: got %T", ErrSafeHTMLRequired, value)
	}
	return htmltemplate.HTML(trusted.value), nil
}

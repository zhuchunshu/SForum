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
	// Rich-content sanitizers cannot mint Host component authority. Islands
	// must be static reviewed template declarations with compiler bindings.
	if strings.Contains(strings.ToLower(trusted.value), "<sf-") {
		return "", fmt.Errorf("%w: SafeHTML contains a host island", ErrInvalidIsland)
	}
	return htmltemplate.HTML(trusted.value), nil
}

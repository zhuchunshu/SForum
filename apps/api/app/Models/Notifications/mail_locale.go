package notifications

import (
	"context"
	"strings"
)

// MailLocaleResolver supplies the live site fallback before notification data
// becomes immutable delivery data.
type MailLocaleResolver interface {
	DefaultMailLocale(context.Context) (string, error)
}

func resolveMailLocale(ctx context.Context, accountLocale string, resolver MailLocaleResolver) string {
	if locale := strings.TrimSpace(accountLocale); locale != "" {
		return locale
	}
	if resolver != nil {
		if locale, err := resolver.DefaultMailLocale(ctx); err == nil && strings.TrimSpace(locale) != "" {
			return locale
		}
	}
	return "zh-CN"
}

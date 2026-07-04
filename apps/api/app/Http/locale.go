package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Localization"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

const requestLocaleKey = "sforum.locale"

func localeMiddleware(cfg config.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		locale := localization.NegotiateAcceptLanguage(
			c.Get("Accept-Language"),
			cfg.SupportedLocales,
			cfg.AppLocale,
		)
		c.Locals(requestLocaleKey, locale)
		return c.Next()
	}
}

func Locale(c fiber.Ctx) string {
	if locale, ok := c.Locals(requestLocaleKey).(string); ok && locale != "" {
		return locale
	}
	return localization.DefaultLocale
}

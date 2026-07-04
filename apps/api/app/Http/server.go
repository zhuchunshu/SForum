package http

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type healthResponse struct {
	Name             string    `json:"name"`
	Status           string    `json:"status"`
	Environment      string    `json:"environment"`
	Locale           string    `json:"locale"`
	SupportedLocales []string  `json:"supportedLocales"`
	Time             time.Time `json:"time"`
}

type RouteProvider interface {
	RegisterRoutes(api fiber.Router)
}

type Dependencies struct {
	RouteProviders []RouteProvider
	Options        *options.Service
}

func NewApp(cfg config.Config, logger *slog.Logger, deps Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ErrorHandler: errorHandler(logger),
	})
	app.Hooks().OnPreStartupMessage(func(sm *fiber.PreStartupMessageData) error {
		sm.BannerHeader = sforumStartupBanner
		return nil
	})

	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(localeMiddleware(cfg, deps.Options))

	registerRoutes(app, cfg, deps)

	return app
}

func registerRoutes(app *fiber.App, cfg config.Config, deps Dependencies) {
	api := app.Group("/api/v1")

	for _, provider := range deps.RouteProviders {
		if provider != nil {
			provider.RegisterRoutes(api)
		}
	}

	api.Get("/health", func(c fiber.Ctx) error {
		settings := runtimeSettings(c.Context(), cfg, deps.Options)
		return OK(c, healthResponse{
			Name:             settings.SiteName,
			Status:           "ok",
			Environment:      cfg.AppEnv,
			Locale:           settings.DefaultLocale,
			SupportedLocales: settings.SupportedLocales,
			Time:             time.Now().UTC(),
		})
	})
}

func runtimeSettings(ctx context.Context, cfg config.Config, service *options.Service) options.RuntimeSettings {
	if service != nil {
		if settings, err := service.RuntimeSettings(ctx); err == nil {
			return settings
		}
	}
	return fallbackRuntimeSettings(cfg)
}

func fallbackRuntimeSettings(cfg config.Config) options.RuntimeSettings {
	return options.RuntimeSettings{
		SiteName:                  cfg.AppName,
		SiteURL:                   "",
		DefaultLocale:             cfg.AppLocale,
		SupportedLocales:          cfg.SupportedLocales,
		HumanVerificationProvider: cfg.HumanVerificationProvider,
	}
}

const sforumStartupBanner = `
   _____ ______
  / ___// ____/___  _______  ______ ___
  \__ \/ /_  / __ \/ ___/ / / / _  _  \
 ___/ / __/ / /_/ / /  / /_/ / / / / / /
/____/_/    \____/_/   \__,_/_/ /_/ /_/    SForum API
--------------------------------------------------`

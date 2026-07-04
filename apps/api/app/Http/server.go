package http

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

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
}

func NewApp(cfg config.Config, logger *slog.Logger, deps Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ErrorHandler: errorHandler(logger),
	})

	app.Use(requestid.New())
	app.Use(recover.New())

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
		return c.JSON(healthResponse{
			Name:             cfg.AppName,
			Status:           "ok",
			Environment:      cfg.AppEnv,
			Locale:           cfg.AppLocale,
			SupportedLocales: cfg.SupportedLocales,
			Time:             time.Now().UTC(),
		})
	})
}

package http

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"github.com/inkedus/sforum/apps/api/internal/config"
)

type healthResponse struct {
	Name             string    `json:"name"`
	Status           string    `json:"status"`
	Environment      string    `json:"environment"`
	Locale           string    `json:"locale"`
	SupportedLocales []string  `json:"supportedLocales"`
	Time             time.Time `json:"time"`
}

func NewApp(cfg config.Config, logger *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ErrorHandler: errorHandler(logger),
	})

	app.Use(requestid.New())
	app.Use(recover.New())

	registerRoutes(app, cfg)

	return app
}

func registerRoutes(app *fiber.App, cfg config.Config) {
	api := app.Group("/api/v1")

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

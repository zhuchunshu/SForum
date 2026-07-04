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

type Dependencies struct {
	IdentityHandler interface {
		RegisterRoutes(api fiber.Router)
	}
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

	if deps.IdentityHandler != nil {
		deps.IdentityHandler.RegisterRoutes(api)
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

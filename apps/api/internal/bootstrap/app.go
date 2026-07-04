package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	"github.com/inkedus/sforum/apps/api/internal/config"
	httpserver "github.com/inkedus/sforum/apps/api/internal/http"
	"github.com/inkedus/sforum/apps/api/internal/modules/identity"
	"github.com/inkedus/sforum/apps/api/internal/platform/postgres"
	redisplatform "github.com/inkedus/sforum/apps/api/internal/platform/redis"
)

type API struct {
	App  *fiber.App
	Addr string

	closeOnce sync.Once
	close     func()
}

func NewAPI(ctx context.Context, cfg config.Config, logger *slog.Logger) (*API, error) {
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres setup failed: %w", err)
	}

	redisStorage, err := redisplatform.NewStorage(cfg.RedisAddr)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("redis session storage setup failed: %w", err)
	}

	sessionStore := session.NewStore(session.Config{
		Storage:        redisStorage,
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
	})
	identityProvider := identity.NewProvider(identity.NewPostgresStore(pool), sessionStore)

	app := httpserver.NewApp(cfg, logger, httpserver.Dependencies{
		RouteProviders: []httpserver.RouteProvider{identityProvider},
	})

	return &API{
		App:  app,
		Addr: apiAddress(cfg),
		close: func() {
			if err := redisStorage.Close(); err != nil {
				logger.Warn("redis session storage close failed", "error", err)
			}
			pool.Close()
		},
	}, nil
}

func (api *API) Close() {
	if api == nil {
		return
	}
	api.closeOnce.Do(func() {
		if api.close != nil {
			api.close()
		}
	})
}

func apiAddress(cfg config.Config) string {
	return fmt.Sprintf("%s:%s", cfg.HTTPHost, cfg.HTTPPort)
}

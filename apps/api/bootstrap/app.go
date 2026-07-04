package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	httpserver "github.com/zhuchunshu/sforum/apps/api/app/Http"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	"github.com/zhuchunshu/sforum/apps/api/app/Providers"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Postgres"
	redisplatform "github.com/zhuchunshu/sforum/apps/api/app/Support/Redis"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type API struct {
	App  *fiber.App
	Addr string

	closeOnce sync.Once
	close     func()
}

func NewAPI(ctx context.Context, cfg config.Config, logger *slog.Logger) (*API, error) {
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns)
	if err != nil {
		return nil, fmt.Errorf("postgres setup failed: %w", err)
	}

	redisStorage, err := redisplatform.NewStorage(cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("redis session storage setup failed: %w", err)
	}
	humanVerifyStore := humanverify.NewRedisStore(humanverify.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword))
	humanVerifier, err := newHumanVerifyService(cfg, humanVerifyStore)
	if err != nil {
		if closeErr := humanVerifyStore.Close(); closeErr != nil {
			logger.Warn("human verification redis close failed", "error", closeErr)
		}
		if closeErr := redisStorage.Close(); closeErr != nil {
			logger.Warn("redis session storage close failed", "error", closeErr)
		}
		pool.Close()
		return nil, err
	}

	sessionStore := session.NewStore(session.Config{
		Storage:        redisStorage,
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
	})
	identityProvider := providers.NewIdentityProviderWithVerifier(identity.NewPostgresStore(pool), sessionStore, humanVerifier)

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
			if err := humanVerifyStore.Close(); err != nil {
				logger.Warn("human verification redis close failed", "error", err)
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

func newHumanVerifyService(cfg config.Config, store humanverify.Store) (*humanverify.Service, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.HumanVerificationProvider)) {
	case "", humanverify.ProviderDisabled:
		return humanverify.NewDisabledService(), nil
	case humanverify.ProviderAltcha:
		return humanverify.NewService(
			humanverify.ServiceConfig{
				Enabled:         true,
				ChallengeTTL:    cfg.AltchaChallengeTTL,
				RateLimit:       60,
				RateLimitWindow: time.Minute,
			},
			humanverify.NewAltchaProvider(humanverify.AltchaConfig{
				Secret:       cfg.AltchaSecret,
				Cost:         cfg.AltchaCost,
				ChallengeTTL: cfg.AltchaChallengeTTL,
			}),
			store,
		), nil
	default:
		return nil, fmt.Errorf("unsupported human verification provider %q", cfg.HumanVerificationProvider)
	}
}

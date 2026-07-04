package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3/middleware/session"

	"github.com/inkedus/sforum/apps/api/internal/config"
	httpserver "github.com/inkedus/sforum/apps/api/internal/http"
	"github.com/inkedus/sforum/apps/api/internal/modules/identity"
	"github.com/inkedus/sforum/apps/api/internal/platform/postgres"
	redisplatform "github.com/inkedus/sforum/apps/api/internal/platform/redis"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("postgres setup failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisStorage, err := redisplatform.NewStorage(cfg.RedisAddr)
	if err != nil {
		logger.Error("redis session storage setup failed", "error", err)
		os.Exit(1)
	}

	sessionStore := session.NewStore(session.Config{
		Storage:        redisStorage,
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
	})
	identityStore := identity.NewPostgresStore(pool)
	identityHandler := identity.NewHandler(identity.NewService(identityStore), sessionStore)

	app := httpserver.NewApp(cfg, logger, httpserver.Dependencies{
		IdentityHandler: identityHandler,
	})
	errCh := make(chan error, 1)

	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.HTTPHost, cfg.HTTPPort)
		logger.Info("starting api server", "addr", addr, "env", cfg.AppEnv, "locale", cfg.AppLocale)
		errCh <- app.Listen(addr)
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-stopCh:
		logger.Info("shutting down api server", "signal", sig.String())
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("api server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("api server shutdown failed", "error", err)
		os.Exit(1)
	}
}

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

	"github.com/inkedus/sforum/apps/api/internal/config"
	httpserver "github.com/inkedus/sforum/apps/api/internal/http"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	app := httpserver.NewApp(cfg, logger)
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

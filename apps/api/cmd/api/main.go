package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhuchunshu/sforum/apps/api/bootstrap"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	ctx := context.Background()
	api, err := bootstrap.NewAPI(ctx, cfg, logger)
	if err != nil {
		logger.Error("api bootstrap failed", "error", err)
		os.Exit(1)
	}
	defer api.Close()

	errCh := make(chan error, 1)

	go func() {
		logger.Info("starting api server", "addr", api.Addr, "env", cfg.AppEnv, "locale", cfg.AppLocale)
		errCh <- api.App.Listen(api.Addr)
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

	if err := api.App.ShutdownWithContext(ctx); err != nil {
		logger.Error("api server shutdown failed", "error", err)
		os.Exit(1)
	}
}

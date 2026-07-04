package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/zhuchunshu/sforum/apps/api/bootstrap"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	worker, err := bootstrap.NewWorker(ctx, cfg)
	if err != nil {
		logger.Error("worker bootstrap failed", "error", err)
		os.Exit(1)
	}
	defer worker.Close()

	logger.Info("starting worker", "env", cfg.AppEnv, "locale", cfg.AppLocale)
	if err := worker.Start(ctx); err != nil {
		logger.Error("worker start failed", "error", err)
		os.Exit(1)
	}

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.WorkerShutdownTimeout)
	defer cancel()

	if err := worker.Stop(shutdownCtx); err != nil {
		logger.Error("worker shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("worker stopped")
}

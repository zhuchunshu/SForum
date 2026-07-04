package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/zhuchunshu/sforum/apps/api/config"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	logger.Info("starting worker", "env", cfg.AppEnv, "locale", cfg.AppLocale)
	logger.Info("worker queue not configured yet; foundation process is alive")

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	sig := <-stopCh

	logger.Info("stopping worker", "signal", sig.String())
}

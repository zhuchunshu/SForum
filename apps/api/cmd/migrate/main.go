package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/zhuchunshu/sforum/apps/api/config"
	"github.com/zhuchunshu/sforum/apps/api/database/migrator"
	platformversion "github.com/zhuchunshu/sforum/apps/api/version"
)

func main() {
	if platformversion.PrintIfRequested(os.Stdout, os.Args[1:]) {
		return
	}
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	if err := migrator.Up(context.Background(), migrator.Config{
		DatabaseURL: cfg.DatabaseURL,
		Logger:      logger.With("component", "database.migrator"),
	}); err != nil {
		logger.Error("migrations failed", "error", err)
		os.Exit(1)
	}
}

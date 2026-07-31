package main

import (
	"context"
	"fmt"
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
	if hasArgument(os.Args[1:], "--check-no-pending") {
		upToDate, err := migrator.IsUpToDate(context.Background(), migrator.Config{
			DatabaseURL: cfg.DatabaseURL,
			Logger:      logger.With("component", "database.migrator"),
		})
		if err != nil {
			logger.Error("migration compatibility check failed", "error", err)
			os.Exit(1)
		}
		if !upToDate {
			logger.Error("zero-downtime update refused: target has pending or mismatched migrations")
			os.Exit(3)
		}
		fmt.Fprintln(os.Stdout, "schema-up-to-date")
		return
	}

	if err := migrator.Up(context.Background(), migrator.Config{
		DatabaseURL: cfg.DatabaseURL,
		Logger:      logger.With("component", "database.migrator"),
	}); err != nil {
		logger.Error("migrations failed", "error", err)
		os.Exit(1)
	}
}

func hasArgument(arguments []string, expected string) bool {
	for _, argument := range arguments {
		if argument == expected {
			return true
		}
	}
	return false
}

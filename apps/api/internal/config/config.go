package config

import (
	"log/slog"
	"os"
	"strings"

	"github.com/inkedus/sforum/apps/api/internal/modules/localization"
)

type Config struct {
	AppEnv           string
	AppName          string
	AppLocale        string
	SupportedLocales []string
	HTTPHost         string
	HTTPPort         string
	DatabaseURL      string
	RedisAddr        string
	MeiliHost        string
	MeiliMasterKey   string
	LogLevel         slog.Level
}

func Load() Config {
	supported := localization.ParseSupportedLocales(env("SUPPORTED_LOCALES", "zh-CN,en-US"))
	defaultLocale := localization.Normalize(env("APP_LOCALE", localization.DefaultLocale), supported)

	return Config{
		AppEnv:           env("APP_ENV", "development"),
		AppName:          env("APP_NAME", "SForum"),
		AppLocale:        defaultLocale,
		SupportedLocales: supported,
		HTTPHost:         env("HTTP_HOST", "0.0.0.0"),
		HTTPPort:         env("HTTP_PORT", "8080"),
		DatabaseURL:      env("DATABASE_URL", "postgres://sforum:sforum@postgres:5432/sforum?sslmode=disable"),
		RedisAddr:        env("REDIS_ADDR", "redis:6379"),
		MeiliHost:        env("MEILI_HOST", "http://meilisearch:7700"),
		MeiliMasterKey:   env("MEILI_MASTER_KEY", "sforum-dev-meili-key"),
		LogLevel:         parseLogLevel(env("LOG_LEVEL", "info")),
	}
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Localization"
)

type Config struct {
	AppEnv                       string
	AppName                      string
	AppLocale                    string
	SupportedLocales             []string
	HTTPHost                     string
	HTTPPort                     string
	DatabaseURL                  string
	DatabaseMaxConns             int32
	WorkerDatabaseMaxConns       int32
	WorkerShutdownTimeout        time.Duration
	RedisAddr                    string
	RedisPassword                string
	HumanVerificationProvider    string
	AltchaSecret                 string
	AltchaChallengeTTL           time.Duration
	AltchaCost                   int
	MeiliHost                    string
	MeiliMasterKey               string
	JobQueueCriticalWorkers      int
	JobQueueDefaultWorkers       int
	JobQueueSearchWorkers        int
	JobQueueMailWorkers          int
	JobQueueNotificationsWorkers int
	JobQueueMaintenanceWorkers   int
	LogLevel                     slog.Level
}

func Load() Config {
	supported := localization.ParseSupportedLocales(env("SUPPORTED_LOCALES", "zh-CN,en-US"))
	defaultLocale := localization.Normalize(env("APP_LOCALE", localization.DefaultLocale), supported)

	return Config{
		AppEnv:                       env("APP_ENV", "development"),
		AppName:                      env("APP_NAME", "SForum"),
		AppLocale:                    defaultLocale,
		SupportedLocales:             supported,
		HTTPHost:                     env("HTTP_HOST", "0.0.0.0"),
		HTTPPort:                     env("HTTP_PORT", "8080"),
		DatabaseURL:                  env("DATABASE_URL", "postgres://sforum:sforum@postgres:5432/sforum?sslmode=disable"),
		DatabaseMaxConns:             int32(envPositiveInt("DATABASE_MAX_CONNS", 10)),
		WorkerDatabaseMaxConns:       int32(envPositiveInt("WORKER_DATABASE_MAX_CONNS", 10)),
		WorkerShutdownTimeout:        envDuration("WORKER_SHUTDOWN_TIMEOUT", 30*time.Second),
		RedisAddr:                    env("REDIS_ADDR", "redis:6379"),
		RedisPassword:                env("REDIS_PASSWORD", ""),
		HumanVerificationProvider:    env("HUMAN_VERIFICATION_PROVIDER", "altcha"),
		AltchaSecret:                 env("ALTCHA_SECRET", "sforum-dev-altcha-secret"),
		AltchaChallengeTTL:           envDuration("ALTCHA_CHALLENGE_TTL", 10*time.Minute),
		AltchaCost:                   envPositiveInt("ALTCHA_COST", 1000),
		MeiliHost:                    env("MEILI_HOST", "http://meilisearch:7700"),
		MeiliMasterKey:               env("MEILI_MASTER_KEY", "sforum-dev-meili-key"),
		JobQueueCriticalWorkers:      envPositiveInt("JOB_QUEUE_CRITICAL_WORKERS", 4),
		JobQueueDefaultWorkers:       envPositiveInt("JOB_QUEUE_DEFAULT_WORKERS", 8),
		JobQueueSearchWorkers:        envPositiveInt("JOB_QUEUE_SEARCH_WORKERS", 6),
		JobQueueMailWorkers:          envPositiveInt("JOB_QUEUE_MAIL_WORKERS", 4),
		JobQueueNotificationsWorkers: envPositiveInt("JOB_QUEUE_NOTIFICATIONS_WORKERS", 6),
		JobQueueMaintenanceWorkers:   envPositiveInt("JOB_QUEUE_MAINTENANCE_WORKERS", 2),
		LogLevel:                     parseLogLevel(env("LOG_LEVEL", "info")),
	}
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envPositiveInt(key string, fallback int) int {
	value := envInt(key, fallback)
	if value <= 0 {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
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

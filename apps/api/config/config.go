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
	AppURL                       string
	AppLocale                    string
	SupportedLocales             []string
	HTTPHost                     string
	HTTPPort                     string
	HTTPReadTimeout              time.Duration
	HTTPWriteTimeout             time.Duration
	HTTPIdleTimeout              time.Duration
	HTTPBodyLimit                int
	CompressLevel                int
	DatabaseURL                  string
	MigrateOnStartup             bool
	DatabaseMaxConns             int32
	DatabaseMinConns             int32
	DatabaseMaxConnIdleTime      time.Duration
	DatabaseMaxConnLifetime      time.Duration
	DatabaseConnectTimeout       time.Duration
	EmbedWorkerInAPI             bool
	WorkerDatabaseMaxConns       int32
	WorkerDatabaseMinConns       int32
	WorkerDatabaseMaxConnIdleTime time.Duration
	WorkerDatabaseMaxConnLifetime time.Duration
	WorkerDatabaseConnectTimeout  time.Duration
	WorkerShutdownTimeout        time.Duration
	RedisAddr                    string
	RedisPassword                string
	RedisPoolSize                int
	RedisMinIdleConns            int
	RedisDialTimeout             time.Duration
	RedisReadTimeout             time.Duration
	RedisWriteTimeout            time.Duration
	RedisConnMaxIdleTime         time.Duration
	RedisConnMaxLifetime         time.Duration
	SessionIdleTimeout           time.Duration
	SessionAbsoluteTimeout       time.Duration
	SessionRenewalInterval       time.Duration
	SessionHashSecret            string
	HumanVerificationProvider    string
	AltchaSecret                 string
	AltchaChallengeTTL           time.Duration
	AltchaCost                   int
	ExtensionRoot                string
	BuiltinExtensionRoot         string
	ThemeReleaseRoot             string
	ThemeWebRoot                 string
	ThemeBunPath                 string
	ThemeBuildTimeout            time.Duration
	ThemePreviewTimeout          time.Duration
	ThemePreviewPath             string
	MeiliHost                    string
	MeiliMasterKey               string
	MeiliTimeout                 time.Duration
	LimiterWriteMax              int
	LimiterWindow                time.Duration
	JobQueueCriticalWorkers      int
	JobQueueDefaultWorkers       int
	JobQueueSearchWorkers        int
	JobQueueMailWorkers          int
	JobQueueNotificationsWorkers int
	JobQueueMaintenanceWorkers   int
	JobQueueThemeWorkers         int
	LogLevel                     slog.Level
}

func Load() Config {
	appEnv := env("APP_ENV", "development")
	supported := localization.ParseSupportedLocales(env("SUPPORTED_LOCALES", "zh-CN,en-US"))
	defaultLocale := localization.Normalize(env("APP_LOCALE", localization.DefaultLocale), supported)
	sessionIdleTimeout := envDuration("SESSION_IDLE_TIMEOUT", 30*24*time.Hour)
	sessionAbsoluteTimeout := envDuration("SESSION_ABSOLUTE_TIMEOUT", 180*24*time.Hour)
	if sessionAbsoluteTimeout < sessionIdleTimeout {
		sessionAbsoluteTimeout = sessionIdleTimeout
	}

	return Config{
		AppEnv:                       appEnv,
		AppName:                      env("APP_NAME", "SForum"),
		AppURL:                       env("APP_URL", "http://127.0.0.1:3000"),
		AppLocale:                    defaultLocale,
		SupportedLocales:             supported,
		HTTPHost:                     env("HTTP_HOST", "0.0.0.0"),
		HTTPPort:                     env("HTTP_PORT", "8080"),
		HTTPReadTimeout:              envDuration("HTTP_READ_TIMEOUT", 10*time.Second),
		HTTPWriteTimeout:             envDuration("HTTP_WRITE_TIMEOUT", 20*time.Second),
		HTTPIdleTimeout:              envDuration("HTTP_IDLE_TIMEOUT", 120*time.Second),
		HTTPBodyLimit:                envPositiveInt("HTTP_BODY_LIMIT", 4*1024*1024),
		CompressLevel:                compressLevelFromEnv(env("COMPRESS_LEVEL", "default")),
		DatabaseURL:                   env("DATABASE_URL", "postgres://sforum:sforum@postgres:5432/sforum?sslmode=disable"),
		MigrateOnStartup:              envBool("MIGRATE_ON_STARTUP", true),
		DatabaseMaxConns:              int32(envPositiveInt("DATABASE_MAX_CONNS", 10)),
		DatabaseMinConns:              int32(envPositiveInt("DATABASE_MIN_CONNS", 2)),
		DatabaseMaxConnIdleTime:       envDuration("DATABASE_MAX_CONN_IDLE_TIME", 30*time.Minute),
		DatabaseMaxConnLifetime:       envDuration("DATABASE_MAX_CONN_LIFETIME", time.Hour),
		DatabaseConnectTimeout:        envDuration("DATABASE_CONNECT_TIMEOUT", 10*time.Second),
		EmbedWorkerInAPI:              envBool("EMBED_WORKER_IN_API", strings.EqualFold(appEnv, "development")),
		WorkerDatabaseMaxConns:        int32(envPositiveInt("WORKER_DATABASE_MAX_CONNS", 10)),
		WorkerDatabaseMinConns:        int32(envPositiveInt("WORKER_DATABASE_MIN_CONNS", 2)),
		WorkerDatabaseMaxConnIdleTime: envDuration("WORKER_DATABASE_MAX_CONN_IDLE_TIME", 30*time.Minute),
		WorkerDatabaseMaxConnLifetime: envDuration("WORKER_DATABASE_MAX_CONN_LIFETIME", time.Hour),
		WorkerDatabaseConnectTimeout:  envDuration("WORKER_DATABASE_CONNECT_TIMEOUT", 10*time.Second),
		WorkerShutdownTimeout:         envDuration("WORKER_SHUTDOWN_TIMEOUT", 30*time.Second),
		RedisAddr:                    env("REDIS_ADDR", "redis:6379"),
		RedisPassword:                env("REDIS_PASSWORD", ""),
		RedisPoolSize:                envPositiveInt("REDIS_POOL_SIZE", 20),
		RedisMinIdleConns:            envPositiveInt("REDIS_MIN_IDLE_CONNS", 5),
		RedisDialTimeout:             envDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
		RedisReadTimeout:             envDuration("REDIS_READ_TIMEOUT", 3*time.Second),
		RedisWriteTimeout:            envDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
		RedisConnMaxIdleTime:         envDuration("REDIS_CONN_MAX_IDLE_TIME", 30*time.Minute),
		RedisConnMaxLifetime:         envDuration("REDIS_CONN_MAX_LIFETIME", time.Hour),
		SessionIdleTimeout:           sessionIdleTimeout,
		SessionAbsoluteTimeout:       sessionAbsoluteTimeout,
		SessionRenewalInterval:       envDuration("SESSION_RENEWAL_INTERVAL", 24*time.Hour),
		SessionHashSecret:            env("SESSION_HASH_SECRET", "sforum-dev-session-hash-secret"),
		HumanVerificationProvider:    env("HUMAN_VERIFICATION_PROVIDER", "disabled"),
		AltchaSecret:                 env("ALTCHA_SECRET", "sforum-dev-altcha-secret"),
		AltchaChallengeTTL:           envDuration("ALTCHA_CHALLENGE_TTL", 10*time.Minute),
		AltchaCost:                   envPositiveInt("ALTCHA_COST", 1000),
		ExtensionRoot:                env("EXTENSION_ROOT", "../../storage/extensions"),
		BuiltinExtensionRoot:         env("BUILTIN_EXTENSION_ROOT", "../../extensions/builtin"),
		ThemeReleaseRoot:             env("THEME_RELEASE_ROOT", "../../storage/theme-releases"),
		ThemeWebRoot:                 env("THEME_WEB_ROOT", "../web"),
		ThemeBunPath:                 env("THEME_BUN_PATH", "bun"),
		ThemeBuildTimeout:            envDuration("THEME_BUILD_TIMEOUT", 5*time.Minute),
		ThemePreviewTimeout:          envDuration("THEME_PREVIEW_TIMEOUT", 30*time.Second),
		ThemePreviewPath:             env("THEME_PREVIEW_PATH", "/"),
		MeiliHost:                    env("MEILI_HOST", "http://meilisearch:7700"),
		MeiliMasterKey:               env("MEILI_MASTER_KEY", "sforum-dev-meili-key"),
		MeiliTimeout:                 envDuration("MEILI_TIMEOUT", 5*time.Second),
		LimiterWriteMax:              envPositiveInt("LIMITER_WRITE_MAX", 30),
		LimiterWindow:                envDuration("LIMITER_WINDOW", time.Minute),
		JobQueueCriticalWorkers:      envPositiveInt("JOB_QUEUE_CRITICAL_WORKERS", 4),
		JobQueueDefaultWorkers:       envPositiveInt("JOB_QUEUE_DEFAULT_WORKERS", 8),
		JobQueueSearchWorkers:        envPositiveInt("JOB_QUEUE_SEARCH_WORKERS", 6),
		JobQueueMailWorkers:          envPositiveInt("JOB_QUEUE_MAIL_WORKERS", 4),
		JobQueueNotificationsWorkers: envPositiveInt("JOB_QUEUE_NOTIFICATIONS_WORKERS", 6),
		JobQueueMaintenanceWorkers:   envPositiveInt("JOB_QUEUE_MAINTENANCE_WORKERS", 2),
		JobQueueThemeWorkers:         envPositiveInt("JOB_QUEUE_THEME_WORKERS", 1),
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

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return fallback
	}
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

// compressLevelFromEnv 把环境变量字符串映射为 fiber compress Level。
// fiber compress: 0=LevelDefault, 1=LevelBestSpeed, 2=LevelBestCompression, -1=LevelDisabled。
func compressLevelFromEnv(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "disabled", "off", "none":
		return -1
	case "best_speed", "speed", "fast":
		return 1
	case "best_compression", "compression", "max":
		return 2
	default:
		return 0
	}
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

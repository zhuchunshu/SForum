package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Localization"
)

type Config struct {
	AppEnv                        string
	AppName                       string
	AppURL                        string
	AppLocale                     string
	SupportedLocales              []string
	HTTPHost                      string
	HTTPPort                      string
	HTTPReadTimeout               time.Duration
	HTTPWriteTimeout              time.Duration
	HTTPIdleTimeout               time.Duration
	HTTPBodyLimit                 int
	CompressLevel                 int
	DatabaseURL                   string
	MigrateOnStartup              bool
	DatabaseMaxConns              int32
	DatabaseMinConns              int32
	DatabaseMaxConnIdleTime       time.Duration
	DatabaseMaxConnLifetime       time.Duration
	DatabaseConnectTimeout        time.Duration
	EmbedWorkerInAPI              bool
	WorkerDatabaseMaxConns        int32
	WorkerDatabaseMinConns        int32
	WorkerDatabaseMaxConnIdleTime time.Duration
	WorkerDatabaseMaxConnLifetime time.Duration
	WorkerDatabaseConnectTimeout  time.Duration
	WorkerShutdownTimeout         time.Duration
	RedisAddr                     string
	RedisPassword                 string
	RedisPoolSize                 int
	RedisMinIdleConns             int
	RedisDialTimeout              time.Duration
	RedisReadTimeout              time.Duration
	RedisWriteTimeout             time.Duration
	RedisConnMaxIdleTime          time.Duration
	RedisConnMaxLifetime          time.Duration
	SessionIdleTimeout            time.Duration
	SessionAbsoluteTimeout        time.Duration
	SessionRenewalInterval        time.Duration
	SessionHashSecret             string
	HumanVerificationProvider     string
	AltchaSecret                  string
	AltchaChallengeTTL            time.Duration
	AltchaCost                    int
	// OptionEncryptionKey 是 web_options 中敏感值（云存储凭证/SFTP 私钥等）AES-GCM 加密的密钥（hex 编码）。
	// 生产环境必须显式配置；缺失或为占位词时启动会被拒绝。
	OptionEncryptionKey      string
	ExtensionRoot            string
	BuiltinExtensionRoot     string
	WebReleaseRoot           string
	WebReleaseWebRoot        string
	WebReleaseBunPath        string
	WebReleaseBuildTimeout   time.Duration
	WebReleasePreviewTimeout time.Duration
	WebReleasePreviewPath    string
	ThemeReleaseRoot         string
	ThemeWebRoot             string
	ThemeBunPath             string
	ThemeBuildTimeout        time.Duration
	ThemePreviewTimeout      time.Duration
	ThemePreviewPath         string
	MeiliHost                string
	MeiliMasterKey           string
	MeiliTimeout             time.Duration
	LimiterWriteMax          int
	LimiterWindow            time.Duration
	// CSRFTrustedOrigins 是 CSRF 中间件信任的来源站点 origin 列表（如 https://forum.example.com）。
	// API 在反向代理后看到的 Host 是内部地址，而 Origin 是公开站点，二者不匹配会被拒绝，
	// 因此必须显式列出公开站点。支持 https://*.example.com 通配符子域。
	CSRFTrustedOrigins []string
	// CSRFEnabled 控制 CSRF 中间件是否启用，默认 true（生产启用）。
	// 测试场景显式置 false 以避免每个测试请求都需携带 token。
	CSRFEnabled                  bool
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
	// CSRF 信任源：优先读 CSRF_TRUSTED_ORIGINS；未配置时从 APP_URL 派生 scheme://host 作为默认。
	csrfOrigins := envStringSlice("CSRF_TRUSTED_ORIGINS")
	if len(csrfOrigins) == 0 {
		csrfOrigins = originsFromAppURL(env("APP_URL", ""))
	}
	webReleaseRoot := env("WEB_RELEASE_ROOT", env("THEME_RELEASE_ROOT", "../../storage/theme-releases"))
	webReleaseWebRoot := env("WEB_RELEASE_WEB_ROOT", env("THEME_WEB_ROOT", "../web"))
	webReleaseBunPath := env("WEB_RELEASE_BUN_PATH", env("THEME_BUN_PATH", "bun"))
	webReleaseBuildTimeout := envDuration("WEB_RELEASE_BUILD_TIMEOUT", envDuration("THEME_BUILD_TIMEOUT", 5*time.Minute))
	webReleasePreviewTimeout := envDuration("WEB_RELEASE_PREVIEW_TIMEOUT", envDuration("THEME_PREVIEW_TIMEOUT", 30*time.Second))
	webReleasePreviewPath := env("WEB_RELEASE_PREVIEW_PATH", env("THEME_PREVIEW_PATH", "/"))

	cfg := Config{
		AppEnv:           appEnv,
		AppName:          env("APP_NAME", "SForum"),
		AppURL:           env("APP_URL", "http://127.0.0.1:3000"),
		AppLocale:        defaultLocale,
		SupportedLocales: supported,
		HTTPHost:         env("HTTP_HOST", "0.0.0.0"),
		HTTPPort:         env("HTTP_PORT", "8080"),
		HTTPReadTimeout:  envDuration("HTTP_READ_TIMEOUT", 10*time.Second),
		HTTPWriteTimeout: envDuration("HTTP_WRITE_TIMEOUT", 20*time.Second),
		HTTPIdleTimeout:  envDuration("HTTP_IDLE_TIMEOUT", 120*time.Second),
		HTTPBodyLimit:    envPositiveInt("HTTP_BODY_LIMIT", 4*1024*1024),
		CompressLevel:    compressLevelFromEnv(env("COMPRESS_LEVEL", "default")),
		// 默认启用 TLS（sslmode=require）；本地开发无 TLS 的 Postgres 需显式设置 sslmode=disable。
		DatabaseURL:                   env("DATABASE_URL", "postgres://sforum:sforum@postgres:5432/sforum?sslmode=require"),
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
		RedisAddr:                     env("REDIS_ADDR", "redis:6379"),
		RedisPassword:                 env("REDIS_PASSWORD", ""),
		RedisPoolSize:                 envPositiveInt("REDIS_POOL_SIZE", 20),
		RedisMinIdleConns:             envPositiveInt("REDIS_MIN_IDLE_CONNS", 5),
		RedisDialTimeout:              envDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
		RedisReadTimeout:              envDuration("REDIS_READ_TIMEOUT", 3*time.Second),
		RedisWriteTimeout:             envDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
		RedisConnMaxIdleTime:          envDuration("REDIS_CONN_MAX_IDLE_TIME", 30*time.Minute),
		RedisConnMaxLifetime:          envDuration("REDIS_CONN_MAX_LIFETIME", time.Hour),
		SessionIdleTimeout:            sessionIdleTimeout,
		SessionAbsoluteTimeout:        sessionAbsoluteTimeout,
		SessionRenewalInterval:        envDuration("SESSION_RENEWAL_INTERVAL", 24*time.Hour),
		SessionHashSecret:             env("SESSION_HASH_SECRET", "sforum-dev-session-hash-secret"),
		HumanVerificationProvider:     env("HUMAN_VERIFICATION_PROVIDER", "disabled"),
		AltchaSecret:                  env("ALTCHA_SECRET", "sforum-dev-altcha-secret"),
		AltchaChallengeTTL:            envDuration("ALTCHA_CHALLENGE_TTL", 10*time.Minute),
		AltchaCost:                    envPositiveInt("ALTCHA_COST", 1000),
		OptionEncryptionKey:           env("APP_OPTION_ENC_KEY", ""),
		ExtensionRoot:                 env("EXTENSION_ROOT", "../../storage/extensions"),
		BuiltinExtensionRoot:          env("BUILTIN_EXTENSION_ROOT", "../../extensions/builtin"),
		WebReleaseRoot:                webReleaseRoot,
		WebReleaseWebRoot:             webReleaseWebRoot,
		WebReleaseBunPath:             webReleaseBunPath,
		WebReleaseBuildTimeout:        webReleaseBuildTimeout,
		WebReleasePreviewTimeout:      webReleasePreviewTimeout,
		WebReleasePreviewPath:         webReleasePreviewPath,
		ThemeReleaseRoot:              webReleaseRoot,
		ThemeWebRoot:                  webReleaseWebRoot,
		ThemeBunPath:                  webReleaseBunPath,
		ThemeBuildTimeout:             webReleaseBuildTimeout,
		ThemePreviewTimeout:           webReleasePreviewTimeout,
		ThemePreviewPath:              webReleasePreviewPath,
		MeiliHost:                     env("MEILI_HOST", "http://meilisearch:7700"),
		MeiliMasterKey:                env("MEILI_MASTER_KEY", "sforum-dev-meili-key"),
		MeiliTimeout:                  envDuration("MEILI_TIMEOUT", 5*time.Second),
		LimiterWriteMax:               envPositiveInt("LIMITER_WRITE_MAX", 30),
		LimiterWindow:                 envDuration("LIMITER_WINDOW", time.Minute),
		CSRFTrustedOrigins:            csrfOrigins,
		CSRFEnabled:                   envBool("CSRF_ENABLED", true),
		JobQueueCriticalWorkers:       envPositiveInt("JOB_QUEUE_CRITICAL_WORKERS", 4),
		JobQueueDefaultWorkers:        envPositiveInt("JOB_QUEUE_DEFAULT_WORKERS", 8),
		JobQueueSearchWorkers:         envPositiveInt("JOB_QUEUE_SEARCH_WORKERS", 6),
		JobQueueMailWorkers:           envPositiveInt("JOB_QUEUE_MAIL_WORKERS", 4),
		JobQueueNotificationsWorkers:  envPositiveInt("JOB_QUEUE_NOTIFICATIONS_WORKERS", 6),
		JobQueueMaintenanceWorkers:    envPositiveInt("JOB_QUEUE_MAINTENANCE_WORKERS", 2),
		JobQueueThemeWorkers:          envPositiveInt("JOB_QUEUE_THEME_WORKERS", 1),
		LogLevel:                      parseLogLevel(env("LOG_LEVEL", "info")),
	}
	validateProductionSecrets(cfg)
	return cfg
}

// insecureSecretValues 是已知的不安全默认/占位值，生产环境不得使用。
var insecureSecretValues = map[string]bool{
	"":                               true,
	"change-me":                      true,
	"sforum-dev-session-hash-secret": true,
	"sforum-dev-altcha-secret":       true,
	"sforum-dev-meili-key":           true,
}

// validateProductionSecrets 在生产环境校验关键密钥非空且非占位词。
// 任一不满足直接 panic 拒绝启动，避免运维忘记配置导致生产静默回退到公开默认值。
func validateProductionSecrets(cfg Config) {
	if !strings.EqualFold(cfg.AppEnv, "production") {
		return
	}
	type secretCheck struct {
		name string
		val  string
	}
	checks := []secretCheck{
		{"SESSION_HASH_SECRET", cfg.SessionHashSecret},
		{"ALTCHA_SECRET", cfg.AltchaSecret},
		{"MEILI_MASTER_KEY", cfg.MeiliMasterKey},
		{"APP_OPTION_ENC_KEY", cfg.OptionEncryptionKey},
	}
	for _, c := range checks {
		if insecureSecretValues[strings.TrimSpace(c.val)] {
			panic(fmt.Sprintf("config: %s must be set to a secure value in production (got empty/placeholder default)", c.name))
		}
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

// envStringSlice 读取逗号分隔的字符串列表，逐项 trim 并丢弃空项与重复项。
func envStringSlice(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	seen := map[string]bool{}
	items := []string{}
	for _, part := range strings.Split(raw, ",") {
		item := strings.TrimSpace(part)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		items = append(items, item)
	}
	return items
}

// ShouldUseSecureCookie 决定 session/CSRF cookie 是否带 Secure 标志。
// 生产环境强制启用；此外当 APP_URL 为 https 时也启用，避免 staging HTTPS 漏配。
func ShouldUseSecureCookie(cfg Config) bool {
	if strings.EqualFold(cfg.AppEnv, "production") {
		return true
	}
	if parsed, err := url.Parse(strings.TrimSpace(cfg.AppURL)); err == nil {
		return strings.EqualFold(parsed.Scheme, "https")
	}
	return false
}

// originsFromAppURL 从 APP_URL 派生 origin（scheme://host），作为 CSRF 信任源默认值。
// 解析失败或为空时返回 nil，交由上层按空信任源处理。
func originsFromAppURL(appURL string) []string {
	appURL = strings.TrimSpace(appURL)
	if appURL == "" {
		return nil
	}
	parsed, err := url.Parse(appURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil
	}
	return []string{parsed.Scheme + "://" + parsed.Host}
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

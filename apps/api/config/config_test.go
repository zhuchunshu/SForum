package config

import (
	"log/slog"
	"os"
	"reflect"
	"testing"
	"time"
)

func clearJobQueueEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"JOB_QUEUE_CRITICAL_WORKERS",
		"JOB_QUEUE_DEFAULT_WORKERS",
		"JOB_QUEUE_SEARCH_WORKERS",
		"JOB_QUEUE_MAIL_WORKERS",
		"JOB_QUEUE_NOTIFICATIONS_WORKERS",
		"JOB_QUEUE_MAINTENANCE_WORKERS",
		"DATABASE_MAX_CONNS",
		"DATABASE_MIN_CONNS",
		"WORKER_DATABASE_MAX_CONNS",
		"WORKER_DATABASE_MIN_CONNS",
		"REDIS_POOL_SIZE",
		"REDIS_MIN_IDLE_CONNS",
		"REDIS_PASSWORD",
		"EXTERNAL_EXTENSION_ROOTS",
		"LOG_LEVEL",
	} {
		t.Setenv(key, "")
	}
}

func TestLoadIncludesDefaultWorkerConfig(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("DATABASE_URL", "postgres://example:example@localhost:5432/example?sslmode=disable")
	clearJobQueueEnv(t)

	cfg := Load()

	if cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("expected info log level, got %v", cfg.LogLevel)
	}
	if cfg.DatabaseMaxConns != 10 {
		t.Fatalf("expected default api database max conns 10, got %d", cfg.DatabaseMaxConns)
	}
	if !cfg.MigrateOnStartup {
		t.Fatal("expected startup migrations to be enabled by default")
	}
	if cfg.WorkerDatabaseMaxConns != 10 {
		t.Fatalf("expected default worker database max conns 10, got %d", cfg.WorkerDatabaseMaxConns)
	}
	if cfg.WorkerShutdownTimeout != 30*time.Second {
		t.Fatalf("expected worker shutdown timeout 30s, got %s", cfg.WorkerShutdownTimeout)
	}
	if cfg.RedisPassword != "" {
		t.Fatalf("expected empty redis password default, got %q", cfg.RedisPassword)
	}
	if cfg.SessionIdleTimeout != 30*24*time.Hour {
		t.Fatalf("expected session idle timeout 30d, got %s", cfg.SessionIdleTimeout)
	}
	if cfg.SessionAbsoluteTimeout != 180*24*time.Hour {
		t.Fatalf("expected session absolute timeout 180d, got %s", cfg.SessionAbsoluteTimeout)
	}
	if cfg.SessionRenewalInterval != 24*time.Hour {
		t.Fatalf("expected session renewal interval 24h, got %s", cfg.SessionRenewalInterval)
	}
	if cfg.SessionHashSecret != "sforum-dev-session-hash-secret" {
		t.Fatalf("expected development session hash secret default, got %q", cfg.SessionHashSecret)
	}
	if cfg.HumanVerificationProvider != "disabled" {
		t.Fatalf("expected disabled human verification provider default, got %q", cfg.HumanVerificationProvider)
	}
	if cfg.AltchaSecret != "sforum-dev-altcha-secret" {
		t.Fatalf("expected development altcha secret default, got %q", cfg.AltchaSecret)
	}
	if cfg.AltchaChallengeTTL != 10*time.Minute {
		t.Fatalf("expected altcha challenge ttl 10m, got %s", cfg.AltchaChallengeTTL)
	}
	if cfg.AltchaCost != 1000 {
		t.Fatalf("expected altcha cost 1000, got %d", cfg.AltchaCost)
	}
	if cfg.ExtensionRoot != "../../storage/extensions" {
		t.Fatalf("expected extension root default, got %q", cfg.ExtensionRoot)
	}
	if cfg.BuiltinExtensionRoot != "../../extensions/builtin" {
		t.Fatalf("expected builtin extension root default, got %q", cfg.BuiltinExtensionRoot)
	}
	if len(cfg.ExternalExtensionRoots) != 0 {
		t.Fatalf("expected no external extension roots by default, got %v", cfg.ExternalExtensionRoots)
	}
	if cfg.V3TrustChallenges {
		t.Fatal("expected V3 trust challenges to remain default-off during migration")
	}
	if cfg.SafeMode {
		t.Fatal("expected safe mode to remain opt-in")
	}
	if cfg.TrustChallengeTTL != 5*time.Minute {
		t.Fatalf("expected trust challenge TTL 5m, got %s", cfg.TrustChallengeTTL)
	}
	if cfg.JobQueueCriticalWorkers != 4 {
		t.Fatalf("expected critical workers 4, got %d", cfg.JobQueueCriticalWorkers)
	}
	if cfg.JobQueueDefaultWorkers != 8 {
		t.Fatalf("expected default workers 8, got %d", cfg.JobQueueDefaultWorkers)
	}
	if cfg.JobQueueSearchWorkers != 6 {
		t.Fatalf("expected search workers 6, got %d", cfg.JobQueueSearchWorkers)
	}
	if cfg.JobQueueMailWorkers != 4 {
		t.Fatalf("expected mail workers 4, got %d", cfg.JobQueueMailWorkers)
	}
	if cfg.JobQueueNotificationsWorkers != 6 {
		t.Fatalf("expected notifications workers 6, got %d", cfg.JobQueueNotificationsWorkers)
	}
	if cfg.JobQueueMaintenanceWorkers != 2 {
		t.Fatalf("expected maintenance workers 2, got %d", cfg.JobQueueMaintenanceWorkers)
	}
	if total := JobQueueWorkerTotal(cfg); total != 30 {
		t.Fatalf("expected production-scale worker total 30 for non-development, got %d", total)
	}
}

func TestLoadDevelopmentLeanJobQueueAndPoolDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://example:example@localhost:5432/example?sslmode=disable")
	clearJobQueueEnv(t)

	cfg := Load()

	if cfg.JobQueueCriticalWorkers != 1 ||
		cfg.JobQueueDefaultWorkers != 2 ||
		cfg.JobQueueSearchWorkers != 1 ||
		cfg.JobQueueMailWorkers != 1 ||
		cfg.JobQueueNotificationsWorkers != 1 ||
		cfg.JobQueueMaintenanceWorkers != 1 {
		t.Fatalf("unexpected development job queue defaults: critical=%d default=%d search=%d mail=%d notifications=%d maintenance=%d",
			cfg.JobQueueCriticalWorkers, cfg.JobQueueDefaultWorkers, cfg.JobQueueSearchWorkers,
			cfg.JobQueueMailWorkers, cfg.JobQueueNotificationsWorkers, cfg.JobQueueMaintenanceWorkers)
	}
	if total := JobQueueWorkerTotal(cfg); total != 7 {
		t.Fatalf("expected lean development worker total 7, got %d", total)
	}
	if total := JobQueueWorkerTotal(cfg); total >= 30 {
		t.Fatalf("development defaults must be leaner than production 30-slot profile, got %d", total)
	}
	if cfg.DatabaseMaxConns != 5 || cfg.DatabaseMinConns != 1 {
		t.Fatalf("expected lean dev database pool 5/1, got %d/%d", cfg.DatabaseMaxConns, cfg.DatabaseMinConns)
	}
	if cfg.RedisPoolSize != 8 || cfg.RedisMinIdleConns != 1 {
		t.Fatalf("expected lean dev redis pool 8/1, got %d/%d", cfg.RedisPoolSize, cfg.RedisMinIdleConns)
	}
}

func TestLoadProductionKeepsFullJobQueueDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	setValidProductionSecrets(t)
	clearJobQueueEnv(t)

	cfg := Load()
	if JobQueueWorkerTotal(cfg) != 30 {
		t.Fatalf("production must keep 30-slot defaults, got %d", JobQueueWorkerTotal(cfg))
	}
	if cfg.DatabaseMaxConns != 10 || cfg.RedisPoolSize != 20 {
		t.Fatalf("production pools drifted: db=%d redis=%d", cfg.DatabaseMaxConns, cfg.RedisPoolSize)
	}
}

func TestLoadJobQueueEnvOverridesDevelopmentLeanDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://example:example@localhost:5432/example?sslmode=disable")
	clearJobQueueEnv(t)
	t.Setenv("JOB_QUEUE_DEFAULT_WORKERS", "9")
	t.Setenv("DATABASE_MAX_CONNS", "12")

	cfg := Load()
	if cfg.JobQueueDefaultWorkers != 9 {
		t.Fatalf("expected env override for default workers, got %d", cfg.JobQueueDefaultWorkers)
	}
	if cfg.DatabaseMaxConns != 12 {
		t.Fatalf("expected env override for database max conns, got %d", cfg.DatabaseMaxConns)
	}
	// 未覆盖的队列仍用 lean 默认。
	if cfg.JobQueueCriticalWorkers != 1 {
		t.Fatalf("expected lean critical default 1, got %d", cfg.JobQueueCriticalWorkers)
	}
}

func TestLoadParsesV3TrustChallengeGate(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("SFORUM_V3_TRUST_CHALLENGES", "true")
	t.Setenv("SFORUM_V3_TRUST_CHALLENGE_TTL", "2m")
	cfg := Load()
	if !cfg.V3TrustChallenges || cfg.TrustChallengeTTL != 2*time.Minute {
		t.Fatalf("unexpected V3 trust config: enabled=%v ttl=%s", cfg.V3TrustChallenges, cfg.TrustChallengeTTL)
	}
}

func TestLoadDefaultsV3TrustChallengesByEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("SFORUM_V3_TRUST_CHALLENGES", "")
	if cfg := Load(); cfg.V3TrustChallenges {
		t.Fatal("expected development V3 trust challenges to remain opt-in")
	}

	t.Setenv("APP_ENV", "production")
	setValidProductionSecrets(t)
	if cfg := Load(); !cfg.V3TrustChallenges {
		t.Fatal("expected production V3 trust challenges to default on after the P1 gate")
	}

	t.Setenv("SFORUM_V3_TRUST_CHALLENGES", "false")
	if cfg := Load(); cfg.V3TrustChallenges {
		t.Fatal("expected an explicit compatibility override to disable V3 trust challenges")
	}
}

func TestLoadDefaultsPublicL2ByEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("SFORUM_V3_PUBLIC_L2", "")
	if cfg := Load(); cfg.V3PublicL2 {
		t.Fatal("expected development public L2 to remain disabled by default")
	}

	t.Setenv("APP_ENV", "production")
	setValidProductionSecrets(t)
	if cfg := Load(); cfg.V3PublicL2 {
		t.Fatal("expected production public L2 to remain disabled before the P9 production gate")
	}

	t.Setenv("SFORUM_V3_PUBLIC_L2", "true")
	if cfg := Load(); !cfg.V3PublicL2 {
		t.Fatal("expected explicit public L2 opt-in to enable the migration slice")
	}
}

func TestLoadParsesSafeMode(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("SFORUM_SAFE_MODE", "1")
	if cfg := Load(); !cfg.SafeMode {
		t.Fatal("expected SFORUM_SAFE_MODE=1 to enable safe mode")
	}
}

// setValidProductionSecrets 给生产环境测试补上有效密钥，避免触发 validateProductionSecrets。
func setValidProductionSecrets(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"SESSION_HASH_SECRET":          "prod-valid-session-secret",
		"ALTCHA_SECRET":                "prod-valid-altcha-secret",
		"APP_OPTION_ENC_KEY":           "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"IDENTITY_SUBJECT_HMAC_SECRET": "prod-valid-identity-subject-hmac-secret-32b",
	} {
		t.Setenv(k, v)
	}
}

func TestLoadEnablesEmbeddedWorkerForDevelopmentOnlyByDefault(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	development := Load()
	if !development.EmbedWorkerInAPI {
		t.Fatal("expected development api to embed the worker by default")
	}

	t.Setenv("APP_ENV", "production")
	setValidProductionSecrets(t)
	production := Load()
	if production.EmbedWorkerInAPI {
		t.Fatal("expected production api to keep worker as a separate process by default")
	}
}

func TestLoadAllowsEmbeddedWorkerOverride(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("EMBED_WORKER_IN_API", "false")
	disabled := Load()
	if disabled.EmbedWorkerInAPI {
		t.Fatal("expected env override to disable embedded api worker")
	}

	t.Setenv("APP_ENV", "production")
	setValidProductionSecrets(t)
	t.Setenv("EMBED_WORKER_IN_API", "true")
	enabled := Load()
	if !enabled.EmbedWorkerInAPI {
		t.Fatal("expected env override to enable embedded api worker")
	}
}

func TestConfigDoesNotExposeAttachmentLocalRootEnv(t *testing.T) {
	if _, ok := reflect.TypeOf(Config{}).FieldByName("AttachmentLocalRoot"); ok {
		t.Fatal("attachment local root should be managed by runtime options, not process environment config")
	}
}

func TestLoadTrustProxyDefaultsByEnvironment(t *testing.T) {
	// 开发：默认开启 TrustProxy，并信任私网/loopback（Docker + Nuxt 反代）。
	t.Setenv("APP_ENV", "development")
	t.Setenv("TRUST_PROXY", "")
	t.Setenv("TRUSTED_PROXIES", "")
	t.Setenv("TRUST_PROXY_PRIVATE", "")
	t.Setenv("TRUST_PROXY_LOOPBACK", "")
	dev := Load()
	if !dev.TrustProxy {
		t.Fatal("expected development TrustProxy=true by default")
	}
	if !dev.TrustProxyPrivate || !dev.TrustProxyLoopback {
		t.Fatal("expected development to trust private/loopback by default")
	}
	if dev.ProxyHeader != "X-Forwarded-For" {
		t.Fatalf("expected default ProxyHeader X-Forwarded-For, got %q", dev.ProxyHeader)
	}

	// 生产：默认关闭 TrustProxy，且不信任私网（须显式配置）。
	t.Setenv("APP_ENV", "production")
	setValidProductionSecrets(t)
	t.Setenv("TRUST_PROXY", "")
	t.Setenv("TRUSTED_PROXIES", "")
	t.Setenv("TRUST_PROXY_PRIVATE", "")
	t.Setenv("TRUST_PROXY_LOOPBACK", "")
	prod := Load()
	if prod.TrustProxy {
		t.Fatal("expected production TrustProxy=false by default")
	}
	if prod.TrustProxyPrivate || prod.TrustProxyLoopback {
		t.Fatal("expected production not to trust private/loopback by default")
	}

	// 生产显式开启 + CIDR。
	t.Setenv("TRUST_PROXY", "true")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8,203.0.113.10")
	t.Setenv("PROXY_HEADER", "X-Real-IP")
	prodOn := Load()
	if !prodOn.TrustProxy {
		t.Fatal("expected explicit TRUST_PROXY=true")
	}
	if len(prodOn.TrustedProxies) != 2 {
		t.Fatalf("expected 2 trusted proxies, got %#v", prodOn.TrustedProxies)
	}
	if prodOn.ProxyHeader != "X-Real-IP" {
		t.Fatalf("expected ProxyHeader override, got %q", prodOn.ProxyHeader)
	}
}

func TestLoadIncludesDefaultWorkerConfigTrustProxyInTestEnv(t *testing.T) {
	// APP_ENV=test 与 development 一样：非 production → 默认信任私网。
	t.Setenv("APP_ENV", "test")
	cfg := Load()
	if !cfg.TrustProxy || !cfg.TrustProxyPrivate {
		t.Fatal("expected test env to trust proxy/private by default")
	}
}

// TestConfigDoesNotExposePhantomSessionOrCSRFFields 防止文档/示例中残留的
// SESSION_SECRET、CSRF_SECRET 被误当成真实配置字段。
// 实际生效的是 SESSION_HASH_SECRET；CSRF 功能落地前不应出现对应字段。
func TestConfigDoesNotExposePhantomSessionOrCSRFFields(t *testing.T) {
	for _, field := range []string{"SessionSecret", "CSRFSecret"} {
		if _, ok := reflect.TypeOf(Config{}).FieldByName(field); ok {
			t.Fatalf("Config should not expose phantom field %q; use SessionHashSecret and wait for CSRF middleware", field)
		}
	}
	if _, ok := reflect.TypeOf(Config{}).FieldByName("SessionHashSecret"); !ok {
		t.Fatal("Config should expose SessionHashSecret (the actually-read env var)")
	}
}

func TestLoadParsesWorkerConfigFromEnv(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("DATABASE_MAX_CONNS", "17")
	t.Setenv("MIGRATE_ON_STARTUP", "false")
	t.Setenv("WORKER_DATABASE_MAX_CONNS", "23")
	t.Setenv("WORKER_SHUTDOWN_TIMEOUT", "45s")
	t.Setenv("JOB_QUEUE_CRITICAL_WORKERS", "1")
	t.Setenv("JOB_QUEUE_DEFAULT_WORKERS", "2")
	t.Setenv("JOB_QUEUE_SEARCH_WORKERS", "3")
	t.Setenv("JOB_QUEUE_MAIL_WORKERS", "4")
	t.Setenv("JOB_QUEUE_NOTIFICATIONS_WORKERS", "5")
	t.Setenv("JOB_QUEUE_MAINTENANCE_WORKERS", "6")
	t.Setenv("REDIS_PASSWORD", "secret")
	t.Setenv("SESSION_IDLE_TIMEOUT", "12h")
	t.Setenv("SESSION_ABSOLUTE_TIMEOUT", "240h")
	t.Setenv("SESSION_RENEWAL_INTERVAL", "6h")
	t.Setenv("SESSION_HASH_SECRET", "test-session-hash-secret")
	t.Setenv("HUMAN_VERIFICATION_PROVIDER", "disabled")
	t.Setenv("ALTCHA_SECRET", "test-altcha-secret")
	t.Setenv("ALTCHA_CHALLENGE_TTL", "2m")
	t.Setenv("ALTCHA_COST", "2000")
	t.Setenv("EXTENSION_ROOT", "/srv/sforum/extensions")
	t.Setenv("BUILTIN_EXTENSION_ROOT", "/srv/sforum/builtin-extensions")
	t.Setenv("EXTERNAL_EXTENSION_ROOTS", "/srv/company-plugins, /srv/community-plugins,/srv/company-plugins")

	cfg := Load()

	if cfg.DatabaseMaxConns != 17 {
		t.Fatalf("expected api max conns from env, got %d", cfg.DatabaseMaxConns)
	}
	if cfg.MigrateOnStartup {
		t.Fatal("expected startup migrations to be disabled from env")
	}
	if cfg.WorkerDatabaseMaxConns != 23 {
		t.Fatalf("expected worker max conns from env, got %d", cfg.WorkerDatabaseMaxConns)
	}
	if cfg.WorkerShutdownTimeout != 45*time.Second {
		t.Fatalf("expected shutdown timeout from env, got %s", cfg.WorkerShutdownTimeout)
	}
	if cfg.JobQueueCriticalWorkers != 1 ||
		cfg.JobQueueDefaultWorkers != 2 ||
		cfg.JobQueueSearchWorkers != 3 ||
		cfg.JobQueueMailWorkers != 4 ||
		cfg.JobQueueNotificationsWorkers != 5 ||
		cfg.JobQueueMaintenanceWorkers != 6 {
		t.Fatalf("unexpected queue worker config: %+v", cfg)
	}
	if cfg.RedisPassword != "secret" {
		t.Fatalf("expected redis password from env, got %q", cfg.RedisPassword)
	}
	if cfg.SessionIdleTimeout != 12*time.Hour {
		t.Fatalf("expected session idle timeout from env, got %s", cfg.SessionIdleTimeout)
	}
	if cfg.SessionAbsoluteTimeout != 240*time.Hour {
		t.Fatalf("expected session absolute timeout from env, got %s", cfg.SessionAbsoluteTimeout)
	}
	if cfg.SessionRenewalInterval != 6*time.Hour {
		t.Fatalf("expected session renewal interval from env, got %s", cfg.SessionRenewalInterval)
	}
	if cfg.SessionHashSecret != "test-session-hash-secret" {
		t.Fatalf("expected session hash secret from env, got %q", cfg.SessionHashSecret)
	}
	if cfg.HumanVerificationProvider != "disabled" {
		t.Fatalf("expected disabled provider from env, got %q", cfg.HumanVerificationProvider)
	}
	if cfg.AltchaSecret != "test-altcha-secret" {
		t.Fatalf("expected altcha secret from env, got %q", cfg.AltchaSecret)
	}
	if cfg.AltchaChallengeTTL != 2*time.Minute {
		t.Fatalf("expected altcha ttl from env, got %s", cfg.AltchaChallengeTTL)
	}
	if cfg.AltchaCost != 2000 {
		t.Fatalf("expected altcha cost from env, got %d", cfg.AltchaCost)
	}
	if cfg.ExtensionRoot != "/srv/sforum/extensions" {
		t.Fatalf("expected extension root from env, got %q", cfg.ExtensionRoot)
	}
	if cfg.BuiltinExtensionRoot != "/srv/sforum/builtin-extensions" {
		t.Fatalf("expected builtin extension root from env, got %q", cfg.BuiltinExtensionRoot)
	}
	if len(cfg.ExternalExtensionRoots) != 2 || cfg.ExternalExtensionRoots[0] != "/srv/company-plugins" || cfg.ExternalExtensionRoots[1] != "/srv/community-plugins" {
		t.Fatalf("unexpected external extension roots: %v", cfg.ExternalExtensionRoots)
	}
}

func TestLoadFallsBackForInvalidWorkerConfig(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("DATABASE_MAX_CONNS", "bad")
	t.Setenv("MIGRATE_ON_STARTUP", "sometimes")
	t.Setenv("WORKER_SHUTDOWN_TIMEOUT", "bad")
	t.Setenv("JOB_QUEUE_SEARCH_WORKERS", "0")
	t.Setenv("SESSION_IDLE_TIMEOUT", "bad")
	t.Setenv("SESSION_ABSOLUTE_TIMEOUT", "0")
	t.Setenv("SESSION_RENEWAL_INTERVAL", "bad")
	t.Setenv("ALTCHA_CHALLENGE_TTL", "bad")
	t.Setenv("ALTCHA_COST", "0")

	cfg := Load()

	if cfg.DatabaseMaxConns != 10 {
		t.Fatalf("expected invalid database conns to fall back, got %d", cfg.DatabaseMaxConns)
	}
	if !cfg.MigrateOnStartup {
		t.Fatal("expected invalid startup migration value to fall back to enabled")
	}
	if cfg.WorkerShutdownTimeout != 30*time.Second {
		t.Fatalf("expected invalid shutdown timeout to fall back, got %s", cfg.WorkerShutdownTimeout)
	}
	if cfg.JobQueueSearchWorkers != 6 {
		t.Fatalf("expected non-positive queue workers to fall back, got %d", cfg.JobQueueSearchWorkers)
	}
	if cfg.SessionIdleTimeout != 30*24*time.Hour {
		t.Fatalf("expected invalid session idle timeout to fall back, got %s", cfg.SessionIdleTimeout)
	}
	if cfg.SessionAbsoluteTimeout != 180*24*time.Hour {
		t.Fatalf("expected non-positive session absolute timeout to fall back, got %s", cfg.SessionAbsoluteTimeout)
	}
	if cfg.SessionRenewalInterval != 24*time.Hour {
		t.Fatalf("expected invalid session renewal interval to fall back, got %s", cfg.SessionRenewalInterval)
	}
	if cfg.AltchaChallengeTTL != 10*time.Minute {
		t.Fatalf("expected invalid altcha ttl to fall back, got %s", cfg.AltchaChallengeTTL)
	}
	if cfg.AltchaCost != 1000 {
		t.Fatalf("expected non-positive altcha cost to fall back, got %d", cfg.AltchaCost)
	}
}

func TestLoadClampsSessionAbsoluteTimeoutToIdleTimeout(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("SESSION_IDLE_TIMEOUT", "24h")
	t.Setenv("SESSION_ABSOLUTE_TIMEOUT", "1h")

	cfg := Load()

	if cfg.SessionIdleTimeout != 24*time.Hour {
		t.Fatalf("expected session idle timeout 24h, got %s", cfg.SessionIdleTimeout)
	}
	if cfg.SessionAbsoluteTimeout != cfg.SessionIdleTimeout {
		t.Fatalf("expected session absolute timeout to clamp to idle timeout, got %s", cfg.SessionAbsoluteTimeout)
	}
}

// TestLoadCSRFTrustedOriginsFromEnv 验证 CSRF_TRUSTED_ORIGINS 逗号分隔解析与去重/去空白。
func TestLoadCSRFTrustedOriginsFromEnv(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("CSRF_TRUSTED_ORIGINS", " https://a.example.com ,https://b.example.com, https://a.example.com ,")

	cfg := Load()

	want := []string{"https://a.example.com", "https://b.example.com"}
	if len(cfg.CSRFTrustedOrigins) != len(want) {
		t.Fatalf("expected %d origins, got %#v", len(want), cfg.CSRFTrustedOrigins)
	}
	for i, o := range want {
		if cfg.CSRFTrustedOrigins[i] != o {
			t.Fatalf("origin %d = %q, want %q (full: %#v)", i, cfg.CSRFTrustedOrigins[i], o, cfg.CSRFTrustedOrigins)
		}
	}
}

// TestLoadCSRFTrustedOriginsDefaultsFromAppURL 验证未配置 CSRF_TRUSTED_ORIGINS 时，
// 从 APP_URL 派生 scheme://host 作为默认信任源（代理后 API 看到的 Host 与公开站点不同）。
func TestLoadCSRFTrustedOriginsDefaultsFromAppURL(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_URL", "https://forum.example.com/some/path")
	os.Unsetenv("CSRF_TRUSTED_ORIGINS")

	cfg := Load()

	if len(cfg.CSRFTrustedOrigins) != 1 || cfg.CSRFTrustedOrigins[0] != "https://forum.example.com" {
		t.Fatalf("expected default origin https://forum.example.com, got %#v", cfg.CSRFTrustedOrigins)
	}
}

// TestShouldUseSecureCookieAlignsSessionAndCSRF 与 session/CSRF 共用 Secure 判定。
func TestShouldUseSecureCookieAlignsSessionAndCSRF(t *testing.T) {
	if !ShouldUseSecureCookie(Config{AppEnv: "production", AppURL: "http://localhost"}) {
		t.Fatal("production must force Secure")
	}
	if !ShouldUseSecureCookie(Config{AppEnv: "staging", AppURL: "https://forum.example.com"}) {
		t.Fatal("https APP_URL must enable Secure outside production")
	}
	if ShouldUseSecureCookie(Config{AppEnv: "development", AppURL: "http://127.0.0.1:3000"}) {
		t.Fatal("plain http non-production must not force Secure")
	}
}

// TestOriginsFromAppURLHandlesInvalidInput 验证无效 APP_URL 不产生 origin。
func TestOriginsFromAppURLHandlesInvalidInput(t *testing.T) {
	for _, input := range []string{"", "   ", "not-a-url", "/relative/path"} {
		if got := originsFromAppURL(input); got != nil {
			t.Fatalf("originsFromAppURL(%q) = %#v, want nil", input, got)
		}
	}
}

// TestLoadRejectsInsecureSecretsInProduction 验证生产环境使用默认/占位密钥时拒绝启动。
func TestLoadRejectsInsecureSecretsInProduction(t *testing.T) {
	// 不设任何 secret 环境变量 → 全部命中占位默认值。
	os.Unsetenv("SESSION_HASH_SECRET")
	os.Unsetenv("ALTCHA_SECRET")
	os.Unsetenv("MEILI_MASTER_KEY")
	os.Unsetenv("APP_OPTION_ENC_KEY")
	os.Unsetenv("IDENTITY_SUBJECT_HMAC_SECRET")
	t.Setenv("APP_ENV", "production")

	// 设 placeholder 也应被拒。
	t.Setenv("SESSION_HASH_SECRET", "change-me")

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected Load to panic when production secrets are insecure, got no panic")
		}
	}()
	Load()
}

// TestLoadRejectsWeakIdentitySubjectHMACInProduction 走真实 APP_ENV=production 路径。
func TestLoadRejectsWeakIdentitySubjectHMACInProduction(t *testing.T) {
	setValidProductionSecrets(t)
	t.Setenv("APP_ENV", "production")
	// 开发默认必须被拒。
	t.Setenv("IDENTITY_SUBJECT_HMAC_SECRET", IdentitySubjectHMACDevDefault)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected Load to panic on identity subject hmac dev default in production")
		}
	}()
	Load()
}

// TestLoadRejectsShortIdentitySubjectHMACInProduction 生产拒绝短密钥。
func TestLoadRejectsShortIdentitySubjectHMACInProduction(t *testing.T) {
	setValidProductionSecrets(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("IDENTITY_SUBJECT_HMAC_SECRET", "tooshort")

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected Load to panic on short identity subject hmac secret in production")
		}
	}()
	Load()
}

// TestLoadAcceptsValidSecretsInProduction 验证生产环境配置有效密钥时正常加载。
func TestLoadAcceptsValidSecretsInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	setValidProductionSecrets(t)

	cfg := Load()
	if cfg.SessionHashSecret != "prod-valid-session-secret" {
		t.Fatalf("expected configured session secret, got %q", cfg.SessionHashSecret)
	}
	if cfg.IdentitySubjectHMACSecret != "prod-valid-identity-subject-hmac-secret-32b" {
		t.Fatalf("expected configured identity subject hmac secret, got %q", cfg.IdentitySubjectHMACSecret)
	}
}

// TestLoadDoesNotValidateSecretsInDevelopment 验证非生产环境允许默认密钥（开发友好）。
func TestLoadDoesNotValidateSecretsInDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	os.Unsetenv("SESSION_HASH_SECRET")
	os.Unsetenv("ALTCHA_SECRET")
	os.Unsetenv("MEILI_MASTER_KEY")
	os.Unsetenv("APP_OPTION_ENC_KEY")
	os.Unsetenv("IDENTITY_SUBJECT_HMAC_SECRET")

	cfg := Load() // 不应 panic
	if cfg.SessionHashSecret != "sforum-dev-session-hash-secret" {
		t.Fatalf("expected dev default secret, got %q", cfg.SessionHashSecret)
	}
	if cfg.IdentitySubjectHMACSecret != IdentitySubjectHMACDevDefault {
		t.Fatalf("expected stable identity subject hmac dev default, got %q", cfg.IdentitySubjectHMACSecret)
	}
}

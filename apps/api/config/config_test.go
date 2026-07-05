package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestLoadIncludesDefaultWorkerConfig(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("DATABASE_URL", "postgres://example:example@localhost:5432/example?sslmode=disable")

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
	if cfg.ExtensionRoot != "/var/lib/sforum/extensions" {
		t.Fatalf("expected extension root default, got %q", cfg.ExtensionRoot)
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

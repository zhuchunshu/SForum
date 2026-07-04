# Performance-First Jobs Queues Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first River/PostgreSQL-backed jobs framework for SForum, with explicit queue configuration, a dispatcher API, a worker runtime, and the first typed search indexing job contract.

**Architecture:** PostgreSQL remains the durable source for both app data and jobs. `internal/platform/jobs` wraps River so modules dispatch typed jobs through SForum-owned APIs, while `cmd/worker` starts the River client and consumes named queues. The first search job is implemented against a narrow `TopicIndexer` interface because forum tables and Meilisearch indexing are not implemented yet.

**Tech Stack:** Go 1.25+, River, River pgx v5 driver, `pgx/v5`, PostgreSQL, `log/slog`, Go unit tests.

---

## File Structure

- Modify `apps/api/go.mod` and `apps/api/go.sum`: add River and pgx dependencies.
- Modify `apps/api/internal/config/config.go`: add worker queue concurrency, worker shutdown timeout, and database pool sizing config.
- Create `apps/api/internal/config/config_test.go`: test default job/worker config parsing.
- Create `apps/api/internal/platform/postgres/pool.go`: build `pgxpool.Config` and open PostgreSQL pools.
- Create `apps/api/internal/platform/postgres/pool_test.go`: test pool config without connecting to a database.
- Create `apps/api/internal/platform/jobs/types.go`: define queue constants and shared enqueue options.
- Create `apps/api/internal/platform/jobs/config.go`: convert app config into River queue config.
- Create `apps/api/internal/platform/jobs/config_test.go`: test queue config and defaults.
- Create `apps/api/internal/platform/jobs/dispatcher.go`: expose SForum dispatch methods over River `Insert` and `InsertTx`.
- Create `apps/api/internal/platform/jobs/dispatcher_test.go`: verify queue options and transaction forwarding with a fake client.
- Create `apps/api/internal/platform/jobs/registry.go`: collect module worker registrations and build a River workers bundle.
- Create `apps/api/internal/platform/jobs/registry_test.go`: verify registration ordering and error behavior.
- Create `apps/api/internal/platform/jobs/runtime.go`: create and run the River client.
- Create `apps/api/internal/modules/search/jobs/index_topic.go`: define the first typed search indexing job and worker.
- Create `apps/api/internal/modules/search/jobs/index_topic_test.go`: test job kind, unique options, validation, and indexer invocation.
- Modify `apps/api/cmd/worker/main.go`: replace the placeholder worker loop with PostgreSQL pool setup, River client startup, signal handling, and graceful stop.
- Modify `docs/development-and-deployment.md`: add the River migration command and local worker verification notes.
- Modify `knowledge/modules/jobs.md`: record implementation status and operational commands.
- Create `knowledge/sessions/2026-07-04-jobs-queue-implementation-plan.md`: short handoff for the next session.

## External References

- River getting started: https://riverqueue.com/docs
- River migrations: https://riverqueue.com/docs/migrations
- River transactional enqueueing: https://riverqueue.com/docs/transactional-enqueueing
- River multiple queues: https://riverqueue.com/docs/multiple-queues
- River scheduled jobs: https://riverqueue.com/docs/scheduled-jobs
- River unique jobs: https://riverqueue.com/docs/unique-jobs
- River Go package docs: https://pkg.go.dev/github.com/riverqueue/river

---

### Task 1: Dependencies And Worker Config

**Files:**
- Modify: `apps/api/go.mod`
- Modify: `apps/api/go.sum`
- Modify: `apps/api/internal/config/config.go`
- Create: `apps/api/internal/config/config_test.go`

- [ ] **Step 1: Add River and pgx dependencies**

Run:

```bash
cd apps/api
go get github.com/jackc/pgx/v5 github.com/riverqueue/river github.com/riverqueue/river/riverdriver/riverpgxv5
```

Expected: `apps/api/go.mod` gains direct requirements for `github.com/jackc/pgx/v5`, `github.com/riverqueue/river`, and `github.com/riverqueue/river/riverdriver/riverpgxv5`; `apps/api/go.sum` updates.

- [ ] **Step 2: Write failing config tests**

Create `apps/api/internal/config/config_test.go`:

```go
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
	if cfg.WorkerDatabaseMaxConns != 10 {
		t.Fatalf("expected default worker database max conns 10, got %d", cfg.WorkerDatabaseMaxConns)
	}
	if cfg.WorkerShutdownTimeout != 30*time.Second {
		t.Fatalf("expected worker shutdown timeout 30s, got %s", cfg.WorkerShutdownTimeout)
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
	t.Setenv("WORKER_DATABASE_MAX_CONNS", "23")
	t.Setenv("WORKER_SHUTDOWN_TIMEOUT", "45s")
	t.Setenv("JOB_QUEUE_CRITICAL_WORKERS", "1")
	t.Setenv("JOB_QUEUE_DEFAULT_WORKERS", "2")
	t.Setenv("JOB_QUEUE_SEARCH_WORKERS", "3")
	t.Setenv("JOB_QUEUE_MAIL_WORKERS", "4")
	t.Setenv("JOB_QUEUE_NOTIFICATIONS_WORKERS", "5")
	t.Setenv("JOB_QUEUE_MAINTENANCE_WORKERS", "6")

	cfg := Load()

	if cfg.DatabaseMaxConns != 17 {
		t.Fatalf("expected api max conns from env, got %d", cfg.DatabaseMaxConns)
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
}

func TestLoadFallsBackForInvalidWorkerConfig(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("DATABASE_MAX_CONNS", "bad")
	t.Setenv("WORKER_SHUTDOWN_TIMEOUT", "bad")
	t.Setenv("JOB_QUEUE_SEARCH_WORKERS", "0")

	cfg := Load()

	if cfg.DatabaseMaxConns != 10 {
		t.Fatalf("expected invalid database conns to fall back, got %d", cfg.DatabaseMaxConns)
	}
	if cfg.WorkerShutdownTimeout != 30*time.Second {
		t.Fatalf("expected invalid shutdown timeout to fall back, got %s", cfg.WorkerShutdownTimeout)
	}
	if cfg.JobQueueSearchWorkers != 6 {
		t.Fatalf("expected non-positive queue workers to fall back, got %d", cfg.JobQueueSearchWorkers)
	}
}
```

- [ ] **Step 3: Run config tests and verify they fail**

Run:

```bash
cd apps/api
go test ./internal/config
```

Expected: FAIL because `Config` does not yet have the worker/database fields and parse helpers.

- [ ] **Step 4: Implement worker config parsing**

Modify `apps/api/internal/config/config.go` to:

```go
package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/inkedus/sforum/apps/api/internal/modules/localization"
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
		DatabaseMaxConns:             int32(envInt("DATABASE_MAX_CONNS", 10)),
		WorkerDatabaseMaxConns:       int32(envInt("WORKER_DATABASE_MAX_CONNS", 10)),
		WorkerShutdownTimeout:        envDuration("WORKER_SHUTDOWN_TIMEOUT", 30*time.Second),
		RedisAddr:                    env("REDIS_ADDR", "redis:6379"),
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
	parsed := envInt(key, fallback)
	if parsed <= 0 {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	if parsed <= 0 {
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
```

- [ ] **Step 5: Run config tests and commit**

Run:

```bash
cd apps/api
go test ./internal/config
```

Expected: PASS.

Commit:

```bash
git add apps/api/go.mod apps/api/go.sum apps/api/internal/config/config.go apps/api/internal/config/config_test.go
git commit -m "feat(api): add worker queue config"
```

---

### Task 2: PostgreSQL Pool Helper

**Files:**
- Create: `apps/api/internal/platform/postgres/pool.go`
- Create: `apps/api/internal/platform/postgres/pool_test.go`

- [ ] **Step 1: Write failing pool config tests**

Create `apps/api/internal/platform/postgres/pool_test.go`:

```go
package postgres

import "testing"

func TestBuildPoolConfigAppliesMaxConns(t *testing.T) {
	cfg, err := BuildPoolConfig("postgres://sforum:sforum@localhost:5432/sforum?sslmode=disable", 12)
	if err != nil {
		t.Fatalf("build pool config: %v", err)
	}

	if cfg.MaxConns != 12 {
		t.Fatalf("expected max conns 12, got %d", cfg.MaxConns)
	}
	if cfg.ConnConfig.Database != "sforum" {
		t.Fatalf("expected database sforum, got %q", cfg.ConnConfig.Database)
	}
}

func TestBuildPoolConfigIgnoresNonPositiveMaxConns(t *testing.T) {
	cfg, err := BuildPoolConfig("postgres://sforum:sforum@localhost:5432/sforum?sslmode=disable", 0)
	if err != nil {
		t.Fatalf("build pool config: %v", err)
	}

	if cfg.MaxConns <= 0 {
		t.Fatalf("expected pgx default max conns to remain positive, got %d", cfg.MaxConns)
	}
}

func TestBuildPoolConfigRejectsInvalidURL(t *testing.T) {
	if _, err := BuildPoolConfig("://not-a-postgres-url", 10); err == nil {
		t.Fatal("expected invalid database URL to fail")
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
cd apps/api
go test ./internal/platform/postgres
```

Expected: FAIL because `BuildPoolConfig` is not defined.

- [ ] **Step 3: Implement pool helper**

Create `apps/api/internal/platform/postgres/pool.go`:

```go
package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func BuildPoolConfig(databaseURL string, maxConns int32) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}
	cfg.HealthCheckPeriod = 30 * time.Second
	return cfg, nil
}

func Open(ctx context.Context, databaseURL string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := BuildPoolConfig(databaseURL, maxConns)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}
```

- [ ] **Step 4: Run pool tests and commit**

Run:

```bash
cd apps/api
go test ./internal/platform/postgres
```

Expected: PASS.

Commit:

```bash
git add apps/api/internal/platform/postgres/pool.go apps/api/internal/platform/postgres/pool_test.go
git commit -m "feat(api): add postgres pool helper"
```

---

### Task 3: Queue Constants And River Queue Config

**Files:**
- Create: `apps/api/internal/platform/jobs/types.go`
- Create: `apps/api/internal/platform/jobs/config.go`
- Create: `apps/api/internal/platform/jobs/config_test.go`

- [ ] **Step 1: Write failing queue config tests**

Create `apps/api/internal/platform/jobs/config_test.go`:

```go
package jobs

import (
	"testing"

	"github.com/riverqueue/river"

	"github.com/inkedus/sforum/apps/api/internal/config"
)

func TestFromAppConfigBuildsRiverQueues(t *testing.T) {
	cfg := FromAppConfig(config.Config{
		JobQueueCriticalWorkers:      1,
		JobQueueDefaultWorkers:       2,
		JobQueueSearchWorkers:        3,
		JobQueueMailWorkers:          4,
		JobQueueNotificationsWorkers: 5,
		JobQueueMaintenanceWorkers:   6,
	})

	queues := cfg.RiverQueues()

	assertWorkers(t, queues, QueueCritical, 1)
	assertWorkers(t, queues, QueueDefault, 2)
	assertWorkers(t, queues, QueueSearch, 3)
	assertWorkers(t, queues, QueueMail, 4)
	assertWorkers(t, queues, QueueNotifications, 5)
	assertWorkers(t, queues, QueueMaintenance, 6)
}

func TestFromAppConfigUsesSafeDefaults(t *testing.T) {
	cfg := FromAppConfig(config.Config{})

	queues := cfg.RiverQueues()

	assertWorkers(t, queues, QueueCritical, 4)
	assertWorkers(t, queues, QueueDefault, 8)
	assertWorkers(t, queues, QueueSearch, 6)
	assertWorkers(t, queues, QueueMail, 4)
	assertWorkers(t, queues, QueueNotifications, 6)
	assertWorkers(t, queues, QueueMaintenance, 2)
}

func assertWorkers(t *testing.T, queues map[string]river.QueueConfig, name string, expected int) {
	t.Helper()

	queue, ok := queues[name]
	if !ok {
		t.Fatalf("expected queue %q to exist", name)
	}
	if queue.MaxWorkers != expected {
		t.Fatalf("expected queue %q workers %d, got %d", name, expected, queue.MaxWorkers)
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
cd apps/api
go test ./internal/platform/jobs
```

Expected: FAIL because the jobs package does not exist.

- [ ] **Step 3: Add queue constants and enqueue option type**

Create `apps/api/internal/platform/jobs/types.go`:

```go
package jobs

import (
	"time"

	"github.com/riverqueue/river"
)

const (
	QueueCritical      = "critical"
	QueueDefault       = river.QueueDefault
	QueueSearch        = "search"
	QueueMail          = "mail"
	QueueNotifications = "notifications"
	QueueMaintenance   = "maintenance"
)

type EnqueueOptions struct {
	Queue       string
	MaxAttempts int
	ScheduledAt time.Time
	Unique      river.UniqueOpts
}

func (opts EnqueueOptions) RiverInsertOpts() *river.InsertOpts {
	insertOpts := &river.InsertOpts{}
	if opts.Queue != "" {
		insertOpts.Queue = opts.Queue
	}
	if opts.MaxAttempts > 0 {
		insertOpts.MaxAttempts = opts.MaxAttempts
	}
	if !opts.ScheduledAt.IsZero() {
		insertOpts.ScheduledAt = opts.ScheduledAt
	}
	insertOpts.UniqueOpts = opts.Unique
	return insertOpts
}
```

- [ ] **Step 4: Add queue config conversion**

Create `apps/api/internal/platform/jobs/config.go`:

```go
package jobs

import (
	"github.com/riverqueue/river"

	"github.com/inkedus/sforum/apps/api/internal/config"
)

type Config struct {
	CriticalWorkers      int
	DefaultWorkers       int
	SearchWorkers        int
	MailWorkers          int
	NotificationsWorkers int
	MaintenanceWorkers   int
}

func FromAppConfig(cfg config.Config) Config {
	return Config{
		CriticalWorkers:      positiveOrDefault(cfg.JobQueueCriticalWorkers, 4),
		DefaultWorkers:       positiveOrDefault(cfg.JobQueueDefaultWorkers, 8),
		SearchWorkers:        positiveOrDefault(cfg.JobQueueSearchWorkers, 6),
		MailWorkers:          positiveOrDefault(cfg.JobQueueMailWorkers, 4),
		NotificationsWorkers: positiveOrDefault(cfg.JobQueueNotificationsWorkers, 6),
		MaintenanceWorkers:   positiveOrDefault(cfg.JobQueueMaintenanceWorkers, 2),
	}
}

func (cfg Config) RiverQueues() map[string]river.QueueConfig {
	return map[string]river.QueueConfig{
		QueueCritical:      {MaxWorkers: cfg.CriticalWorkers},
		QueueDefault:       {MaxWorkers: cfg.DefaultWorkers},
		QueueSearch:        {MaxWorkers: cfg.SearchWorkers},
		QueueMail:          {MaxWorkers: cfg.MailWorkers},
		QueueNotifications: {MaxWorkers: cfg.NotificationsWorkers},
		QueueMaintenance:   {MaxWorkers: cfg.MaintenanceWorkers},
	}
}

func positiveOrDefault(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
```

- [ ] **Step 5: Run jobs tests and commit**

Run:

```bash
cd apps/api
go test ./internal/platform/jobs
```

Expected: PASS.

Commit:

```bash
git add apps/api/internal/platform/jobs/types.go apps/api/internal/platform/jobs/config.go apps/api/internal/platform/jobs/config_test.go
git commit -m "feat(api): add job queue config"
```

---

### Task 4: Dispatcher API

**Files:**
- Create: `apps/api/internal/platform/jobs/dispatcher.go`
- Create: `apps/api/internal/platform/jobs/dispatcher_test.go`

- [ ] **Step 1: Write failing dispatcher tests**

Create `apps/api/internal/platform/jobs/dispatcher_test.go`:

```go
package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type testArgs struct {
	ID int64 `json:"id" river:"unique"`
}

func (testArgs) Kind() string { return "test.args" }

type fakeRiverClient struct {
	insertArgs river.JobArgs
	insertOpts *river.InsertOpts
	insertTx   pgx.Tx
	err        error
}

func (f *fakeRiverClient) Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	f.insertArgs = args
	f.insertOpts = opts
	return &rivertype.JobInsertResult{}, f.err
}

func (f *fakeRiverClient) InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	f.insertTx = tx
	f.insertArgs = args
	f.insertOpts = opts
	return &rivertype.JobInsertResult{}, f.err
}

func TestDispatcherEnqueueConvertsOptions(t *testing.T) {
	client := &fakeRiverClient{}
	dispatcher := NewDispatcher(client)
	runAt := time.Now().UTC().Add(time.Hour)

	_, err := dispatcher.Enqueue(context.Background(), testArgs{ID: 42}, EnqueueOptions{
		Queue:       QueueSearch,
		MaxAttempts: 3,
		ScheduledAt: runAt,
		Unique:      river.UniqueOpts{ByArgs: true},
	})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if client.insertArgs.Kind() != "test.args" {
		t.Fatalf("expected args kind test.args, got %q", client.insertArgs.Kind())
	}
	if client.insertOpts.Queue != QueueSearch {
		t.Fatalf("expected search queue, got %q", client.insertOpts.Queue)
	}
	if client.insertOpts.MaxAttempts != 3 {
		t.Fatalf("expected max attempts 3, got %d", client.insertOpts.MaxAttempts)
	}
	if !client.insertOpts.ScheduledAt.Equal(runAt) {
		t.Fatalf("expected scheduled time %s, got %s", runAt, client.insertOpts.ScheduledAt)
	}
	if !client.insertOpts.UniqueOpts.ByArgs {
		t.Fatal("expected unique by args")
	}
}

func TestDispatcherEnqueueReturnsClientErrors(t *testing.T) {
	expected := errors.New("insert failed")
	dispatcher := NewDispatcher(&fakeRiverClient{err: expected})

	if _, err := dispatcher.Enqueue(context.Background(), testArgs{ID: 1}, EnqueueOptions{}); !errors.Is(err, expected) {
		t.Fatalf("expected insert error, got %v", err)
	}
}
```

- [ ] **Step 2: Run dispatcher tests and verify they fail**

Run:

```bash
cd apps/api
go test ./internal/platform/jobs
```

Expected: FAIL because `NewDispatcher` is not defined.

- [ ] **Step 3: Implement dispatcher**

Create `apps/api/internal/platform/jobs/dispatcher.go`:

```go
package jobs

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type RiverClient interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
	InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

type Dispatcher struct {
	client RiverClient
}

func NewDispatcher(client RiverClient) *Dispatcher {
	return &Dispatcher{client: client}
}

func (d *Dispatcher) Enqueue(ctx context.Context, args river.JobArgs, opts EnqueueOptions) (*rivertype.JobInsertResult, error) {
	return d.client.Insert(ctx, args, opts.RiverInsertOpts())
}

func (d *Dispatcher) EnqueueTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts EnqueueOptions) (*rivertype.JobInsertResult, error) {
	return d.client.InsertTx(ctx, tx, args, opts.RiverInsertOpts())
}
```

- [ ] **Step 4: Run dispatcher tests and commit**

Run:

```bash
cd apps/api
go test ./internal/platform/jobs
```

Expected: PASS.

Commit:

```bash
git add apps/api/internal/platform/jobs/dispatcher.go apps/api/internal/platform/jobs/dispatcher_test.go
git commit -m "feat(api): add job dispatcher"
```

---

### Task 5: Worker Registry And Runtime

**Files:**
- Create: `apps/api/internal/platform/jobs/registry.go`
- Create: `apps/api/internal/platform/jobs/registry_test.go`
- Create: `apps/api/internal/platform/jobs/runtime.go`

- [ ] **Step 1: Write failing registry tests**

Create `apps/api/internal/platform/jobs/registry_test.go`:

```go
package jobs

import (
	"errors"
	"testing"

	"github.com/riverqueue/river"
)

func TestRegistryBuildsWorkers(t *testing.T) {
	registry := NewRegistry()
	called := false
	registry.Add(func(workers *river.Workers) error {
		called = true
		return nil
	})

	workers, err := registry.Build()
	if err != nil {
		t.Fatalf("build workers: %v", err)
	}
	if workers == nil {
		t.Fatal("expected workers bundle")
	}
	if !called {
		t.Fatal("expected registrar to be called")
	}
}

func TestRegistryReturnsRegistrarError(t *testing.T) {
	expected := errors.New("bad registration")
	registry := NewRegistry()
	registry.Add(func(workers *river.Workers) error {
		return expected
	})

	if _, err := registry.Build(); !errors.Is(err, expected) {
		t.Fatalf("expected registrar error, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
cd apps/api
go test ./internal/platform/jobs
```

Expected: FAIL because `NewRegistry` is not defined.

- [ ] **Step 3: Implement registry**

Create `apps/api/internal/platform/jobs/registry.go`:

```go
package jobs

import "github.com/riverqueue/river"

type Registrar func(workers *river.Workers) error

type Registry struct {
	registrars []Registrar
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Add(registrar Registrar) {
	r.registrars = append(r.registrars, registrar)
}

func (r *Registry) Build() (*river.Workers, error) {
	workers := river.NewWorkers()
	for _, registrar := range r.registrars {
		if err := registrar(workers); err != nil {
			return nil, err
		}
	}
	return workers, nil
}
```

- [ ] **Step 4: Implement runtime client construction**

Create `apps/api/internal/platform/jobs/runtime.go`:

```go
package jobs

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type Client = river.Client[pgx.Tx]

func NewClient(pool *pgxpool.Pool, cfg Config, workers *river.Workers) (*Client, error) {
	return river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  cfg.RiverQueues(),
		Workers: workers,
	})
}

func Start(ctx context.Context, client *Client) error {
	return client.Start(ctx)
}

func Stop(ctx context.Context, client *Client) error {
	return client.Stop(ctx)
}
```

- [ ] **Step 5: Run jobs tests and commit**

Run:

```bash
cd apps/api
go test ./internal/platform/jobs
```

Expected: PASS.

Commit:

```bash
git add apps/api/internal/platform/jobs/registry.go apps/api/internal/platform/jobs/registry_test.go apps/api/internal/platform/jobs/runtime.go
git commit -m "feat(api): add job worker runtime"
```

---

### Task 6: Search Index Topic Job Contract

**Files:**
- Create: `apps/api/internal/modules/search/jobs/index_topic.go`
- Create: `apps/api/internal/modules/search/jobs/index_topic_test.go`

- [ ] **Step 1: Write failing search job tests**

Create `apps/api/internal/modules/search/jobs/index_topic_test.go`:

```go
package jobs

import (
	"context"
	"testing"

	"github.com/riverqueue/river"

	platformjobs "github.com/inkedus/sforum/apps/api/internal/platform/jobs"
)

type fakeTopicIndexer struct {
	topicID int64
	err     error
}

func (f *fakeTopicIndexer) IndexTopic(ctx context.Context, topicID int64) error {
	f.topicID = topicID
	return f.err
}

func TestIndexTopicArgsKindAndOptions(t *testing.T) {
	args := IndexTopicArgs{TopicID: 42}

	if args.Kind() != "search.index_topic" {
		t.Fatalf("expected search.index_topic kind, got %q", args.Kind())
	}

	opts := args.EnqueueOptions()
	if opts.Queue != platformjobs.QueueSearch {
		t.Fatalf("expected search queue, got %q", opts.Queue)
	}
	if !opts.Unique.ByArgs {
		t.Fatal("expected unique by args")
	}
	if opts.MaxAttempts != 10 {
		t.Fatalf("expected max attempts 10, got %d", opts.MaxAttempts)
	}
}

func TestIndexTopicWorkerCallsIndexer(t *testing.T) {
	indexer := &fakeTopicIndexer{}
	worker := &IndexTopicWorker{Indexer: indexer}

	err := worker.Work(context.Background(), &river.Job[IndexTopicArgs]{
		Args: IndexTopicArgs{TopicID: 42},
	})
	if err != nil {
		t.Fatalf("work: %v", err)
	}
	if indexer.topicID != 42 {
		t.Fatalf("expected topic 42, got %d", indexer.topicID)
	}
}

func TestIndexTopicWorkerRejectsInvalidTopicID(t *testing.T) {
	worker := &IndexTopicWorker{Indexer: &fakeTopicIndexer{}}

	err := worker.Work(context.Background(), &river.Job[IndexTopicArgs]{
		Args: IndexTopicArgs{TopicID: 0},
	})
	if err == nil {
		t.Fatal("expected invalid topic id error")
	}
}

func TestRegisterAddsWorker(t *testing.T) {
	registry := platformjobs.NewRegistry()
	Register(registry, &fakeTopicIndexer{})

	workers, err := registry.Build()
	if err != nil {
		t.Fatalf("build workers: %v", err)
	}
	if workers == nil {
		t.Fatal("expected workers bundle")
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
cd apps/api
go test ./internal/modules/search/jobs
```

Expected: FAIL because the search jobs package does not exist.

- [ ] **Step 3: Implement search job**

Create `apps/api/internal/modules/search/jobs/index_topic.go`:

```go
package jobs

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"

	platformjobs "github.com/inkedus/sforum/apps/api/internal/platform/jobs"
)

type TopicIndexer interface {
	IndexTopic(ctx context.Context, topicID int64) error
}

type IndexTopicArgs struct {
	TopicID int64 `json:"topic_id" river:"unique"`
}

func (IndexTopicArgs) Kind() string {
	return "search.index_topic"
}

func (IndexTopicArgs) EnqueueOptions() platformjobs.EnqueueOptions {
	return platformjobs.EnqueueOptions{
		Queue:       platformjobs.QueueSearch,
		MaxAttempts: 10,
		Unique:      river.UniqueOpts{ByArgs: true},
	}
}

type IndexTopicWorker struct {
	river.WorkerDefaults[IndexTopicArgs]
	Indexer TopicIndexer
}

func (w *IndexTopicWorker) Work(ctx context.Context, job *river.Job[IndexTopicArgs]) error {
	if job.Args.TopicID <= 0 {
		return fmt.Errorf("index topic job requires positive topic id: %d", job.Args.TopicID)
	}
	if w.Indexer == nil {
		return fmt.Errorf("index topic worker requires indexer")
	}
	return w.Indexer.IndexTopic(ctx, job.Args.TopicID)
}

func Register(registry *platformjobs.Registry, indexer TopicIndexer) {
	registry.Add(func(workers *river.Workers) error {
		return river.AddWorkerSafely[IndexTopicArgs](workers, &IndexTopicWorker{Indexer: indexer})
	})
}
```

- [ ] **Step 4: Run search job tests and commit**

Run:

```bash
cd apps/api
go test ./internal/modules/search/jobs
```

Expected: PASS.

Commit:

```bash
git add apps/api/internal/modules/search/jobs/index_topic.go apps/api/internal/modules/search/jobs/index_topic_test.go
git commit -m "feat(api): add search index topic job"
```

---

### Task 7: Worker Process Runtime

**Files:**
- Modify: `apps/api/cmd/worker/main.go`

- [ ] **Step 1: Build before changing worker**

Run:

```bash
cd apps/api
go build ./cmd/worker
```

Expected: PASS with the existing placeholder worker.

- [ ] **Step 2: Replace worker placeholder with River runtime**

Modify `apps/api/cmd/worker/main.go`:

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/inkedus/sforum/apps/api/internal/config"
	platformjobs "github.com/inkedus/sforum/apps/api/internal/platform/jobs"
	"github.com/inkedus/sforum/apps/api/internal/platform/postgres"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbPool, err := postgres.Open(ctx, cfg.DatabaseURL, cfg.WorkerDatabaseMaxConns)
	if err != nil {
		logger.Error("worker database connection failed", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	registry := platformjobs.NewRegistry()
	workers, err := registry.Build()
	if err != nil {
		logger.Error("worker registration failed", "error", err)
		os.Exit(1)
	}

	riverClient, err := platformjobs.NewClient(dbPool, platformjobs.FromAppConfig(cfg), workers)
	if err != nil {
		logger.Error("worker client creation failed", "error", err)
		os.Exit(1)
	}

	logger.Info("starting worker", "env", cfg.AppEnv, "locale", cfg.AppLocale)
	if err := platformjobs.Start(ctx, riverClient); err != nil {
		logger.Error("worker start failed", "error", err)
		os.Exit(1)
	}

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.WorkerShutdownTimeout)
	defer cancel()

	if err := platformjobs.Stop(shutdownCtx, riverClient); err != nil {
		logger.Error("worker shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("worker stopped")
}
```

- [ ] **Step 3: Build worker and run API tests**

Run:

```bash
cd apps/api
go build ./cmd/worker
go test ./...
```

Expected: PASS. The worker build does not require connecting to PostgreSQL because it only compiles the command.

- [ ] **Step 4: Commit worker runtime**

Commit:

```bash
git add apps/api/cmd/worker/main.go
git commit -m "feat(api): start river worker runtime"
```

---

### Task 8: River Migration And Local Verification Notes

**Files:**
- Modify: `docs/development-and-deployment.md`
- Modify: `knowledge/modules/jobs.md`
- Create: `knowledge/sessions/2026-07-04-jobs-queue-implementation-plan.md`

- [ ] **Step 1: Update development docs with River migration command**

Add this section under `## Jobs And Worker Runtime` in `docs/development-and-deployment.md`:

```md
### River Migrations

River owns its internal job tables and runs its own migration line. Before a
worker can consume durable jobs in a new database, run:

```sh
cd apps/api
go install github.com/riverqueue/river/cmd/river@latest
river migrate-up --line main --database-url "$DATABASE_URL"
```

Use `river migrate-list --line main --database-url "$DATABASE_URL"` to inspect
which River migrations have been applied. Do not use `river migrate-down` in
production unless an operator has already accepted that it removes River job
tables and queued jobs.
```

- [ ] **Step 2: Update jobs module status**

Change `knowledge/modules/jobs.md` current status to:

```md
## Current Status

Architecture accepted on 2026-07-04. Implementation plan added on 2026-07-04.

The selected durable queue foundation is River backed by PostgreSQL.
```

Add this under `## Next Steps`:

```md
- Run River migrations before starting workers against a fresh database.
- Wire module registrations into `cmd/worker` as real domain jobs become
  available.
```

- [ ] **Step 3: Add session handoff**

Create `knowledge/sessions/2026-07-04-jobs-queue-implementation-plan.md`:

```md
# 2026-07-04 Jobs Queue Implementation Plan

## Changed

- Added the implementation plan for the River/PostgreSQL jobs framework.

## Decisions

- The first implementation slice builds the platform framework, worker runtime,
  and a typed search indexing job contract.
- The search job uses a `TopicIndexer` interface because forum tables and the
  Meilisearch indexer are not implemented yet.

## Next

- Execute `docs/superpowers/plans/2026-07-04-performance-first-jobs-queues.md`.
- Run River migrations before testing workers against a real database.

## Open Questions

- Which module will first dispatch `search.index_topic` once topic writes
  exist.
- Whether River migrations should be wrapped by the future SForum migrate
  command or left as an explicit River CLI call.
```

- [ ] **Step 4: Run docs verification**

Run:

```bash
rg -n "River|JOB_QUEUE|search.index_topic" docs/development-and-deployment.md knowledge/modules/jobs.md knowledge/sessions/2026-07-04-jobs-queue-implementation-plan.md
```

Expected: output includes the River migration command, queue environment variables, and `search.index_topic`.

- [ ] **Step 5: Commit docs**

Commit:

```bash
git add docs/development-and-deployment.md knowledge/modules/jobs.md knowledge/sessions/2026-07-04-jobs-queue-implementation-plan.md
git commit -m "docs: document jobs queue implementation plan"
```

---

### Task 9: Full Verification

**Files:**
- Verify: `apps/api`
- Verify: `docs/superpowers/plans/2026-07-04-performance-first-jobs-queues.md`

- [ ] **Step 1: Run all API tests**

Run:

```bash
cd apps/api
go test ./...
```

Expected: PASS.

- [ ] **Step 2: Build API and worker commands**

Run:

```bash
cd apps/api
go build ./cmd/api
go build ./cmd/worker
```

Expected: both commands compile.

- [ ] **Step 3: Verify plan text has no placeholders**

Run:

```bash
rg -n "$(printf '%s' 'TB''D|TO''DO|implement ''later|fill in ''details|Similar to ''Task|appropriate ''error')" docs/superpowers/plans/2026-07-04-performance-first-jobs-queues.md
```

Expected: no output.

- [ ] **Step 4: Verify git state**

Run:

```bash
git status --short
```

Expected: only user-owned unrelated changes remain. Do not stage or commit unrelated frontend or AGENTS changes.

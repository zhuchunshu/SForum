package adminoverview

import (
	"context"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	health "github.com/zhuchunshu/sforum/apps/api/app/Support/Health"
)

type RuntimeCollector struct {
	startedAt    time.Time
	pool         *pgxpool.Pool
	heartbeat    health.HeartbeatStore
	queueLagPool *pgxpool.Pool
	staleAfter   time.Duration
	now          func() time.Time
}

func NewRuntimeCollector(startedAt time.Time, pool *pgxpool.Pool) RuntimeCollector {
	return RuntimeCollector{startedAt: startedAt.UTC(), pool: pool, staleAfter: health.DefaultHeartbeatStaleAfter, now: time.Now}
}

// WithHeartbeat 注入 worker last_seen 读取（Redis）。
func (c RuntimeCollector) WithHeartbeat(store health.HeartbeatStore) RuntimeCollector {
	c.heartbeat = store
	return c
}

// WithQueueLag 启用廉价 River 队列积压统计（共用 PG pool）。
func (c RuntimeCollector) WithQueueLag(pool *pgxpool.Pool) RuntimeCollector {
	c.queueLagPool = pool
	return c
}

func (c RuntimeCollector) Snapshot() RuntimeStats {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	lastPause := uint64(0)
	if stats.NumGC > 0 {
		lastPause = stats.PauseNs[(stats.NumGC+255)%256]
	}

	nowFn := c.now
	if nowFn == nil {
		nowFn = time.Now
	}

	return RuntimeStats{
		StartedAt:      c.startedAt,
		UptimeSeconds:  int64(time.Since(c.startedAt).Seconds()),
		MemoryBytes:    stats.Sys,
		HeapAllocBytes: stats.HeapAlloc,
		HeapSysBytes:   stats.HeapSys,
		GoroutineCount: runtime.NumGoroutine(),
		GCCount:        stats.NumGC,
		LastGCPauseNs:  lastPause,
		Database:       c.databaseStats(),
		Worker:         c.workerStats(nowFn().UTC()),
		QueueLag:       c.queueLagStats(),
	}
}

func (c RuntimeCollector) databaseStats() DatabaseRuntimeStats {
	if c.pool == nil {
		return DatabaseRuntimeStats{}
	}
	stats := c.pool.Stat()
	return DatabaseRuntimeStats{
		MaxConnections:      stats.MaxConns(),
		TotalConnections:    stats.TotalConns(),
		AcquiredConnections: stats.AcquiredConns(),
		IdleConnections:     stats.IdleConns(),
	}
}

func (c RuntimeCollector) workerStats(now time.Time) *WorkerRuntimeStats {
	if c.heartbeat == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	lastSeen, found, err := c.heartbeat.LastSeen(ctx)
	if err != nil {
		// Redis 读失败：标 unknown，避免 overview 整体失败。
		return &WorkerRuntimeStats{Stale: true, Status: "unknown"}
	}
	observed := health.ObserveHeartbeat(lastSeen, found, now, c.staleAfter)
	return &WorkerRuntimeStats{
		LastSeenAt: observed.LastSeenAt,
		AgeSeconds: observed.AgeSeconds,
		Stale:      observed.Stale,
		Status:     observed.Status,
	}
}

func (c RuntimeCollector) queueLagStats() *QueueLagStats {
	pool := c.queueLagPool
	if pool == nil {
		pool = c.pool
	}
	if pool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	var waiting, running, failed int64
	// River 官方表；失败时静默返回 nil，不拖垮 overview。
	err := pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE state IN ('available', 'scheduled', 'retryable', 'pending')),
			count(*) FILTER (WHERE state = 'running'),
			count(*) FILTER (WHERE state = 'discarded')
		FROM river_job
	`).Scan(&waiting, &running, &failed)
	if err != nil {
		return nil
	}
	return &QueueLagStats{Waiting: waiting, Running: running, Failed: failed}
}

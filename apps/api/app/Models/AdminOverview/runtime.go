package adminoverview

import (
	"context"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	health "github.com/zhuchunshu/sforum/apps/api/app/Support/Health"
	platformversion "github.com/zhuchunshu/sforum/apps/api/version"
)

type RuntimeCollector struct {
	startedAt    time.Time
	pool         *pgxpool.Pool
	heartbeat    health.HeartbeatStore
	queueLagPool *pgxpool.Pool
	staleAfter   time.Duration
	now          func() time.Time
	// sampler 可注入；nil 时使用 defaultProcessSampler。
	sampler ProcessSampler
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

// WithProcessSampler 注入进程内存采样（测试用）。
func (c RuntimeCollector) WithProcessSampler(sampler ProcessSampler) RuntimeCollector {
	c.sampler = sampler
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

	sampler := c.sampler
	if sampler == nil {
		sampler = defaultProcessSampler
	}

	memoryBytes := uint64(0)
	var familyPtr *uint64
	pluginChildren := 0
	if selfRSS, familyRSS, children, ok := sampleSelfAndFamily(sampler); ok {
		memoryBytes = selfRSS
		pluginChildren = children
		family := familyRSS
		familyPtr = &family
	} else if fallback, ok := readSelfRSSFallback(); ok {
		// ps 失败时 Linux 仍可读 /proc/self/statm；全家内存省略。
		memoryBytes = fallback
	} else {
		// 最后回退：避免主 KPI 为 0；语义上仍暴露 Sys 诊断字段。
		memoryBytes = stats.Sys
	}

	return RuntimeStats{
		StartedAt:         c.startedAt,
		UptimeSeconds:     int64(time.Since(c.startedAt).Seconds()),
		Build:             platformversion.Get(),
		MemoryBytes:       memoryBytes,
		HeapAllocBytes:    stats.HeapAlloc,
		HeapSysBytes:      stats.HeapSys,
		SysBytes:          stats.Sys,
		FamilyMemoryBytes: familyPtr,
		PluginChildCount:  pluginChildren,
		GoroutineCount:    runtime.NumGoroutine(),
		GCCount:           stats.NumGC,
		LastGCPauseNs:     lastPause,
		Database:          c.databaseStats(),
		Worker:            c.workerStats(nowFn().UTC()),
		QueueLag:          c.queueLagStats(),
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

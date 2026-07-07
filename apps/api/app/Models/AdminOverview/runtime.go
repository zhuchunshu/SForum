package adminoverview

import (
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RuntimeCollector struct {
	startedAt time.Time
	pool      *pgxpool.Pool
}

func NewRuntimeCollector(startedAt time.Time, pool *pgxpool.Pool) RuntimeCollector {
	return RuntimeCollector{startedAt: startedAt.UTC(), pool: pool}
}

func (c RuntimeCollector) Snapshot() RuntimeStats {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	lastPause := uint64(0)
	if stats.NumGC > 0 {
		lastPause = stats.PauseNs[(stats.NumGC+255)%256]
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

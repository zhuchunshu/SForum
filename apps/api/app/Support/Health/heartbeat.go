package health

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// 默认策略：worker 每 10s 写一次 Redis key，TTL 45s；超过 45s 无心跳视为 stale。
const (
	DefaultHeartbeatKey        = "sforum:worker:heartbeat"
	DefaultHeartbeatInterval   = 10 * time.Second
	DefaultHeartbeatTTL        = 45 * time.Second
	DefaultHeartbeatStaleAfter = 45 * time.Second
)

// HeartbeatStore 读写 worker last_seen。
type HeartbeatStore interface {
	// Touch 写入当前时间（Unix 秒），并刷新 TTL。
	Touch(ctx context.Context, at time.Time) error
	// LastSeen 返回最近心跳时间；found=false 表示无记录。
	LastSeen(ctx context.Context) (at time.Time, found bool, err error)
}

// RedisHeartbeatStore 用 Redis string 存 Unix 秒时间戳。
type RedisHeartbeatStore struct {
	Client *redis.Client
	Key    string
	TTL    time.Duration
}

func NewRedisHeartbeatStore(client *redis.Client) *RedisHeartbeatStore {
	return &RedisHeartbeatStore{
		Client: client,
		Key:    DefaultHeartbeatKey,
		TTL:    DefaultHeartbeatTTL,
	}
}

func (s *RedisHeartbeatStore) Touch(ctx context.Context, at time.Time) error {
	if s == nil || s.Client == nil {
		return fmt.Errorf("heartbeat store is not configured")
	}
	key := s.Key
	if key == "" {
		key = DefaultHeartbeatKey
	}
	ttl := s.TTL
	if ttl <= 0 {
		ttl = DefaultHeartbeatTTL
	}
	return s.Client.Set(ctx, key, strconv.FormatInt(at.UTC().Unix(), 10), ttl).Err()
}

func (s *RedisHeartbeatStore) LastSeen(ctx context.Context) (time.Time, bool, error) {
	if s == nil || s.Client == nil {
		return time.Time{}, false, fmt.Errorf("heartbeat store is not configured")
	}
	key := s.Key
	if key == "" {
		key = DefaultHeartbeatKey
	}
	raw, err := s.Client.Get(ctx, key).Result()
	if err == redis.Nil {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	sec, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("parse heartbeat: %w", err)
	}
	return time.Unix(sec, 0).UTC(), true, nil
}

// WorkerHeartbeat 是 admin / ready 展示用的 worker 状态投影。
type WorkerHeartbeat struct {
	LastSeenAt *time.Time `json:"lastSeenAt,omitempty"`
	AgeSeconds *int64     `json:"ageSeconds,omitempty"`
	Stale      bool       `json:"stale"`
	// Status: ok | stale | unknown
	Status string `json:"status"`
}

// ObserveHeartbeat 根据 last_seen 与 stale 阈值生成投影。
func ObserveHeartbeat(lastSeen time.Time, found bool, now time.Time, staleAfter time.Duration) WorkerHeartbeat {
	if staleAfter <= 0 {
		staleAfter = DefaultHeartbeatStaleAfter
	}
	now = now.UTC()
	if !found || lastSeen.IsZero() {
		return WorkerHeartbeat{Stale: true, Status: "unknown"}
	}
	lastSeen = lastSeen.UTC()
	age := int64(now.Sub(lastSeen).Seconds())
	if age < 0 {
		age = 0
	}
	stale := now.Sub(lastSeen) > staleAfter
	status := "ok"
	if stale {
		status = "stale"
	}
	return WorkerHeartbeat{
		LastSeenAt: &lastSeen,
		AgeSeconds: &age,
		Stale:      stale,
		Status:     status,
	}
}

// Publisher 周期性写心跳，直到 ctx 取消。
type Publisher struct {
	Store    HeartbeatStore
	Interval time.Duration
	Now      func() time.Time
}

// Run 阻塞直到 ctx 结束。interval 默认 10s。
func (p *Publisher) Run(ctx context.Context) {
	if p == nil || p.Store == nil {
		return
	}
	interval := p.Interval
	if interval <= 0 {
		interval = DefaultHeartbeatInterval
	}
	nowFn := p.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	// 启动立刻写一次，缩短「刚启动但 overview 仍 unknown」窗口。
	_ = p.Store.Touch(ctx, nowFn().UTC())

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = p.Store.Touch(ctx, nowFn().UTC())
		}
	}
}

package main

import (
	"context"
	"database/sql"
)

// ---- user_metric_counters：通用 (user_id, metric) 计数表 ----

// 已知指标名。引擎按事件路由到对应指标；未知指标拒收。
const (
	MetricTopicCount     = "topic_count"
	MetricCommentCount   = "comment_count"
	MetricRegisteredDays = "registered_days" // 评估时现算，不落计数
)

func validMetric(metric string) bool {
	switch metric {
	case MetricTopicCount, MetricCommentCount:
		return true
	}
	return false
}

// IncrementCounter 计数 +n。与授予写入同事务由引擎层保证；这里是单语句幂等 upsert。
func (s *gamificationStore) IncrementCounter(ctx context.Context, userID int64, metric string, delta int64) error {
	if userID <= 0 || !validMetric(metric) {
		return errStore("gamification.counter_metric_invalid", "counter metric is not supported")
	}
	if delta == 0 {
		return nil
	}
	if s.db != nil {
		_, err := s.db.ExecContext(ctx, `
INSERT INTO ` + s.schema + `.user_metric_counters (user_id, metric, value)
VALUES ($1,$2,$3)
ON CONFLICT (user_id, metric) DO UPDATE SET value = ` + s.schema + `.user_metric_counters.value + EXCLUDED.value`,
			userID, metric, delta)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	userKey := userBadgeKey(userID)
	if s.counters[userKey] == nil {
		s.counters[userKey] = map[string]int64{}
	}
	s.counters[userKey][metric] += delta
	return nil
}

// SetCounter 写入权威计数（回填/重建语义，覆盖而非累加）。
// 日常事件仍走 IncrementCounter；只有「权威重建」入口使用它。
func (s *gamificationStore) SetCounter(ctx context.Context, userID int64, metric string, value int64) error {
	if userID <= 0 || !validMetric(metric) {
		return errStore("gamification.counter_metric_invalid", "counter metric is not supported")
	}
	if value < 0 {
		return errStore("gamification.counter_negative", "counters can never be negative")
	}
	if s.db != nil {
		_, err := s.db.ExecContext(ctx, `
INSERT INTO ` + s.schema + `.user_metric_counters (user_id, metric, value)
VALUES ($1,$2,$3)
ON CONFLICT (user_id, metric) DO UPDATE SET value = EXCLUDED.value`, userID, metric, value)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.counters[userBadgeKey(userID)] == nil {
		s.counters[userBadgeKey(userID)] = map[string]int64{}
	}
	s.counters[userBadgeKey(userID)][metric] = value
	return nil
}

// GetCounter 读取当前计数（未记录返回 0）。
func (s *gamificationStore) GetCounter(ctx context.Context, userID int64, metric string) (int64, error) {
	if !validMetric(metric) && metric != MetricRegisteredDays {
		return 0, errStore("gamification.counter_metric_invalid", "counter metric is not supported")
	}
	if metric == MetricRegisteredDays {
		return 0, errStore("gamification.counter_computed", "registered_days is computed at evaluation time")
	}
	if s.db != nil {
		var value sql.NullInt64
		err := s.db.QueryRowContext(ctx,
			`SELECT value FROM ` + s.schema + `.user_metric_counters WHERE user_id = $1 AND metric = $2`,
			userID, metric).Scan(&value)
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return value.Int64, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.counters[userBadgeKey(userID)][metric], nil
}

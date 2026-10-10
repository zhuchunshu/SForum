package main

import (
	"context"
	"database/sql"
	"time"
)

// RegistrationSnapshot 是 user_registrations 一行：注册事件到达时的
// registered_at 快照，每日周年巡检的自包含数据源。
type RegistrationSnapshot struct {
	UserID       int64
	RegisteredAt time.Time
}

// SnapshotRegistration 记录注册时间快照（幂等：只写一次，已存在不覆盖——
// 首次观察即权威）。
func (s *gamificationStore) SnapshotRegistration(ctx context.Context, userID int64, registeredAt time.Time) error {
	if userID <= 0 {
		return errStore("gamification.registration_invalid", "registration snapshot requires a positive user")
	}
	if registeredAt.IsZero() {
		return errStore("gamification.registration_time_required", "registration snapshot requires a time")
	}
	if s.db != nil {
		_, err := s.db.ExecContext(ctx, `
INSERT INTO ` + s.schema + `.user_registrations (user_id, registered_at)
VALUES ($1,$2)
ON CONFLICT (user_id) DO NOTHING`, userID, registeredAt)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.registrations == nil {
		s.registrations = map[int64]time.Time{}
	}
	if _, exists := s.registrations[userID]; !exists {
		s.registrations[userID] = registeredAt
	}
	return nil
}

// GetRegistration 读取单个快照。
func (s *gamificationStore) GetRegistration(ctx context.Context, userID int64) (time.Time, error) {
	if s.db != nil {
		var at time.Time
		err := s.db.QueryRowContext(ctx,
			`SELECT registered_at FROM ` + s.schema + `.user_registrations WHERE user_id = $1`, userID).Scan(&at)
		return at, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	at, ok := s.registrations[userID]
	if !ok {
		return time.Time{}, sql.ErrNoRows
	}
	return at, nil
}

// ListRegistrationSnapshots 按 user_id 升序分页（巡检游标）。
func (s *gamificationStore) ListRegistrationSnapshots(ctx context.Context, afterUserID int64, limit int) ([]RegistrationSnapshot, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	if s.db != nil {
		rows, err := s.db.QueryContext(ctx, `
SELECT user_id, registered_at FROM ` + s.schema + `.user_registrations
WHERE user_id > $1 ORDER BY user_id LIMIT $2`, afterUserID, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var result []RegistrationSnapshot
		for rows.Next() {
			var snap RegistrationSnapshot
			if err := rows.Scan(&snap.UserID, &snap.RegisteredAt); err != nil {
				return nil, err
			}
			result = append(result, snap)
		}
		return result, rows.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []RegistrationSnapshot
	// 内存态按 userID 升序（map 遍历无序，小规模插入排序足够）。
	ids := make([]int64, 0, len(s.registrations))
	for id := range s.registrations {
		if id > afterUserID {
			ids = append(ids, id)
		}
	}
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && ids[j] < ids[j-1]; j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
	if len(ids) > limit {
		ids = ids[:limit]
	}
	for _, id := range ids {
		result = append(result, RegistrationSnapshot{UserID: id, RegisteredAt: s.registrations[id]})
	}
	return result, nil
}

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func jsonRaw(b []byte) json.RawMessage { return append(json.RawMessage(nil), b...) }

func jsonUnmarshalStrict(data []byte, target any) error {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	return dec.Decode(target)
}

// ---- user_points_ledger / user_badges：同一事务的授予写入 ----

// GrantResult 是一次授予的结果。Already=true 表示幂等重放（未重复写）。
type GrantResult struct {
	Granted   bool
	Already   bool
	UserBadge UserBadge
}

// AppendLedger 幂等追加成长值。event_id+rule_id 唯一冲突时返回 Already。
func (s *gamificationStore) AppendLedger(ctx context.Context, entry LedgerEntry) (bool, error) {
	if entry.EventID == "" || entry.RuleID == "" || entry.UserID <= 0 {
		return false, errStore("gamification.ledger_key_required", "event and rule ids with a positive user are required")
	}
	if entry.Amount < 0 {
		return false, errStore("gamification.ledger_negative", "growth value can never be negative")
	}
	if s.db != nil {
		tag, err := s.db.ExecContext(ctx, `
INSERT INTO ` + s.schema + `.user_points_ledger (event_id, rule_id, type, amount, user_id, entity_ref)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (event_id, rule_id) DO NOTHING`,
			entry.EventID, entry.RuleID, entry.Type, entry.Amount, entry.UserID, entry.EntityRef)
		if err != nil {
			return false, err
		}
		n, _ := tag.RowsAffected()
		return n > 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.ledger {
		if existing.EventID == entry.EventID && existing.RuleID == entry.RuleID {
			return false, nil
		}
	}
	s.ledger = append(s.ledger, entry)
	return true, nil
}

// SumPoints 聚合账本得当前成长值（无余额列，可重放修复）。
func (s *gamificationStore) SumPoints(ctx context.Context, userID int64) (int64, error) {
	if s.db != nil {
		var total sql.NullInt64
		err := s.db.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(amount),0) FROM ` + s.schema + `.user_points_ledger WHERE user_id = $1`, userID).
			Scan(&total)
		return total.Int64, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var total int64
	for _, e := range s.ledger {
		if e.UserID == userID {
			total += e.Amount
		}
	}
	return total, nil
}

// ResolveLevel 返回 min_points <= total 的最高等级。无规则表时返回零值。
func (s *gamificationStore) ResolveLevel(ctx context.Context, userID int64) (LevelRule, int64, error) {
	levels, err := s.ListLevels(ctx)
	if err != nil {
		return LevelRule{}, 0, err
	}
	total, err := s.SumPoints(ctx, userID)
	if err != nil {
		return LevelRule{}, 0, err
	}
	var current LevelRule
	for _, rule := range levels {
		if rule.MinPoints <= total {
			current = rule
		}
	}
	return current, total, nil
}

// GrantBadge 写 user_badges 一行（幂等：已授予返回 Already）。
// badge 存在且未停用由调用方（引擎/管理端）先校验。
func (s *gamificationStore) GrantBadge(ctx context.Context, grant UserBadge) (GrantResult, error) {
	if grant.UserID <= 0 || grant.BadgeSlug == "" {
		return GrantResult{}, errStore("gamification.grant_invalid", "user and badge slug are required")
	}
	if grant.Source != "auto" && grant.Source != "manual" {
		return GrantResult{}, errStore("gamification.grant_source_invalid", "source must be auto or manual")
	}
	if grant.Source == "manual" && grant.ActorID <= 0 {
		return GrantResult{}, errStore("gamification.grant_actor_required", "manual grants require an admin actor")
	}
	grant.GrantedAt = time.Now().UTC()
	if s.db != nil {
		tag, err := s.db.ExecContext(ctx, `
INSERT INTO ` + s.schema + `.user_badges (user_id, badge_slug, source, actor_id, granted_at)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (user_id, badge_slug) DO NOTHING`,
			grant.UserID, grant.BadgeSlug, grant.Source, grant.ActorID, grant.GrantedAt)
		if err != nil {
			return GrantResult{}, err
		}
		n, _ := tag.RowsAffected()
		return GrantResult{Granted: n > 0, Already: n == 0, UserBadge: grant}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	userKey := userBadgeKey(grant.UserID)
	grants, ok := s.userBadge[userKey]
	if !ok {
		grants = map[string]UserBadge{}
		s.userBadge[userKey] = grants
	}
	if _, exists := grants[grant.BadgeSlug]; exists {
		return GrantResult{Already: true, UserBadge: grants[grant.BadgeSlug]}, nil
	}
	grants[grant.BadgeSlug] = grant
	return GrantResult{Granted: true, UserBadge: grant}, nil
}

// RevokeBadge 删除 user_badges 一行（仅 manual 撤销路径使用；auto 由幂等键防复发）。
func (s *gamificationStore) RevokeBadge(ctx context.Context, userID int64, badgeSlug string) error {
	if s.db != nil {
		tag, err := s.db.ExecContext(ctx,
			`DELETE FROM ` + s.schema + `.user_badges WHERE user_id = $1 AND badge_slug = $2`, userID, badgeSlug)
		if err != nil {
			return err
		}
		if n, _ := tag.RowsAffected(); n == 0 {
			return sql.ErrNoRows
		}
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	grants := s.userBadge[userBadgeKey(userID)]
	if _, ok := grants[badgeSlug]; !ok {
		return sql.ErrNoRows
	}
	delete(grants, badgeSlug)
	return nil
}

// ListUserBadges 返回某用户全部授予（含已停用徽章的历史授予）。
func (s *gamificationStore) ListUserBadges(ctx context.Context, userID int64) ([]UserBadge, error) {
	if s.db != nil {
		rows, err := s.db.QueryContext(ctx, `
SELECT user_id, badge_slug, source, actor_id, granted_at
FROM ` + s.schema + `.user_badges WHERE user_id = $1 ORDER BY granted_at`, userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var result []UserBadge
		for rows.Next() {
			var g UserBadge
			if err := rows.Scan(&g.UserID, &g.BadgeSlug, &g.Source, &g.ActorID, &g.GrantedAt); err != nil {
				return nil, err
			}
			result = append(result, g)
		}
		return result, rows.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []UserBadge
	for _, g := range s.userBadge[userBadgeKey(userID)] {
		result = append(result, g)
	}
	return result, nil
}

func userBadgeKey(userID int64) string { return strconv.FormatInt(userID, 10) }

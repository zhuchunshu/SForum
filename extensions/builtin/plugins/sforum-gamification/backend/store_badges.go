package main

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// ---- badge_definitions ----

func (s *gamificationStore) UpsertBadge(ctx context.Context, def BadgeDefinition) error {
	if err := validateBadgeDefinition(def); err != nil {
		return err
	}
	if s.db != nil {
		_, err := s.db.ExecContext(ctx, `
INSERT INTO ` + s.schema + `.badge_definitions
  (slug, name, description, rarity, rule, icon, grants_points, disabled_at, sort)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (slug) DO UPDATE SET
  name = EXCLUDED.name, description = EXCLUDED.description, rarity = EXCLUDED.rarity,
  rule = EXCLUDED.rule, icon = EXCLUDED.icon, grants_points = EXCLUDED.grants_points,
  disabled_at = EXCLUDED.disabled_at, sort = EXCLUDED.sort`,
			def.Slug, def.Name, def.Description, def.Rarity, []byte(def.Rule), def.Icon,
			def.Points, def.DisabledAt, def.Sort)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.badges[def.Slug] = def
	return nil
}

// ListBadges 返回启用中的徽章定义（disabled 视为停用，不删除；历史授予保留）。
func (s *gamificationStore) ListBadges(ctx context.Context, includeDisabled bool) ([]BadgeDefinition, error) {
	if s.db != nil {
		query := `SELECT slug, name, description, rarity, rule, icon, grants_points, disabled_at, sort
FROM ` + s.schema + `.badge_definitions`
		if !includeDisabled {
			query += ` WHERE disabled_at IS NULL`
		}
		query += ` ORDER BY sort, slug`
		rows, err := s.db.QueryContext(ctx, query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var result []BadgeDefinition
		for rows.Next() {
			def, err := scanBadge(rows)
			if err != nil {
				return nil, err
			}
			result = append(result, def)
		}
		return result, rows.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []BadgeDefinition
	for _, def := range s.badges {
		if !includeDisabled && def.DisabledAt != nil {
			continue
		}
		result = append(result, def)
	}
	sortBadges(result)
	return result, nil
}

func (s *gamificationStore) GetBadge(ctx context.Context, slug string) (BadgeDefinition, error) {
	if s.db != nil {
		row := s.db.QueryRowContext(ctx, `
SELECT slug, name, description, rarity, rule, icon, grants_points, disabled_at, sort
FROM ` + s.schema + `.badge_definitions WHERE slug = $1`, slug)
		return scanBadge(row)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	def, ok := s.badges[slug]
	if !ok {
		return BadgeDefinition{}, sql.ErrNoRows
	}
	return def, nil
}

// DisableBadge 置 disabled_at 而非删除；唯一名不可复用直到显式恢复。
func (s *gamificationStore) DisableBadge(ctx context.Context, slug string) error {
	if s.db != nil {
		tag, err := s.db.ExecContext(ctx, `
UPDATE ` + s.schema + `.badge_definitions SET disabled_at = now() WHERE slug = $1 AND disabled_at IS NULL`, slug)
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
	def, ok := s.badges[slug]
	if !ok || def.DisabledAt != nil {
		return sql.ErrNoRows
	}
	now := time.Now().UTC()
	def.DisabledAt = &now
	s.badges[slug] = def
	return nil
}

// ---- level_rules ----

func (s *gamificationStore) UpsertLevel(ctx context.Context, rule LevelRule) error {
	if rule.MinPoints < 0 || rule.Title == "" {
		return errStore("gamification.level_invalid", "level min_points must be >= 0 with a title")
	}
	if s.db != nil {
		_, err := s.db.ExecContext(ctx, `
INSERT INTO ` + s.schema + `.level_rules (min_points, title, icon) VALUES ($1,$2,$3)
ON CONFLICT (min_points) DO UPDATE SET title = EXCLUDED.title, icon = EXCLUDED.icon`,
			rule.MinPoints, rule.Title, rule.Icon)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.levels {
		if existing.MinPoints == rule.MinPoints {
			s.levels[i] = rule
			return nil
		}
	}
	s.levels = append(s.levels, rule)
	sortLevels(s.levels)
	return nil
}

func (s *gamificationStore) ListLevels(ctx context.Context) ([]LevelRule, error) {
	if s.db != nil {
		rows, err := s.db.QueryContext(ctx, `SELECT min_points, title, icon FROM ` + s.schema + `.level_rules ORDER BY min_points`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var result []LevelRule
		for rows.Next() {
			var rule LevelRule
			if err := rows.Scan(&rule.MinPoints, &rule.Title, &rule.Icon); err != nil {
				return nil, err
			}
			result = append(result, rule)
		}
		return result, rows.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]LevelRule(nil), s.levels...)
	sortLevels(out)
	return out, nil
}

type rowScanner interface{ Scan(dest ...any) error }

func scanBadge(row rowScanner) (BadgeDefinition, error) {
	var def BadgeDefinition
	var rule []byte
	err := row.Scan(&def.Slug, &def.Name, &def.Description, &def.Rarity, &rule,
		&def.Icon, &def.Points, &def.DisabledAt, &def.Sort)
	if err != nil {
		return BadgeDefinition{}, err
	}
	def.Rule = jsonRaw(rule)
	return def, nil
}

func validateBadgeDefinition(def BadgeDefinition) error {
	if !slugPattern.MatchString(def.Slug) {
		return errStore("gamification.badge_slug_invalid", "badge slug must be kebab-case")
	}
	if strings.TrimSpace(def.Name) == "" {
		return errStore("gamification.badge_name_required", "badge name is required")
	}
	switch def.Rarity {
	case "common", "rare", "epic", "legendary":
	default:
		return errStore("gamification.badge_rarity_invalid", "rarity must be one of common/rare/epic/legendary")
	}
	if len(def.Rule) == 0 {
		return errStore("gamification.badge_rule_required", "rule JSON is required")
	}
	var rule struct {
		Kind    string `json:"kind"`
		Metric  string `json:"metric"`
		Threshold *int64 `json:"threshold"`
	}
	if err := jsonUnmarshalStrict(def.Rule, &rule); err != nil {
		return errStore("gamification.badge_rule_invalid_json", "rule must be an object")
	}
	switch rule.Kind {
	case "threshold":
		switch rule.Metric {
		case "topic_count", "comment_count", "registered_days":
		default:
			return errStore("gamification.badge_rule_metric_unknown", "rule metric is not supported")
		}
		if rule.Threshold == nil || *rule.Threshold < 1 {
			return errStore("gamification.badge_rule_threshold_invalid", "threshold must be >= 1")
		}
	case "manual":
		// manual 徽章无 metric/threshold。
	default:
		return errStore("gamification.badge_rule_kind_unknown", "rule kind must be threshold or manual")
	}
	return nil
}

func sortBadges(list []BadgeDefinition) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && badgeLess(list[j], list[j-1]); j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}

func badgeLess(a, b BadgeDefinition) bool {
	if a.Sort != b.Sort {
		return a.Sort < b.Sort
	}
	return a.Slug < b.Slug
}

func sortLevels(list []LevelRule) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j].MinPoints < list[j-1].MinPoints; j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}




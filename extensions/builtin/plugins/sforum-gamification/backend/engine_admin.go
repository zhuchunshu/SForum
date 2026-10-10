package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ---- 默认数据：徽章 + 等级（推荐默认，一键恢复） ----

// defaultBadges 是 M0 确认的一期默认徽章集（slug, 规则, 成长值）。
func defaultBadges() []BadgeDefinition {
	rule := func(metric string, threshold int64) json.RawMessage {
		raw, _ := json.Marshal(map[string]any{
			"kind": "threshold", "metric": metric, "threshold": threshold,
		})
		return raw
	}
	manual := json.RawMessage(`{"kind":"manual"}`)
	return []BadgeDefinition{
		{Slug: "first-topic", Name: "初露锋芒", Description: "发布了第一个主题", Rarity: "common",
			Rule: rule("topic_count", 1), Icon: "i-tabler-seedling", Points: 5, Sort: 10},
		{Slug: "first-comment", Name: "小有所言", Description: "发表了第一条评论", Rarity: "common",
			Rule: rule("comment_count", 1), Icon: "i-tabler-message", Points: 3, Sort: 20},
		{Slug: "ten-topics", Name: "笔耕不辍", Description: "发布了 10 个主题", Rarity: "common",
			Rule: rule("topic_count", 10), Icon: "i-tabler-pencil", Points: 20, Sort: 30},
		{Slug: "fifty-topics", Name: "社区之声", Description: "发布了 50 个主题", Rarity: "rare",
			Rule: rule("topic_count", 50), Icon: "i-tabler-speakerphone", Points: 50, Sort: 40},
		{Slug: "two-hundred-topics", Name: "高产作家", Description: "发布了 200 个主题", Rarity: "epic",
			Rule: rule("topic_count", 200), Icon: "i-tabler-book", Points: 150, Sort: 50},
		{Slug: "hundred-comments", Name: "健谈之人", Description: "发表了 100 条评论", Rarity: "rare",
			Rule: rule("comment_count", 100), Icon: "i-tabler-messages", Points: 40, Sort: 60},
		{Slug: "veteran", Name: "元老", Description: "注册满一年", Rarity: "epic",
			Rule: rule("registered_days", 365), Icon: "i-tabler-cake", Points: 30, Sort: 70},
		{Slug: "outstanding", Name: "优秀贡献者", Description: "由管理员手动授予", Rarity: "legendary",
			Rule: manual, Icon: "i-tabler-award", Points: 0, Sort: 80},
	}
}

func defaultLevels() []LevelRule {
	return []LevelRule{
		{MinPoints: 0, Title: "新芽", Icon: "i-tabler-seedling"},
		{MinPoints: 20, Title: "幼苗", Icon: "i-tabler-plant"},
		{MinPoints: 60, Title: "小树", Icon: "i-tabler-tree"},
		{MinPoints: 150, Title: "大树", Icon: "i-tabler-trees"},
		{MinPoints: 400, Title: "守望者", Icon: "i-tabler-shield-check"},
	}
}

// SeedResult 是一键恢复默认的写入计数。
type SeedResult struct {
	Badges int
	Levels int
}

// SeedDefaults 写入推荐默认徽章与等级（upsert：同名覆盖规则，历史授予不动）。
func (e *Engine) SeedDefaults(ctx context.Context) (SeedResult, error) {
	var result SeedResult
	for _, badge := range defaultBadges() {
		if err := e.Store.UpsertBadge(ctx, badge); err != nil {
			return result, err
		}
		result.Badges++
	}
	for _, level := range defaultLevels() {
		if err := e.Store.UpsertLevel(ctx, level); err != nil {
			return result, err
		}
		result.Levels++
	}
	return result, nil
}

// ---- 历史回填（管理员手动触发） ----

// Backfill 把传入的历史计数写入计量表并补发成长值。payload 示例：
// {"users":[{"userId":42,"topicCount":17,"commentCount":3}]}
// 幂等：回填事件的 rule_id 固定，重跑不会重复入账。
func (e *Engine) Backfill(ctx context.Context, payload map[string]any) (int, int64, error) {
	users := 0
	var credited int64
	raw, ok := payload["users"].([]any)
	if !ok {
		return 0, 0, errStore("gamification.backfill_payload_invalid", "users array is required")
	}
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		userID := int64FromAny(entry["userId"])
		if userID <= 0 {
			continue
		}
		topics := int64FromAny(entry["topicCount"])
		comments := int64FromAny(entry["commentCount"])
		if topics > 0 {
			// 回填是权威重建：写入实际计数而非累加，重跑安全。
			if err := e.Store.SetCounter(ctx, userID, MetricTopicCount, topics); err != nil {
				return users, credited, err
			}
			wrote, err := e.Store.AppendLedger(ctx, LedgerEntry{
				EventID: fmt.Sprintf("backfill:%s:%d", MetricTopicCount, userID),
				RuleID:  "growth.topic", Type: "growth",
				Amount:  topics * e.Values.TopicCreated, UserID: userID,
			})
			if err != nil {
				return users, credited, err
			}
			if wrote {
				credited += topics * e.Values.TopicCreated
			}
		}
		if comments > 0 {
			if err := e.Store.SetCounter(ctx, userID, MetricCommentCount, comments); err != nil {
				return users, credited, err
			}
			wrote, err := e.Store.AppendLedger(ctx, LedgerEntry{
				EventID: fmt.Sprintf("backfill:%s:%d", MetricCommentCount, userID),
				RuleID:  "growth.comment", Type: "growth",
				Amount:  comments * e.Values.CommentCreated, UserID: userID,
			})
			if err != nil {
				return users, credited, err
			}
			if wrote {
				credited += comments * e.Values.CommentCreated
			}
		}
		users++
		// 回填后按当前计数重跑规则匹配（补发应得的徽章）。
		if _, err := e.EvaluateEvent(ctx, EngineEvent{
			EventID: fmt.Sprintf("backfill-eval:%d", userID), Name: "gamification.backfill", UserID: userID,
		}); err != nil {
			return users, credited, err
		}
	}
	return users, credited, nil
}

// ---- 公开资料徽章墙（M5 前端接 forum.profile.tabs，这里先给数据契约） ----

// ProfileWall 汇总某用户的成长值、等级与徽章墙。
func (e *Engine) ProfileWall(ctx context.Context, userID int64) (map[string]any, error) {
	if userID <= 0 {
		return nil, errStore("gamification.user_required", "a positive user id is required")
	}
	grants, err := e.Store.ListUserBadges(ctx, userID)
	if err != nil {
		return nil, err
	}
	badges, err := e.Store.ListBadges(ctx, false)
	if err != nil {
		return nil, err
	}
	bySlug := map[string]BadgeDefinition{}
	for _, badge := range badges {
		bySlug[badge.Slug] = badge
	}
	wall := make([]map[string]any, 0, len(grants))
	for _, grant := range grants {
		badge, ok := bySlug[grant.BadgeSlug]
		if !ok {
			continue // 已停用徽章的历史授予不公开展示
		}
		wall = append(wall, map[string]any{
			"slug": badge.Slug, "name": badge.Name, "description": badge.Description,
			"rarity": badge.Rarity, "icon": badge.Icon,
			"grantedAt": grant.GrantedAt.UTC().Format(time.RFC3339),
		})
	}
	level, total, err := e.Store.ResolveLevel(ctx, userID)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(wall)
	if err != nil {
		return nil, err
	}
	levelJSON, err := json.Marshal(map[string]any{
		"title": level.Title, "icon": level.Icon, "minPoints": level.MinPoints,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"userId": userID, "points": total,
		// 结构化数组以 JSON 字符串回传（TypedDocument 值域为原始类型）。
		"level":  string(levelJSON),
		"badges": string(encoded),
	}, nil
}

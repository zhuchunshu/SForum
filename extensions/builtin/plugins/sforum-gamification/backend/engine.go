package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ---- 授予引擎：五步流水线 ----
// 1. 接收事件（River 异步投递到 InvokeHook）
// 2. 读指标计数
// 3. 匹配启用规则（badge_definitions.rule JSON）
// 4. 幂等检查（event_id + rule_id / user+badge 唯一）
// 5. 写 user_badges + user_points_ledger（通知由 Host 侧在 M5 接）

// Engine 输入事件。Host 投递的 payload 被解析为这三个字段。
type EngineEvent struct {
	EventID string // 幂等锚点（correlation id / delivery id）
	Name    string // user.registered | topic.created | comment.created
	UserID  int64
	// registered_days 评估需要：用户注册时间（user.registered 携带或查询补齐）。
	RegisteredAt time.Time
}

// GrowthValues 是插件设置里的成长值配置（admin 可编辑，带推荐默认）。
type GrowthValues struct {
	TopicCreated  int64
	CommentCreated int64
	Registration  int64
}

func DefaultGrowthValues() GrowthValues {
	return GrowthValues{TopicCreated: 5, CommentCreated: 3, Registration: 10}
}

// BadgeRule 是 badge_definitions.rule 的解析形态（v1 语法）。
type BadgeRule struct {
	Kind      string `json:"kind"`
	Metric    string `json:"metric"`
	Threshold int64  `json:"threshold"`
}

// GrantOutcome 是一次引擎评估命中的单个徽章。
type GrantOutcome struct {
	Badge BadgeDefinition
	Newly bool // false = 已持有（幂等重放或早达阈值）
}

// EvaluateEvent 对一个事件跑完整流水线。返回本次新授予的徽章。
// 引擎错误（存储故障）原样上抛供 River 重试；规则不匹配是正常空结果。
func (e *Engine) EvaluateEvent(ctx context.Context, event EngineEvent) ([]GrantOutcome, error) {
	if err := e.validateEvent(event); err != nil {
		return nil, err
	}
	// 第 2 步：计数器增量与账本同事务性由存储层幂等保证——
	// River unique-by-args 会在投递层去重一次，(event_id, metric) 计数
	// 只在账本写入成功时发生（见 applyGrowth）。
	outcomes := []GrantOutcome{}
	for _, badge := range e.enabledBadges(ctx) {
		match, err := e.matchesThreshold(ctx, badge, event)
		if err != nil {
			return nil, err
		}
		if !match {
			continue
		}
		newly, err := e.grantBadgeForEvent(ctx, badge, event)
		if err != nil {
			return nil, err
		}
		outcomes = append(outcomes, GrantOutcome{Badge: badge, Newly: newly})
	}
	return outcomes, nil
}

// EvaluateDaily 对一个用户评估 registered_days（周期巡检入口）。
// registeredAt 由调用方（HostQuery safe_user 或注册事件快照）提供。
func (e *Engine) EvaluateDaily(ctx context.Context, userID int64, registeredAt time.Time) ([]GrantOutcome, error) {
	if userID <= 0 || registeredAt.IsZero() {
		return nil, errStore("gamification.daily_invalid", "daily evaluation needs a user and registration time")
	}
	days := int64(time.Since(registeredAt).Hours() / 24)
	outcomes := []GrantOutcome{}
	for _, badge := range e.enabledBadges(ctx) {
		rule, ok := parseBadgeRule(badge.Rule)
		if !ok || rule.Kind != "threshold" || rule.Metric != MetricRegisteredDays {
			continue
		}
		if days < rule.Threshold {
			continue
		}
		// 周年事件用稳定幂等键：metric + 周年日（同一年只授一次）。
		eventID := fmt.Sprintf("daily:%d:%d:%d", userID, rule.Threshold, registeredAt.AddDate(0, 0, int(rule.Threshold)).Year())
		newly, err := e.grantBadgeForEvent(ctx, badge, EngineEvent{
			EventID: eventID, Name: "gamification.daily", UserID: userID, RegisteredAt: registeredAt,
		})
		if err != nil {
			return nil, err
		}
		outcomes = append(outcomes, GrantOutcome{Badge: badge, Newly: newly})
	}
	return outcomes, nil
}

// CelebrationEmitter 在徽章新授予时调用；nil 表示不发送。
// 参数是用户、徽章与当前等级称号。
type CelebrationEmitter func(ctx context.Context, userID int64, badge BadgeDefinition, levelTitle string)

// Engine 持存储与配置。授权/通知等 Host 侧能力在 M3+ 注入。
type Engine struct {
	Store      *gamificationStore
	Values     GrowthValues
	Now        func() time.Time
	Celebrate  CelebrationEmitter
}

func NewEngine(store *gamificationStore) *Engine {
	return &Engine{Store: store, Values: DefaultGrowthValues(), Now: time.Now}
}

func (e *Engine) validateEvent(event EngineEvent) error {
	switch event.Name {
	case "user.registered", "topic.created", "comment.created":
	// 回填评估是内部合成事件：跳过具体指标事件名限制。
	case "gamification.backfill":
	default:
		return errStore("gamification.event_unknown", "engine does not subscribe to "+event.Name)
	}
	if event.UserID <= 0 {
		return errStore("gamification.event_user_required", "event requires a positive user id")
	}
	if event.EventID == "" {
		return errStore("gamification.event_id_required", "event id is required for idempotency")
	}
	return nil
}

func (e *Engine) enabledBadges(ctx context.Context) []BadgeDefinition {
	badges, err := e.Store.ListBadges(ctx, false)
	if err != nil {
		return nil // 存储故障：跳过本轮匹配，River 会重试
	}
	return badges
}

// matchesThreshold 判断规则当前是否达标。historically-invalid 规则跳过不崩。
func (e *Engine) matchesThreshold(ctx context.Context, badge BadgeDefinition, event EngineEvent) (bool, error) {
	rule, ok := parseBadgeRule(badge.Rule)
	if !ok || rule.Kind != "threshold" {
		return false, nil
	}
	var value int64
	switch rule.Metric {
	case MetricTopicCount, MetricCommentCount:
		// 日常事件只在该指标的事件上评估；回填/合成评估不区分事件名。
		if event.Name != metricEvent(rule.Metric) && !evaluatesAllMetrics(event.Name) {
			return false, nil
		}
		v, err := e.Store.GetCounter(ctx, event.UserID, rule.Metric)
		if err != nil {
			return false, err
		}
		value = v
	case MetricRegisteredDays:
		if event.Name != "user.registered" && event.Name != "gamification.daily" {
			return false, nil
		}
		if event.RegisteredAt.IsZero() {
			return false, nil
		}
		value = int64(time.Since(event.RegisteredAt).Hours() / 24)
	default:
		return false, nil // 未知指标：跳过（老规则兼容）
	}
	return value >= rule.Threshold, nil
}

// grantBadgeForEvent 完成一次授予写入（第 4、5 步）。
// 返回 newly=false 表示已持有（幂等重放）。
func (e *Engine) grantBadgeForEvent(ctx context.Context, badge BadgeDefinition, event EngineEvent) (bool, error) {
	result, err := e.Store.GrantBadge(ctx, UserBadge{
		UserID: event.UserID, BadgeSlug: badge.Slug, Source: "auto",
	})
	if err != nil || !result.Granted {
		return false, err // 已持有：幂等，不入账
	}
	// 徽章成长值：同一事件同一徽章只入账一次。
	if badge.Points > 0 {
		if _, err := e.Store.AppendLedger(ctx, LedgerEntry{
			EventID: event.EventID, RuleID: "badge.grant." + badge.Slug,
			Type: "badge_grant", Amount: badge.Points, UserID: event.UserID,
		}); err != nil {
			return false, err
		}
	}
	e.celebrate(ctx, event.UserID, badge)
	return true, nil
}

// celebrate 发送庆祝通知（失败不影响授予结果）。
func (e *Engine) celebrate(ctx context.Context, userID int64, badge BadgeDefinition) {
	if e.Celebrate == nil {
		return
	}
	level, _, err := e.Store.ResolveLevel(ctx, userID)
	if err != nil {
		level = LevelRule{}
	}
	e.Celebrate(ctx, userID, badge, level.Title)
}

// applyGrowth 是事件带来的成长值入账（发帖/评论/注册）。
// 计数增量与账本同批：先入账成功才计数，River 重放时账本幂等挡住双计。
func (e *Engine) applyGrowth(ctx context.Context, event EngineEvent) error {
	var metric, ruleID string
	amount := int64(0)
	switch event.Name {
	case "topic.created":
		metric, ruleID, amount = MetricTopicCount, "growth.topic", e.Values.TopicCreated
	case "comment.created":
		metric, ruleID, amount = MetricCommentCount, "growth.comment", e.Values.CommentCreated
	case "user.registered":
		ruleID, amount = "growth.registration", e.Values.Registration
	}
	if amount == 0 {
		return nil
	}
	wrote, err := e.Store.AppendLedger(ctx, LedgerEntry{
		EventID: event.EventID, RuleID: ruleID, Type: "growth", Amount: amount, UserID: event.UserID,
	})
	if err != nil || !wrote {
		return err // 已入账（重放）：不再计数
	}
	if metric != "" {
		if err := e.Store.IncrementCounter(ctx, event.UserID, metric, 1); err != nil {
			return err
		}
	}
	return nil
}

// HandleEvent 是 InvokeHook 的完整入口：成长值 + 规则匹配。
func (e *Engine) HandleEvent(ctx context.Context, event EngineEvent) ([]GrantOutcome, error) {
	if event.Name == "user.registered" {
		// 注册时间快照：取接收时刻（目录 payload 无时间字段，最多偏一天，
		// 周年语义不受影响）。写入幂等：快照只记一次。
		if err := e.Store.SnapshotRegistration(ctx, event.UserID, event.RegisteredAt); err != nil {
			return nil, err
		}
	}
	if err := e.applyGrowth(ctx, event); err != nil {
		return nil, err
	}
	return e.EvaluateEvent(ctx, event)
}

// RunDailyScan 周期巡检入口（manifest schedules → ExecuteJob 在 M4+ 接）。
// 逐批读注册快照，对快到期用户跑 EvaluateDaily。返回本轮新授予数。
func (e *Engine) RunDailyScan(ctx context.Context, batchSize int) (int, error) {
	if batchSize < 1 || batchSize > 500 {
		batchSize = 100
	}
	granted := 0
	var lastUserID int64
	for {
		snapshots, err := e.Store.ListRegistrationSnapshots(ctx, lastUserID, batchSize)
		if err != nil {
			return granted, err
		}
		if len(snapshots) == 0 {
			return granted, nil
		}
		for _, snap := range snapshots {
			lastUserID = snap.UserID
			outcomes, err := e.EvaluateDaily(ctx, snap.UserID, snap.RegisteredAt)
			if err != nil {
				// 单用户失败不中止整轮巡检（下轮重试）；记录计数继续。
				continue
			}
			for _, outcome := range outcomes {
				if outcome.Newly {
					granted++
				}
			}
		}
		if len(snapshots) < batchSize {
			return granted, nil
		}
	}
}

// ManualGrant 构造合成事件进入同一流水线。
func (e *Engine) ManualGrant(ctx context.Context, badgeSlug string, userID, actorID int64) (GrantResult, error) {
	badge, err := e.Store.GetBadge(ctx, badgeSlug)
	if err != nil {
		return GrantResult{}, err
	}
	if badge.DisabledAt != nil {
		return GrantResult{}, errStore("gamification.badge_disabled", "disabled badges cannot be granted")
	}
	if rule, ok := parseBadgeRule(badge.Rule); ok && rule.Kind != "manual" {
		return GrantResult{}, errStore("gamification.badge_not_manual", "only manual badges can be granted through the admin path")
	}
	result, err := e.Store.GrantBadge(ctx, UserBadge{
		UserID: userID, BadgeSlug: badgeSlug, Source: "manual", ActorID: actorID,
	})
	if err != nil || !result.Granted {
		return result, err // 已授予：幂等返回
	}
	if badge.Points > 0 {
		if _, err := e.Store.AppendLedger(ctx, LedgerEntry{
			EventID: result.UserBadge.GrantedAt.Format(time.RFC3339Nano) + ":" + badgeSlug,
			RuleID:  "badge.grant." + badgeSlug, Type: "badge_grant", Amount: badge.Points, UserID: userID,
		}); err != nil {
			return result, err
		}
	}
	e.celebrate(ctx, userID, badge)
	return result, nil
}

func parseBadgeRule(raw json.RawMessage) (BadgeRule, bool) {
	var rule BadgeRule
	if err := json.Unmarshal(raw, &rule); err != nil {
		return BadgeRule{}, false
	}
	if rule.Kind == "" {
		return BadgeRule{}, false
	}
	return rule, true
}

// evaluatesAllMetrics 判断事件是否为「全指标评估」的合成入口
// （回填、每日巡检）：这些入口不需要按事件名过滤计数指标。
func evaluatesAllMetrics(eventName string) bool {
	return eventName == "gamification.backfill" || eventName == "gamification.daily"
}

func metricEvent(metric string) string {
	switch metric {
	case MetricTopicCount:
		return "topic.created"
	case MetricCommentCount:
		return "comment.created"
	}
	return ""
}

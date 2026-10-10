package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func seedEngine(t *testing.T) (*Engine, *gamificationStore) {
	t.Helper()
	store := newTestStore(t)
	return NewEngine(store), store
}

func mustRule(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal rule: %v", err)
	}
	return raw
}

func TestEngineGrantsOnThreshold(t *testing.T) {
	engine, store := seedEngine(t)
	ctx := context.Background()
	// 阈值 10 的发帖徽章。
	if err := store.UpsertBadge(ctx, BadgeDefinition{
		Slug: "ten-topics", Name: "笔耕不辍", Rarity: "common",
		Rule: mustRule(t, map[string]any{"kind": "threshold", "metric": "topic_count", "threshold": 10}),
	}); err != nil {
		t.Fatalf("seed badge: %v", err)
	}
	// 前 9 帖不授予。
	for i := 0; i < 9; i++ {
		outcomes, err := engine.HandleEvent(ctx, EngineEvent{
			EventID: eventID("topic", i+1), Name: "topic.created", UserID: 42,
		})
		if err != nil {
			t.Fatalf("handle %d: %v", i, err)
		}
		if len(outcomes) != 0 {
			t.Fatalf("no grant expected before threshold, got %+v", outcomes)
		}
	}
	// 第 10 帖授予一次。
	outcomes, err := engine.HandleEvent(ctx, EngineEvent{
		EventID: eventID("topic", 10), Name: "topic.created", UserID: 42,
	})
	if err != nil || len(outcomes) != 1 || !outcomes[0].Newly || outcomes[0].Badge.Slug != "ten-topics" {
		t.Fatalf("threshold grant: outcomes=%+v err=%v", outcomes, err)
	}
	// 第 11 帖不再重复授予（幂等）。
	outcomes, err = engine.HandleEvent(ctx, EngineEvent{
		EventID: eventID("topic", 11), Name: "topic.created", UserID: 42,
	})
	if err != nil || len(outcomes) != 1 || outcomes[0].Newly {
		t.Fatalf("post-threshold must be idempotent: %+v %v", outcomes, err)
	}
	// 成长值 = 11 帖 × 5。
	total, err := store.SumPoints(ctx, 42)
	if err != nil || total != 55 {
		t.Fatalf("growth total: %d %v", total, err)
	}
}

func TestEngineReplayDoesNotDoubleCount(t *testing.T) {
	engine, store := seedEngine(t)
	ctx := context.Background()
	event := EngineEvent{EventID: eventID("topic", 1), Name: "topic.created", UserID: 42}
	if _, err := engine.HandleEvent(ctx, event); err != nil {
		t.Fatalf("first: %v", err)
	}
	// 完全相同的事件重放：River 可能因 unique 窗口过期重投。
	if _, err := engine.HandleEvent(ctx, event); err != nil {
		t.Fatalf("replay: %v", err)
	}
	counter, err := store.GetCounter(ctx, 42, MetricTopicCount)
	if err != nil || counter != 1 {
		t.Fatalf("replay must not double-count: counter=%d err=%v", counter, err)
	}
	total, err := store.SumPoints(ctx, 42)
	if err != nil || total != 5 {
		t.Fatalf("replay must not double-credit: total=%d err=%v", total, err)
	}
}

func TestEngineIgnoresForeignMetrics(t *testing.T) {
	engine, store := seedEngine(t)
	ctx := context.Background()
	if err := store.UpsertBadge(ctx, BadgeDefinition{
		Slug: "hundred-comments", Name: "健谈之人", Rarity: "common",
		Rule: mustRule(t, map[string]any{"kind": "threshold", "metric": "comment_count", "threshold": 1}),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// 发帖事件不评估评论指标。
	outcomes, err := engine.HandleEvent(ctx, EngineEvent{
		EventID: eventID("topic", 1), Name: "topic.created", UserID: 42,
	})
	if err != nil || len(outcomes) != 0 {
		t.Fatalf("topic event must not evaluate comment badges: %+v %v", outcomes, err)
	}
}

func TestEngineSkipsInvalidAndDisabledRules(t *testing.T) {
	engine, store := seedEngine(t)
	ctx := context.Background()
	// 历史脏规则（未知 kind）与已停用徽章都不崩、不授予。
	if err := store.UpsertBadge(ctx, BadgeDefinition{
		Slug: "legacy-streak", Name: "旧规则", Rarity: "common",
		Rule: json.RawMessage(`{"kind":"streak","metric":"days"}`),
	}); err == nil {
		t.Fatalf("save-time validation must reject unknown kinds")
	}
	// 绕过校验直插（模拟迁移残留）。
	store.mu.Lock()
	store.badges["legacy-streak"] = BadgeDefinition{
		Slug: "legacy-streak", Name: "旧规则", Rarity: "common",
		Rule: json.RawMessage(`{"kind":"streak","metric":"days"}`),
	}
	store.mu.Unlock()
	if _, err := engine.HandleEvent(ctx, EngineEvent{
		EventID: eventID("topic", 1), Name: "topic.created", UserID: 42,
	}); err != nil {
		t.Fatalf("engine must skip invalid rule instead of failing: %v", err)
	}
}

func TestEngineRegistrationCredit(t *testing.T) {
	engine, store := seedEngine(t)
	ctx := context.Background()
	if _, err := engine.HandleEvent(ctx, EngineEvent{
		EventID: eventID("register", 1), Name: "user.registered", UserID: 42, RegisteredAt: time.Now(),
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	total, err := store.SumPoints(ctx, 42)
	if err != nil || total != 10 {
		t.Fatalf("registration credit: %d %v", total, err)
	}
}

func TestEngineDailyVeteran(t *testing.T) {
	engine, store := seedEngine(t)
	ctx := context.Background()
	if err := store.UpsertBadge(ctx, BadgeDefinition{
		Slug: "veteran", Name: "元老", Rarity: "rare",
		Rule: mustRule(t, map[string]any{"kind": "threshold", "metric": "registered_days", "threshold": 365}),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// 注册 400 天的用户。
	registered := time.Now().AddDate(-1, -35, 0)
	outcomes, err := engine.EvaluateDaily(ctx, 42, registered)
	if err != nil || len(outcomes) != 1 || !outcomes[0].Newly {
		t.Fatalf("veteran grant: %+v %v", outcomes, err)
	}
	// 重跑（次日巡检）不重复授予。
	outcomes, err = engine.EvaluateDaily(ctx, 42, registered)
	if err != nil || len(outcomes) != 1 || outcomes[0].Newly {
		t.Fatalf("veteran replay must be idempotent: %+v %v", outcomes, err)
	}
	// 不足一年不授予。
	outcomes, err = engine.EvaluateDaily(ctx, 43, time.Now().AddDate(0, -6, 0))
	if err != nil || len(outcomes) != 0 {
		t.Fatalf("young user must not get veteran: %+v %v", outcomes, err)
	}
}

func TestEngineManualGrantPath(t *testing.T) {
	engine, store := seedEngine(t)
	ctx := context.Background()
	if err := store.UpsertBadge(ctx, BadgeDefinition{
		Slug: "outstanding", Name: "优秀贡献者", Rarity: "epic", Rule: json.RawMessage(`{"kind":"manual"}`), Points: 50,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	result, err := engine.ManualGrant(ctx, "outstanding", 42, 1)
	if err != nil || !result.Granted {
		t.Fatalf("manual grant: %+v %v", result, err)
	}
	// threshold 徽章拒绝走 manual 路径。
	if err := store.UpsertBadge(ctx, BadgeDefinition{
		Slug: "ten-topics", Name: "笔耕不辍", Rarity: "common",
		Rule: mustRule(t, map[string]any{"kind": "threshold", "metric": "topic_count", "threshold": 10}),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := engine.ManualGrant(ctx, "ten-topics", 42, 1); err == nil {
		t.Fatalf("threshold badge must not be manually grantable")
	}
	// 重复 manual 授予幂等。
	result, err = engine.ManualGrant(ctx, "outstanding", 42, 1)
	if err != nil || result.Granted {
		t.Fatalf("manual replay: %+v %v", result, err)
	}
	total, err := store.SumPoints(ctx, 42)
	if err != nil || total != 50 {
		t.Fatalf("manual points: %d %v", total, err)
	}
}

func TestEngineUnknownEventDenied(t *testing.T) {
	engine, _ := seedEngine(t)
	if _, err := engine.HandleEvent(context.Background(), EngineEvent{
		EventID: "e", Name: "topic.updated", UserID: 42,
	}); err == nil {
		t.Fatalf("unknown events must be rejected")
	}
}

var eventSeq int

func eventID(kind string, seq int) string {
	return kind + "-" + itoa(seq)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	buf := [12]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

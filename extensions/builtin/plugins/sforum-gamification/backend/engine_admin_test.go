package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSeedDefaultsRestoresRecommendedBadges(t *testing.T) {
	engine, store := seedEngine(t)
	ctx := context.Background()
	result, err := engine.SeedDefaults(ctx)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if result.Badges != 8 || result.Levels != 5 {
		t.Fatalf("seed counts: %+v", result)
	}
	badges, err := store.ListBadges(ctx, false)
	if err != nil || len(badges) != 8 {
		t.Fatalf("badges: %d err=%v", len(badges), err)
	}
	// 重复 seed 幂等：数量不变。
	if _, err := engine.SeedDefaults(ctx); err != nil {
		t.Fatalf("reseed: %v", err)
	}
	badges, err = store.ListBadges(ctx, false)
	if err != nil || len(badges) != 8 {
		t.Fatalf("reseed must be idempotent: %d err=%v", len(badges), err)
	}
	// 默认等级查表正确。
	level, _, err := store.ResolveLevel(ctx, 1)
	if err != nil || level.Title != "新芽" {
		t.Fatalf("default level: %+v err=%v", level, err)
	}
}

func TestBackfillGrantsHistoricalBadges(t *testing.T) {
	engine, store := seedEngine(t)
	ctx := context.Background()
	if _, err := engine.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// 存量用户：17 帖 4 评论应拿到 初露锋芒、小有所言、笔耕不辍，
	// 但拿不到 社区之声(50) 与 健谈之人(100)。
	users, points, err := engine.Backfill(ctx, map[string]any{
		"users": []any{map[string]any{"userId": 55, "topicCount": 17, "commentCount": 4}},
	})
	if err != nil || users != 1 {
		t.Fatalf("backfill: users=%d points=%d err=%v", users, points, err)
	}
	// 17*5 + 4*3 = 97（仅成长值，不含徽章成长值）。
	if points != 97 {
		t.Fatalf("backfill points: %d", points)
	}
	grants, err := store.ListUserBadges(ctx, 55)
	if err != nil || len(grants) != 3 {
		t.Fatalf("backfill grants (expect first-topic, first-comment, ten-topics): %+v err=%v", grants, err)
	}
	// 重跑回填不双计、不双授。
	_, points, err = engine.Backfill(ctx, map[string]any{
		"users": []any{map[string]any{"userId": 55, "topicCount": 17, "commentCount": 4}},
	})
	if err != nil {
		t.Fatalf("replay backfill: %v", err)
	}
	if points != 0 {
		t.Fatalf("backfill replay must credit nothing: %d", points)
	}
	// 97 成长值 + 徽章成长值（初露锋芒5 + 小有所言3 + 笔耕不辍20）= 125。
	total, err := store.SumPoints(ctx, 55)
	if err != nil || total != 125 {
		t.Fatalf("backfill replay total: %d err=%v", total, err)
	}
	if counter, _ := store.GetCounter(ctx, 55, MetricTopicCount); counter != 17 {
		t.Fatalf("backfill counter: %d", counter)
	}
}

func TestProfileWall(t *testing.T) {
	engine, store := seedEngine(t)
	ctx := context.Background()
	if _, err := engine.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := engine.HandleEvent(ctx, EngineEvent{
		EventID: eventID("topic", 1), Name: "topic.created", UserID: 77,
	}); err != nil {
		t.Fatalf("topic: %v", err)
	}
	wall, err := engine.ProfileWall(ctx, 77)
	if err != nil {
		t.Fatalf("wall: %v", err)
	}
	if wall["userId"] != int64(77) {
		t.Fatalf("wall user: %v", wall["userId"])
	}
	// 结构化数组以 JSON 字符串回传（TypedDocument 值域为原始类型）。
	badges := decodeJSONArray(t, wall["badges"])
	if len(badges) != 1 || badges[0]["slug"] != "first-topic" {
		t.Fatalf("wall badges: %#v", wall["badges"])
	}
	level := decodeJSONObject(t, wall["level"])
	if level["title"] != "新芽" {
		t.Fatalf("wall level: %#v", wall["level"])
	}
	// 停用徽章的历史授予不公开展示。
	if err := store.DisableBadge(ctx, "first-topic"); err != nil {
		t.Fatalf("disable: %v", err)
	}
	wall, err = engine.ProfileWall(ctx, 77)
	if err != nil {
		t.Fatalf("wall after disable: %v", err)
	}
	badges = decodeJSONArray(t, wall["badges"])
	if len(badges) != 0 {
		t.Fatalf("disabled badge must leave the public wall: %#v", badges)
	}
	// 但历史授予仍在（停用不删数据）。
	grants, _ := store.ListUserBadges(ctx, 77)
	if len(grants) != 1 {
		t.Fatalf("historical grants must survive: %#v", grants)
	}
}

func TestRunDailyScanIdempotent(t *testing.T) {
	engine, store := seedEngine(t)
	ctx := context.Background()
	if _, err := engine.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// 两个老用户（注册 400 天）+ 一个新用户。
	for _, id := range []int64{201, 202} {
		if err := store.SnapshotRegistration(ctx, id, time.Now().AddDate(-1, -35, 0)); err != nil {
			t.Fatalf("snapshot: %v", err)
		}
	}
	if err := store.SnapshotRegistration(ctx, 203, time.Now().AddDate(0, -1, 0)); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	granted, err := engine.RunDailyScan(ctx, 100)
	if err != nil || granted != 2 {
		t.Fatalf("daily scan: granted=%d err=%v", granted, err)
	}
	// 重跑不重复授予。
	granted, err = engine.RunDailyScan(ctx, 100)
	if err != nil || granted != 0 {
		t.Fatalf("daily scan replay: granted=%d err=%v", granted, err)
	}
}

func TestRegistrationSnapshotFirstWriteWins(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	first := time.Now().AddDate(-2, 0, 0)
	if err := store.SnapshotRegistration(ctx, 301, first); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	// 事件重投携带不同时间：首次观察即权威，不覆盖。
	if err := store.SnapshotRegistration(ctx, 301, time.Now()); err != nil {
		t.Fatalf("resnapshot: %v", err)
	}
	at, err := store.GetRegistration(ctx, 301)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !at.Equal(first) {
		t.Fatalf("snapshot must not be overwritten: %v vs %v", at, first)
	}
}

func decodeJSONArray(t *testing.T, value any) []map[string]any {
	t.Helper()
	text, ok := value.(string)
	if !ok {
		t.Fatalf("expected JSON string payload, got %T", value)
	}
	var result []map[string]any
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return result
}

func decodeJSONObject(t *testing.T, value any) map[string]any {
	t.Helper()
	text, ok := value.(string)
	if !ok {
		t.Fatalf("expected JSON string payload, got %T", value)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return result
}

func TestCelebrationFiresOnlyOnNewGrants(t *testing.T) {
	engine, store := seedEngine(t)
	ctx := context.Background()
	if _, err := engine.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}
	emitted := []string{}
	engine.Celebrate = func(_ context.Context, userID int64, badge BadgeDefinition, level string) {
		emitted = append(emitted, fmt.Sprintf("%d:%s:%s", userID, badge.Slug, level))
	}
	// 首帖：新授予 → 发射一次。
	if _, err := engine.HandleEvent(ctx, EngineEvent{
		EventID: eventID("topic", 1), Name: "topic.created", UserID: 88,
	}); err != nil {
		t.Fatalf("topic: %v", err)
	}
	if len(emitted) != 1 || !strings.HasPrefix(emitted[0], "88:first-topic:") {
		t.Fatalf("celebration: %#v", emitted)
	}
	// 重放：不重复发射。
	if _, err := engine.HandleEvent(ctx, EngineEvent{
		EventID: eventID("topic", 2), Name: "topic.created", UserID: 88,
	}); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(emitted) != 1 {
		t.Fatalf("celebration must not repeat: %#v", emitted)
	}
	// 通知发射器抛错/缺失都不影响已落库的授予。
	engine.Celebrate = nil
	grants, err := store.ListUserBadges(ctx, 88)
	if err != nil || len(grants) != 1 {
		t.Fatalf("grant must persist regardless of celebration: %+v err=%v", grants, err)
	}
}

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *gamificationStore {
	t.Helper()
	store, err := newGamificationStore()
	if err != nil {
		t.Fatalf("newGamificationStore: %v", err)
	}
	t.Cleanup(func() { _ = store.close() })
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	return store
}

func thresholdRule(t *testing.T, metric string, threshold int64) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"kind": "threshold", "metric": metric, "threshold": threshold})
	if err != nil {
		t.Fatalf("marshal rule: %v", err)
	}
	return raw
}

func TestBadgeDefinitionValidation(t *testing.T) {
	cases := []struct {
		name    string
		def     BadgeDefinition
		wantErr bool
	}{
		{"valid threshold", BadgeDefinition{Slug: "first-topic", Name: "初露锋芒", Rarity: "common", Rule: thresholdRule(t, "topic_count", 1)}, false},
		{"valid manual", BadgeDefinition{Slug: "outstanding", Name: "优秀贡献者", Rarity: "epic", Rule: json.RawMessage(`{"kind":"manual"}`)}, false},
		{"bad slug", BadgeDefinition{Slug: "Not A Slug", Name: "x", Rarity: "common", Rule: thresholdRule(t, "topic_count", 1)}, true},
		{"empty name", BadgeDefinition{Slug: "x", Name: " ", Rarity: "common", Rule: thresholdRule(t, "topic_count", 1)}, true},
		{"bad rarity", BadgeDefinition{Slug: "x", Name: "x", Rarity: "mythic", Rule: thresholdRule(t, "topic_count", 1)}, true},
		{"unknown kind", BadgeDefinition{Slug: "x", Name: "x", Rarity: "common", Rule: json.RawMessage(`{"kind":"streak"}`)}, true},
		{"unknown metric", BadgeDefinition{Slug: "x", Name: "x", Rarity: "common", Rule: thresholdRule(t, "likes", 1)}, true},
		{"zero threshold", BadgeDefinition{Slug: "x", Name: "x", Rarity: "common", Rule: thresholdRule(t, "topic_count", 0)}, true},
		{"missing rule", BadgeDefinition{Slug: "x", Name: "x", Rarity: "common"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBadgeDefinition(tc.def)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestGrantBadgeIdempotent(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	grant := UserBadge{UserID: 7, BadgeSlug: "first-topic", Source: "auto"}
	first, err := store.GrantBadge(ctx, grant)
	if err != nil || !first.Granted {
		t.Fatalf("first grant: granted=%v err=%v", first.Granted, err)
	}
	second, err := store.GrantBadge(ctx, grant)
	if err != nil {
		t.Fatalf("replay grant: %v", err)
	}
	if second.Granted || !second.Already {
		t.Fatalf("replay must be Already: granted=%v already=%v", second.Granted, second.Already)
	}
}

func TestManualGrantRequiresActor(t *testing.T) {
	store := newTestStore(t)
	_, err := store.GrantBadge(context.Background(), UserBadge{UserID: 7, BadgeSlug: "x", Source: "manual"})
	if err == nil || err.(*storeError).reason != "gamification.grant_actor_required" {
		t.Fatalf("manual grant without actor must fail with grant_actor_required, got %v", err)
	}
}

func TestLedgerIdempotentAndMonotonic(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	entry := LedgerEntry{EventID: "evt-1", RuleID: "growth.topic", Type: "growth", Amount: 5, UserID: 7}
	wrote, err := store.AppendLedger(ctx, entry)
	if err != nil || !wrote {
		t.Fatalf("first ledger append: wrote=%v err=%v", wrote, err)
	}
	wrote, err = store.AppendLedger(ctx, entry) // 幂等重放
	if err != nil || wrote {
		t.Fatalf("replay append must be no-op: wrote=%v err=%v", wrote, err)
	}
	if _, err := store.AppendLedger(ctx, LedgerEntry{EventID: "e", RuleID: "r", Amount: -1, UserID: 7}); err == nil {
		t.Fatalf("negative amount must be rejected")
	}
	total, err := store.SumPoints(ctx, 7)
	if err != nil || total != 5 {
		t.Fatalf("sum after replay: total=%d err=%v", total, err)
	}
}

func TestLevelResolution(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	for _, level := range []LevelRule{
		{MinPoints: 0, Title: "新芽"},
		{MinPoints: 20, Title: "幼苗"},
		{MinPoints: 60, Title: "小树"},
	} {
		if err := store.UpsertLevel(ctx, level); err != nil {
			t.Fatalf("upsert level: %v", err)
		}
	}
	_, _ = store.AppendLedger(ctx, LedgerEntry{EventID: "e1", RuleID: "growth.topic", Amount: 5, UserID: 9})
	_, _ = store.AppendLedger(ctx, LedgerEntry{EventID: "e2", RuleID: "badge.grant", Amount: 20, UserID: 9})
	level, total, err := store.ResolveLevel(ctx, 9)
	if err != nil || total != 25 || level.Title != "幼苗" {
		t.Fatalf("resolve: level=%+v total=%d err=%v", level, total, err)
	}
}

func TestDisableBadgeKeepsGrants(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.UpsertBadge(ctx, BadgeDefinition{Slug: "veteran", Name: "元老", Rarity: "rare", Rule: thresholdRule(t, "registered_days", 365)}); err != nil {
		t.Fatalf("upsert badge: %v", err)
	}
	if _, err := store.GrantBadge(ctx, UserBadge{UserID: 7, BadgeSlug: "veteran", Source: "auto"}); err != nil {
		t.Fatalf("grant: %v", err)
	}
	if err := store.DisableBadge(ctx, "veteran"); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if err := store.DisableBadge(ctx, "veteran"); err == nil { // 二次停用
		t.Fatalf("second disable must be not found")
	}
	grants, err := store.ListUserBadges(ctx, 7)
	if err != nil || len(grants) != 1 {
		t.Fatalf("historical grants must survive disable: grants=%v err=%v", grants, err)
	}
	enabled, err := store.ListBadges(ctx, false)
	if err != nil || len(enabled) != 0 {
		t.Fatalf("disabled badge must leave enabled list: %v %v", enabled, err)
	}
}

func TestCounters(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.IncrementCounter(ctx, 7, MetricTopicCount, 1); err != nil {
		t.Fatalf("increment: %v", err)
	}
	if err := store.IncrementCounter(ctx, 7, MetricTopicCount, 9); err != nil {
		t.Fatalf("increment: %v", err)
	}
	v, err := store.GetCounter(ctx, 7, MetricTopicCount)
	if err != nil || v != 10 {
		t.Fatalf("counter: v=%d err=%v", v, err)
	}
	if err := store.IncrementCounter(ctx, 7, "likes", 1); err == nil {
		t.Fatalf("unknown metric must be rejected")
	}
	if _, err := store.GetCounter(ctx, 7, MetricRegisteredDays); err == nil {
		t.Fatalf("registered_days must stay computed, not stored")
	}
}

// 真实 PG 集成测试：设置 SFORUM_GAMIFICATION_TEST_DATABASE_URL 时启用。
func TestPostgresStoreInvariants(t *testing.T) {
	dsn := os.Getenv("SFORUM_GAMIFICATION_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SFORUM_GAMIFICATION_TEST_DATABASE_URL not set")
	}
	t.Setenv("SFORUM_DATABASE_URL", dsn)
	// 不设 SFORUM_DATABASE_SCHEMA：本地直连走 SchemaName 回退 schema。
	store, err := newGamificationStore()
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer store.close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	// 双发授予在真实 PG 唯一约束下仍然幂等。
	grant := UserBadge{UserID: time.Now().UnixNano() % 100000, BadgeSlug: "first-topic", Source: "auto"}
	if err := store.UpsertBadge(ctx, BadgeDefinition{Slug: "first-topic", Name: "初露锋芒", Rarity: "common", Rule: thresholdRule(t, "topic_count", 1)}); err != nil {
		t.Fatalf("upsert badge: %v", err)
	}
	first, err := store.GrantBadge(ctx, grant)
	if err != nil || !first.Granted {
		t.Fatalf("first grant: %+v %v", first, err)
	}
	second, err := store.GrantBadge(ctx, grant)
	if err != nil || second.Granted || !second.Already {
		t.Fatalf("replay: %+v %v", second, err)
	}
	var count int
	if err := store.db.QueryRowContext(ctx,
		`SELECT count(*) FROM sforum_gamification.user_badges WHERE user_id = $1`, grant.UserID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one row, got %d", count)
	}
	_ = sql.ErrNoRows
}

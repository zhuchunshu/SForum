package main

import (
	"math/rand/v2"
	"strings"
	"testing"
)

func TestBuildPerfSeedPlanDefaultScale(t *testing.T) {
	opts := seedOptions{
		Profile:      seedProfilePerf1m,
		Count:        defaultPerfTopicCount,
		Users:        defaultPerfUsers,
		CommentsMax:  defaultPerfRegularComments,
		CategoryCount: defaultPerfCategoryCount,
		HotComments:  defaultPerfHotComments,
		HotSlug:      defaultPerfHotSlug,
	}
	plan, err := buildPerfSeedPlan(opts, rand.New(rand.NewPCG(1, 2)))
	if err != nil {
		t.Fatalf("buildPerfSeedPlan: %v", err)
	}

	if plan.TopicCount != 1_000_000 {
		t.Fatalf("expected 1e6 topics, got %d", plan.TopicCount)
	}
	if plan.HotComments < 50_000 {
		t.Fatalf("expected >=50k hot comments, got %d", plan.HotComments)
	}
	if plan.CategoryCount < 2 {
		t.Fatalf("expected multi-category, got %d", plan.CategoryCount)
	}
	if plan.CategorySlugs[0] != "general" {
		t.Fatalf("first category should be general, got %q", plan.CategorySlugs[0])
	}
	if plan.HotSlug != "perf-hot-thread" {
		t.Fatalf("unexpected hot slug %q", plan.HotSlug)
	}

	// 分类分布合计必须精确等于主题数。
	sum := 0
	for _, n := range plan.TopicsPerCategory {
		sum += n
		if n < 1 {
			t.Fatalf("each category should get >=1 topic when count allows, got %d", n)
		}
	}
	if sum != plan.TopicCount {
		t.Fatalf("topics per category sum %d != topic count %d", sum, plan.TopicCount)
	}

	// general 约 20%。
	generalShare := float64(plan.TopicsPerCategory[0]) / float64(plan.TopicCount)
	if generalShare < 0.15 || generalShare > 0.25 {
		t.Fatalf("general share %.2f out of expected ~0.20 band", generalShare)
	}

	// 热帖落在某分类段内。
	if plan.HotTopicIndex < 0 || plan.HotTopicIndex >= plan.TopicCount {
		t.Fatalf("hot topic index out of range: %d", plan.HotTopicIndex)
	}
	catIdx, _ := plan.categoryForTopicIndex(plan.HotTopicIndex)
	if catIdx != plan.HotTopicCategoryIndex {
		t.Fatalf("hot topic category mismatch: got %d want %d", catIdx, plan.HotTopicCategoryIndex)
	}
	if !plan.isHotTopic(plan.HotTopicIndex) {
		t.Fatal("isHotTopic should be true for HotTopicIndex")
	}
	if plan.isHotTopic(plan.HotTopicIndex + 1) {
		t.Fatal("isHotTopic should be false for neighbor")
	}

	if !strings.Contains(plan.DiskNote, "GiB") {
		t.Fatalf("disk note should mention GiB scale: %q", plan.DiskNote)
	}
	if plan.TimeNote == "" {
		t.Fatal("time note should be non-empty")
	}
}

func TestBuildPerfSeedPlanProofScale(t *testing.T) {
	// proof seed：缩小 count/hot 仍保持多分类 + 热帖结构，供 dry-run/单元测试。
	opts := seedOptions{
		Profile:       seedProfilePerf1m,
		Count:         1000,
		Users:         20,
		CommentsMax:   2,
		CategoryCount: 10,
		HotComments:   500,
		HotSlug:       "perf-hot-thread",
	}
	plan, err := buildPerfSeedPlan(opts, rand.New(rand.NewPCG(3, 4)))
	if err != nil {
		t.Fatalf("buildPerfSeedPlan: %v", err)
	}
	if plan.TopicCount != 1000 {
		t.Fatalf("want 1000 topics, got %d", plan.TopicCount)
	}
	if plan.CategoryCount != 10 {
		t.Fatalf("want 10 categories, got %d", plan.CategoryCount)
	}
	if plan.HotComments != 500 {
		t.Fatalf("want 500 hot comments, got %d", plan.HotComments)
	}
	sum := 0
	for _, n := range plan.TopicsPerCategory {
		sum += n
	}
	if sum != 1000 {
		t.Fatalf("distribution sum %d", sum)
	}
}

func TestApplyPerfProfileDefaults(t *testing.T) {
	opts := seedOptions{Profile: seedProfilePerf1m, Count: 1000, Users: 50, CommentsMax: 5, Batch: 20}
	applyPerfProfileDefaults(&opts, false, false, false)
	if opts.Count != defaultPerfTopicCount {
		t.Fatalf("count default: got %d", opts.Count)
	}
	if opts.Users != defaultPerfUsers {
		t.Fatalf("users default: got %d", opts.Users)
	}
	if opts.CommentsMax != defaultPerfRegularComments {
		t.Fatalf("comments-max default: got %d", opts.CommentsMax)
	}
	if opts.CategoryCount != defaultPerfCategoryCount {
		t.Fatalf("categories default: got %d", opts.CategoryCount)
	}
	if opts.HotComments != defaultPerfHotComments {
		t.Fatalf("hot-comments default: got %d", opts.HotComments)
	}
	if opts.HotSlug != defaultPerfHotSlug {
		t.Fatalf("hot-slug default: got %q", opts.HotSlug)
	}

	// 显式 flag 不被覆盖。
	opts2 := seedOptions{Profile: seedProfilePerf1m, Count: 1234, Users: 9, CommentsMax: 3}
	applyPerfProfileDefaults(&opts2, true, true, true)
	if opts2.Count != 1234 || opts2.Users != 9 || opts2.CommentsMax != 3 {
		t.Fatalf("explicit flags should stick: %+v", opts2)
	}

	// small 不受影响。
	opts3 := seedOptions{Profile: seedProfileSmall, Count: 100}
	applyPerfProfileDefaults(&opts3, false, false, false)
	if opts3.Count != 100 {
		t.Fatalf("small profile should not change count")
	}
}

func TestValidateSeedOptionsProfile(t *testing.T) {
	err := validateSeedOptions(&seedOptions{Profile: "nope", Count: 1, Users: 1})
	if err == nil {
		t.Fatal("expected error for bad profile")
	}
	err = validateSeedOptions(&seedOptions{Profile: seedProfilePerf1m, Count: 1, Users: 1, HotComments: -1})
	if err == nil {
		t.Fatal("expected error for negative hot-comments")
	}
	if err := validateSeedOptions(&seedOptions{Profile: seedProfilePerf1m, Count: 10, Users: 2, HotComments: 100}); err != nil {
		t.Fatalf("valid perf opts: %v", err)
	}
}

func TestValidateSeedOptionsExistingCasesStillPass(t *testing.T) {
	// 兼容旧 small 用例：Profile 空时 runSeedCommand 会填 small；validate 要求显式合法 profile。
	opts := seedOptions{Profile: seedProfileSmall, Count: 100, Users: 10, CommentsMax: 5, Batch: 20}
	if err := validateSeedOptions(&opts); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

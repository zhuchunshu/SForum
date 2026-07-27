package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// 规模种子 profile 名称（与 --profile flag 对齐）。
const (
	seedProfileSmall  = "small"
	seedProfilePerf1m = "perf-1m"
)

// perf 默认目标（可被 flag 覆盖，便于 proof seed / dry-run 测试）。
const (
	defaultPerfTopicCount    = 1_000_000
	defaultPerfUsers         = 200
	defaultPerfCategoryCount = 20
	defaultPerfHotComments   = 50_000
	// 普通主题默认不灌评论：成功指标依赖热帖 5e4 评论 + 列表/详情路径；
	// 全表每帖评论会把 1e6 seed 的磁盘与时间再翻倍。需要时可 --comments-max=N。
	defaultPerfRegularComments = 0
	defaultPerfHotSlug         = "perf-hot-thread"
	// 磁盘/时间量级说明（help 与 dry-run 共用）。
	perfDiskOrderOfMagnitude = "约 5–15 GiB PostgreSQL（含索引），视评论量与正文长度浮动"
	perfTimeOrderOfMagnitude = "bulk COPY 路径约 10–60 分钟（SSD + 本机 Docker PG）；Service 逐条路径不可行"
)

// perfSeedPlan 是 perf 规模种子的纯规划结果：不物化百万条正文，只描述计数与分布，
// 供 dry-run / 单元测试断言，以及 bulk 写入路径按计划流式生成。
type perfSeedPlan struct {
	Profile       string
	Users         int
	TopicCount    int
	CategoryCount int
	CategorySlugs []string // 含 general + perf-cat-XX
	// TopicsPerCategory[i] 对应 CategorySlugs[i] 的目标主题数（合计 = TopicCount）。
	TopicsPerCategory  []int
	RegularCommentsMax int
	// ExpectedRegularComments 是普通主题评论期望总数的上界估计（每帖 0..max 均匀）。
	ExpectedRegularCommentsMax int
	HotTopicIndex              int // 在全部主题序号中的下标（0-based）
	HotTopicCategoryIndex      int
	HotComments                int
	HotSlug                    string
	// EstimatedTotalComments = ExpectedRegularCommentsMax + HotComments（上界）。
	EstimatedTotalCommentsMax int
	DiskNote                  string
	TimeNote                  string
}

// applyPerfProfileDefaults 在 profile=perf-1m 时填入默认规模；显式 flag 优先。
// countExplicit / usersExplicit 表示调用方是否改过对应 flag。
func applyPerfProfileDefaults(opts *seedOptions, countExplicit, usersExplicit, commentsMaxExplicit bool) {
	if opts == nil || opts.Profile != seedProfilePerf1m {
		return
	}
	if !countExplicit {
		opts.Count = defaultPerfTopicCount
	}
	if !usersExplicit {
		opts.Users = defaultPerfUsers
	}
	if !commentsMaxExplicit {
		opts.CommentsMax = defaultPerfRegularComments
	}
	if opts.CategoryCount <= 0 {
		opts.CategoryCount = defaultPerfCategoryCount
	}
	if opts.HotComments <= 0 {
		opts.HotComments = defaultPerfHotComments
	}
	if strings.TrimSpace(opts.HotSlug) == "" {
		opts.HotSlug = defaultPerfHotSlug
	}
	if opts.Batch <= 0 {
		opts.Batch = 5000
	}
}

// buildPerfSeedPlan 根据 opts 生成规模计划（确定性，除 rng 驱动的分布抖动外）。
// 不分配 TopicCount 级别的切片正文，保证 dry-run 与单元测试可在秒级完成。
func buildPerfSeedPlan(opts seedOptions, rng *rand.Rand) (perfSeedPlan, error) {
	if opts.Count <= 0 {
		return perfSeedPlan{}, fmt.Errorf("perf plan requires positive topic count")
	}
	if opts.Users <= 0 {
		return perfSeedPlan{}, fmt.Errorf("perf plan requires positive user count")
	}
	catCount := opts.CategoryCount
	if catCount <= 0 {
		catCount = defaultPerfCategoryCount
	}
	if catCount > opts.Count {
		catCount = opts.Count
	}
	hotComments := opts.HotComments
	if hotComments <= 0 {
		hotComments = defaultPerfHotComments
	}
	hotSlug := strings.TrimSpace(opts.HotSlug)
	if hotSlug == "" {
		hotSlug = defaultPerfHotSlug
	}
	regularMax := opts.CommentsMax
	if regularMax < 0 {
		regularMax = 0
	}

	slugs := make([]string, 0, catCount)
	// 第一分类固定 general，便于与现有默认分类共存（append-only）。
	slugs = append(slugs, "general")
	for i := 1; i < catCount; i++ {
		slugs = append(slugs, fmt.Sprintf("perf-cat-%02d", i))
	}

	// 分布：约 20% 主题落入 general（首页/大分类场景），其余均分；
	// 再把余数摊到前面几个分类，保证合计精确等于 TopicCount。
	perCat := make([]int, catCount)
	if catCount == 1 {
		perCat[0] = opts.Count
	} else {
		generalShare := opts.Count / 5 // 20%
		if generalShare < 1 {
			generalShare = 1
		}
		if generalShare > opts.Count-(catCount-1) {
			generalShare = opts.Count - (catCount - 1)
		}
		perCat[0] = generalShare
		rest := opts.Count - generalShare
		base := rest / (catCount - 1)
		rem := rest % (catCount - 1)
		for i := 1; i < catCount; i++ {
			perCat[i] = base
			if rem > 0 {
				perCat[i]++
				rem--
			}
		}
	}

	// 热帖优先放在主题量最大的分类（通常为 general 的 ~20% 份额），
	// 便于「单分类 200k 级」列表与热帖详情同时测到。
	hotCatIdx := 0
	maxInCat := perCat[0]
	for i := 1; i < catCount; i++ {
		if perCat[i] > maxInCat {
			maxInCat = perCat[i]
			hotCatIdx = i
		}
	}
	// 热帖在全局序号中的位置：取该分类段的第一条（写入时按分类顺序展开）。
	hotTopicIndex := 0
	for i := 0; i < hotCatIdx; i++ {
		hotTopicIndex += perCat[i]
	}

	// 普通评论上界：每帖最多 regularMax 条（均匀 0..max 时期望约为 max/2）。
	// 热帖本身也计入 TopicCount，其「普通」评论由 HotComments 覆盖，这里按 TopicCount-1 估上界。
	regularTopics := opts.Count - 1
	if regularTopics < 0 {
		regularTopics = 0
	}
	expectedRegularMax := regularTopics * regularMax

	_ = rng // 预留：未来可对 perCat 做轻微抖动；当前保持确定性便于测试。

	return perfSeedPlan{
		Profile:                    seedProfilePerf1m,
		Users:                      opts.Users,
		TopicCount:                 opts.Count,
		CategoryCount:              catCount,
		CategorySlugs:              slugs,
		TopicsPerCategory:          perCat,
		RegularCommentsMax:         regularMax,
		ExpectedRegularCommentsMax: expectedRegularMax,
		HotTopicIndex:              hotTopicIndex,
		HotTopicCategoryIndex:      hotCatIdx,
		HotComments:                hotComments,
		HotSlug:                    hotSlug,
		EstimatedTotalCommentsMax:  expectedRegularMax + hotComments,
		DiskNote:                   perfDiskOrderOfMagnitude,
		TimeNote:                   perfTimeOrderOfMagnitude,
	}, nil
}

// categoryTopicIndex 返回全局主题序号 topicIndex（0-based）所属的分类下标与分类内偏移。
func (p perfSeedPlan) categoryForTopicIndex(topicIndex int) (catIdx int, offsetInCat int) {
	if topicIndex < 0 || topicIndex >= p.TopicCount {
		return 0, 0
	}
	remaining := topicIndex
	for i, n := range p.TopicsPerCategory {
		if remaining < n {
			return i, remaining
		}
		remaining -= n
	}
	return len(p.TopicsPerCategory) - 1, 0
}

// isHotTopic 判断全局主题序号是否为热帖。
func (p perfSeedPlan) isHotTopic(topicIndex int) bool {
	return topicIndex == p.HotTopicIndex
}

func (p perfSeedPlan) String() string {
	return fmt.Sprintf(
		"profile=%s topics=%d users=%d categories=%d hot_comments=%d hot_slug=%q regular_comments_max/topic=%d est_comments_max=%d",
		p.Profile, p.TopicCount, p.Users, p.CategoryCount, p.HotComments, p.HotSlug, p.RegularCommentsMax, p.EstimatedTotalCommentsMax,
	)
}

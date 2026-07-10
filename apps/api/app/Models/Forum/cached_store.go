package forum

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Cache"
)

// CachedStore 通过嵌入 Store 接口装饰底层 store：
//   - 读方法（ListCategories/ListCategoryGroups/ListTags/GetTopic/ListTopics）走短 TTL 缓存。
//   - 写方法在成功后使相关缓存失效（generation 递增 + 精确 key 删除）。
//
// 失效策略采用 generation 版本号，避免对前缀 key 做 SCAN：
//   - 主题列表 key 含 topics generation，写主题/评论时递增，旧 key 自然过期。
//   - 分类/分组/标签列表同理。
//   - 主题详情按 topicID 精确删除（创建/更新/删除/状态变更时）。
//
// cache 为 nil 时退化为透传（不缓存），方便测试和显式关闭。
type CachedStore struct {
	Store
	cache cache.Cache
}

// 生成 generation key 时使用的固定前缀。
const (
	genTopics         = "forum:gen:topics" // 主题列表 generation
	genCategories     = "forum:gen:cats"   // 分类列表 generation
	genCategoryGroups = "forum:gen:groups" // 分类分组列表 generation
	genTags           = "forum:gen:tags"   // 标签列表 generation

	// 缓存 key 前缀。
	prefixTopicDetail = "forum:topic:"
	prefixTopicBySlug = "forum:topic-slug:"
	prefixTopicsList  = "forum:topics:"
	prefixCatsList    = "forum:cats"
	prefixGroupsList  = "forum:groups"
	prefixTagsList    = "forum:tags"

	ttlList        = 15 * time.Second // 主题列表：变化较频繁
	ttlTaxonomy    = 60 * time.Second // 分类/分组/标签：变化极少
	ttlTopicDetail = 30 * time.Second // 主题详情
)

// NewCachedStore 包装底层 store。cache 为 nil 时直接返回底层 store（不装饰）。
func NewCachedStore(inner Store, cache cache.Cache) Store {
	if cache == nil || inner == nil {
		return inner
	}
	return &CachedStore{Store: inner, cache: cache}
}

// currentGen 读取 generation 当前值（不递增），用于构造读缓存 key。
// 失败或 key 不存在时返回 "0"，等价于"首代"——此时缓存 miss，回源后写入，
// 行为与无 generation 一致。
func (s *CachedStore) currentGen(ctx context.Context, key string) string {
	raw, found, err := s.cache.Get(ctx, key)
	if err != nil {
		slog.WarnContext(ctx, "forum cache: read generation failed", "key", key, "err", err)
		return "0"
	}
	if !found {
		return "0"
	}
	return string(raw)
}

// bumpGen 递增 generation 版本号。
// 采用 Get→Set 的非原子递增：对短 TTL 缓存（15-60s）可接受，最坏情况下
// 并发写入产生相同版本号，导致旧列表多缓存一个 TTL 周期，数据最终一致。
// 这避免了 Cache 接口必须暴露原子 Incr 的强约束，简化实现。
func (s *CachedStore) bumpGen(ctx context.Context, key string) {
	current := s.currentGen(ctx, key)
	n, err := strconv.ParseInt(current, 10, 64)
	if err != nil {
		n = 0
	}
	n++
	_ = s.cache.Set(ctx, key, []byte(strconv.FormatInt(n, 10)), 24*time.Hour)
}

// --- 读方法：带缓存 ---

func (s *CachedStore) ListCategories(ctx context.Context) ([]Category, error) {
	var out []Category
	if s.loadJSON(ctx, prefixCatsList, &out) {
		return out, nil
	}
	out, err := s.Store.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	s.saveJSON(ctx, prefixCatsList, out, ttlTaxonomy)
	return out, nil
}

func (s *CachedStore) ListCategoryGroups(ctx context.Context) ([]CategoryGroup, error) {
	var out []CategoryGroup
	if s.loadJSON(ctx, prefixGroupsList, &out) {
		return out, nil
	}
	out, err := s.Store.ListCategoryGroups(ctx)
	if err != nil {
		return nil, err
	}
	s.saveJSON(ctx, prefixGroupsList, out, ttlTaxonomy)
	return out, nil
}

func (s *CachedStore) ListTags(ctx context.Context, includePending bool) ([]Tag, error) {
	key := prefixTagsList
	if includePending {
		key = prefixTagsList + ":all"
	}
	var out []Tag
	if s.loadJSON(ctx, key, &out) {
		return out, nil
	}
	out, err := s.Store.ListTags(ctx, includePending)
	if err != nil {
		return nil, err
	}
	s.saveJSON(ctx, key, out, ttlTaxonomy)
	return out, nil
}

func (s *CachedStore) GetTopic(ctx context.Context, topicID int64) (TopicDetail, error) {
	key := fmt.Sprintf("%s%d", prefixTopicDetail, topicID)
	var out TopicDetail
	if s.loadJSON(ctx, key, &out) {
		return out, nil
	}
	out, err := s.Store.GetTopic(ctx, topicID)
	if err != nil {
		return TopicDetail{}, err
	}
	s.saveJSON(ctx, key, out, ttlTopicDetail)
	return out, nil
}

func (s *CachedStore) ListAuthorReviewItems(ctx context.Context, authorUserID int64) (AuthorReviewList, error) {
	store, ok := s.Store.(AuthorReviewStore)
	if !ok {
		return AuthorReviewList{}, ErrInvalidAction
	}
	return store.ListAuthorReviewItems(ctx, authorUserID)
}

// GetTopicBySlug 缓存按 slug 查询的主题详情。slug 模式下访问量高，缓存命中可减少 DB 压力。
func (s *CachedStore) GetTopicBySlug(ctx context.Context, slug string) (TopicDetail, error) {
	key := prefixTopicBySlug + slug
	var out TopicDetail
	if s.loadJSON(ctx, key, &out) {
		return out, nil
	}
	out, err := s.Store.GetTopicBySlug(ctx, slug)
	if err != nil {
		return TopicDetail{}, err
	}
	s.saveJSON(ctx, key, out, ttlTopicDetail)
	return out, nil
}

// TopicSlugExists 不缓存：写路径的实时唯一性校验，必须读最新数据。
func (s *CachedStore) TopicSlugExists(ctx context.Context, slug string, excludeTopicID int64) (bool, error) {
	return s.Store.TopicSlugExists(ctx, slug, excludeTopicID)
}

func (s *CachedStore) ListTopics(ctx context.Context, input TopicListInput) (TopicList, error) {
	gen := s.currentGen(ctx, genTopics)
	key := fmt.Sprintf("%s%s:%s:%s:%d:%d", prefixTopicsList, gen, input.CategorySlug, input.TagSlug, input.Page, input.PerPage)
	var out TopicList
	if s.loadJSON(ctx, key, &out) {
		return out, nil
	}
	out, err := s.Store.ListTopics(ctx, input)
	if err != nil {
		return TopicList{}, err
	}
	s.saveJSON(ctx, key, out, ttlList)
	return out, nil
}

// --- 写方法：成功后失效 ---

func (s *CachedStore) CreateTopic(ctx context.Context, input CreateTopicRecord) (TopicDetail, error) {
	out, err := s.Store.CreateTopic(ctx, input)
	if err != nil {
		return out, err
	}
	s.invalidateTopics(ctx)
	s.invalidateTaxonomy(ctx)
	s.invalidateTopicBySlug(ctx, out.Slug)
	return out, nil
}

func (s *CachedStore) UpdateTopic(ctx context.Context, input UpdateTopicRecord) (TopicDetail, error) {
	out, err := s.Store.UpdateTopic(ctx, input)
	if err != nil {
		return out, err
	}
	s.invalidateTopicDetail(ctx, input.TopicID)
	s.invalidateTopics(ctx)
	s.invalidateTaxonomy(ctx)
	s.invalidateTopicBySlug(ctx, out.Slug)
	return out, nil
}

func (s *CachedStore) DeleteTopic(ctx context.Context, topicID int64) (TopicDetail, error) {
	out, err := s.Store.DeleteTopic(ctx, topicID)
	if err != nil {
		return out, err
	}
	s.invalidateTopicDetail(ctx, topicID)
	s.invalidateTopicBySlug(ctx, out.Slug)
	s.invalidateTopics(ctx)
	s.invalidateTaxonomy(ctx)
	return out, nil
}

func (s *CachedStore) ApplyTopicAction(ctx context.Context, input TopicLifecycleInput) (TopicLifecycleRecord, error) {
	out, err := s.Store.ApplyTopicAction(ctx, input)
	if err != nil {
		return out, err
	}
	s.invalidateTopicDetail(ctx, input.TopicID)
	s.invalidateTopics(ctx)
	return out, nil
}

func (s *CachedStore) CreateComment(ctx context.Context, input CreateCommentRecord) (Comment, error) {
	out, err := s.Store.CreateComment(ctx, input)
	if err != nil {
		return out, err
	}
	// 新评论更新主题 last_activity_at，影响列表排序与详情评论计数。
	s.invalidateTopicDetail(ctx, input.TopicID)
	s.invalidateTopics(ctx)
	s.invalidateTaxonomy(ctx)
	return out, nil
}

func (s *CachedStore) CreateCategory(ctx context.Context, input CreateCategoryInput) (Category, error) {
	out, err := s.Store.CreateCategory(ctx, input)
	if err != nil {
		return out, err
	}
	s.invalidateTaxonomy(ctx)
	return out, nil
}

func (s *CachedStore) UpdateCategory(ctx context.Context, input UpdateCategoryInput) (Category, error) {
	out, err := s.Store.UpdateCategory(ctx, input)
	if err != nil {
		return out, err
	}
	s.invalidateTaxonomy(ctx)
	return out, nil
}

func (s *CachedStore) CreateCategoryGroup(ctx context.Context, input CreateCategoryGroupInput) (CategoryGroup, error) {
	out, err := s.Store.CreateCategoryGroup(ctx, input)
	if err != nil {
		return out, err
	}
	s.invalidateTaxonomy(ctx)
	return out, nil
}

func (s *CachedStore) UpdateCategoryGroup(ctx context.Context, input UpdateCategoryGroupInput) (CategoryGroup, error) {
	out, err := s.Store.UpdateCategoryGroup(ctx, input)
	if err != nil {
		return out, err
	}
	s.invalidateTaxonomy(ctx)
	return out, nil
}

func (s *CachedStore) CreateTag(ctx context.Context, input CreateTagInput) (Tag, error) {
	out, err := s.Store.CreateTag(ctx, input)
	if err != nil {
		return out, err
	}
	s.invalidateTaxonomy(ctx)
	return out, nil
}

func (s *CachedStore) UpdateTag(ctx context.Context, input UpdateTagInput) (Tag, error) {
	out, err := s.Store.UpdateTag(ctx, input)
	if err != nil {
		return out, err
	}
	s.invalidateTaxonomy(ctx)
	return out, nil
}

// --- 失效辅助 ---

func (s *CachedStore) invalidateTopics(ctx context.Context) {
	s.bumpGen(ctx, genTopics)
}

func (s *CachedStore) invalidateTaxonomy(ctx context.Context) {
	s.bumpGen(ctx, genCategories)
	s.bumpGen(ctx, genCategoryGroups)
	s.bumpGen(ctx, genTags)
	_ = s.cache.Delete(ctx, prefixCatsList, prefixGroupsList, prefixTagsList, prefixTagsList+":all")
}

func (s *CachedStore) invalidateTopicDetail(ctx context.Context, topicID int64) {
	_ = s.cache.Delete(ctx, fmt.Sprintf("%s%d", prefixTopicDetail, topicID))
}

// invalidateTopicBySlug 清除按 slug 缓存的详情条目。用于写操作后保证 slug 维度缓存及时更新；
// 旧 slug 的残留条目依赖 TTL（30s）自然过期，规范化 301 兜底。
func (s *CachedStore) invalidateTopicBySlug(ctx context.Context, slug string) {
	if slug == "" {
		return
	}
	_ = s.cache.Delete(ctx, prefixTopicBySlug+slug)
}

// --- JSON 序列化辅助 ---

// loadJSON 尝试从缓存加载并反序列化。命中返回 true。
func (s *CachedStore) loadJSON(ctx context.Context, key string, dst any) bool {
	raw, found, err := s.cache.Get(ctx, key)
	if err != nil {
		slog.WarnContext(ctx, "forum cache: get failed", "key", key, "err", err)
		return false
	}
	if !found {
		return false
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		slog.WarnContext(ctx, "forum cache: unmarshal failed", "key", key, "err", err)
		return false
	}
	return true
}

// saveJSON 序列化并写入缓存。失败只记日志（缓存写失败不应影响主流程）。
func (s *CachedStore) saveJSON(ctx context.Context, key string, value any, ttl time.Duration) {
	raw, err := json.Marshal(value)
	if err != nil {
		slog.WarnContext(ctx, "forum cache: marshal failed", "key", key, "err", err)
		return
	}
	if err := s.cache.Set(ctx, key, raw, ttl); err != nil {
		slog.WarnContext(ctx, "forum cache: set failed", "key", key, "err", err)
	}
}

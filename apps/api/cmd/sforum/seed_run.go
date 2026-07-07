package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

// seedDeps 把 runSeed 依赖的 service/store 收拢在一起，方便命令装配。
// forumStore 用于 ListCategories（service.ListCategories 直接透传 store），
// 其余写操作都走 service，保证权限、渲染、计数器与真实路径一致。
type seedDeps struct {
	identityService identityService
	forumService    forumService
}

// identityService 是 identity.Service 在 seed 场景下用到的方法子集。
// 抽成接口是为了将来能在测试里注入假实现，符合现有 controller 的接口隔离风格。
type identityService interface {
	Register(ctx context.Context, input identity.RegisterInput) (identity.CurrentUser, error)
}

// forumService 是 forum.Service 在 seed 场景下用到的方法子集。
type forumService interface {
	ListCategories(ctx context.Context) ([]forum.Category, error)
	CreateTopic(ctx context.Context, actor identity.Actor, input forum.CreateTopicInput) (forum.TopicDetail, error)
	CreateComment(ctx context.Context, actor identity.Actor, input forum.CreateCommentInput) (forum.Comment, error)
}

// actorLoader 从 userID 加载带权限的 Actor。对应 identity.Store.LoadActor。
type actorLoader interface {
	LoadActor(ctx context.Context, userID int64) (identity.Actor, error)
}

// seedResult 记录一次 seed 运行的产物统计。
type seedResult struct {
	UsersCreated    int
	TopicsCreated   int
	CommentsCreated int
	Elapsed         time.Duration
}

func (r seedResult) String() string {
	return fmt.Sprintf(
		"%d users, %d topics, %d comments in %s",
		r.UsersCreated, r.TopicsCreated, r.CommentsCreated, r.Elapsed.Round(time.Millisecond),
	)
}

// runSeed 按 dataset 逐条把假数据写入数据库。
//
// 写入路径完全复用领域 Service：
//   - 用户走 identity.Register（自动分配 member 角色、走 argon2 密码哈希）
//   - 主题/评论走 forum.CreateTopic / CreateComment（自动渲染 Markdown、生成 slug、
//     维护 path_key/depth、更新各类计数器、做权限检查）
//
// 因此种子数据与真实用户产生的数据在完整性上没有差别。代价是速度：每条记录一个事务，
// 串行执行。这是开发/测试种子数据，简单可靠优先于吞吐。
func runSeed(
	ctx context.Context,
	opts seedOptions,
	dataset seedDataset,
	deps seedDeps,
	actors actorLoader,
	logf func(format string, args ...any),
) (seedResult, error) {
	start := time.Now()
	result := seedResult{}

	// 1. 校验目标分类存在。空则用 forum 默认 general。
	categorySlug := strings.TrimSpace(opts.CategorySlug)
	categories, err := deps.forumService.ListCategories(ctx)
	if err != nil {
		return result, fmt.Errorf("list categories: %w", err)
	}
	if categorySlug == "" {
		categorySlug = "general"
	}
	if !categoryExists(categories, categorySlug) {
		available := categorySlugs(categories)
		return result, fmt.Errorf("category %q not found; available: %v", categorySlug, available)
	}

	// 2. 批量注册假用户。冲突时重生成后缀重试。
	userIDs := make([]int64, 0, len(dataset.Users))
	for _, u := range dataset.Users {
		current, err := registerSeedUser(ctx, deps.identityService, u)
		if err != nil {
			return result, fmt.Errorf("register seed user %q: %w", u.Username, err)
		}
		userIDs = append(userIDs, current.ID)
		result.UsersCreated++
		if result.UsersCreated%10 == 0 {
			logf("  registered %d/%d users", result.UsersCreated, len(dataset.Users))
		}
	}
	logf("registered %d seed users", result.UsersCreated)

	// 3. 逐个主题写入，并在主题下生成评论。
	for i, plan := range dataset.Topics {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		authorActor, err := actors.LoadActor(ctx, userIDs[plan.Topic.AuthorIndex])
		if err != nil {
			return result, fmt.Errorf("load actor for topic %d: %w", i+1, err)
		}

		topic, err := deps.forumService.CreateTopic(ctx, authorActor, forum.CreateTopicInput{
			CategorySlug: categorySlug,
			Title:        plan.Topic.Title,
			Content: forum.ContentInput{
				RawContent:   plan.Topic.Body,
				SourceFormat: forum.SourceFormatMarkdown,
				EditorType:   forum.EditorTypeMarkdown,
			},
		})
		if err != nil {
			return result, fmt.Errorf("create topic %d %q: %w", i+1, plan.Topic.Title, err)
		}
		result.TopicsCreated++

		// 在当前主题下生成评论。topicCommentIDs 按创建顺序保存已建评论 ID，
		// 用于把 seedComment.ParentOffset 映射成真实评论 ID。
		topicCommentIDs := make([]int64, 0, len(plan.Comments))
		for _, c := range plan.Comments {
			commenterActor, err := actors.LoadActor(ctx, userIDs[c.AuthorIndex])
			if err != nil {
				return result, fmt.Errorf("load actor for comment: %w", err)
			}

			input := forum.CreateCommentInput{
				TopicID: topic.ID,
				Content: forum.ContentInput{
					RawContent:   c.Body,
					SourceFormat: forum.SourceFormatMarkdown,
					EditorType:   forum.EditorTypeMarkdown,
				},
			}
			if c.ParentOffset >= 0 && c.ParentOffset < len(topicCommentIDs) {
				// Service 层会自行加载 parent summary 并维护 path_key/depth/reply_count。
				parentID := topicCommentIDs[c.ParentOffset]
				input.ParentID = &parentID
			}

			created, err := deps.forumService.CreateComment(ctx, commenterActor, input)
			if err != nil {
				return result, fmt.Errorf("create comment on topic %d: %w", topic.ID, err)
			}
			topicCommentIDs = append(topicCommentIDs, created.ID)
			result.CommentsCreated++
		}

		if opts.Batch > 0 && result.TopicsCreated%opts.Batch == 0 {
			logf("  seeded %d/%d topics (%d comments so far)", result.TopicsCreated, len(dataset.Topics), result.CommentsCreated)
		}
	}

	result.Elapsed = time.Since(start)
	return result, nil
}

// registerSeedUser 注册一个种子用户。用户名/邮箱冲突时换后缀重试最多 3 次，
// 以支持重复运行 seed:forum 而不报错（追加生成语义）。
func registerSeedUser(ctx context.Context, svc identityService, base seedUser) (identity.CurrentUser, error) {
	// 首次直接用预生成的随机后缀尝试；冲突再重抽。
	current, err := svc.Register(ctx, identity.RegisterInput{
		Username:    base.Username,
		Email:       base.Email,
		Password:    base.Password,
		DisplayName: base.DisplayName,
	})
	if err == nil {
		return current, nil
	}
	// 只对参数冲突（用户名/邮箱已存在）重试，其它错误直接上抛。
	var invalid *identity.RegisterInvalidError
	if !errors.As(err, &invalid) {
		return identity.CurrentUser{}, err
	}

	for attempt := 0; attempt < 3; attempt++ {
		retry, regenErr := generateSeedUser(0)
		if regenErr != nil {
			return identity.CurrentUser{}, regenErr
		}
		current, err = svc.Register(ctx, identity.RegisterInput{
			Username:    retry.Username,
			Email:       retry.Email,
			Password:    retry.Password,
			DisplayName: base.DisplayName, // 保留原序号显示名
		})
		if err == nil {
			return current, nil
		}
		if !errors.As(err, &invalid) {
			return identity.CurrentUser{}, err
		}
	}
	return identity.CurrentUser{}, fmt.Errorf("seed user registration kept conflicting after retries: %w", err)
}

func categoryExists(categories []forum.Category, slug string) bool {
	for _, c := range categories {
		if c.Slug == slug {
			return true
		}
	}
	return false
}

func categorySlugs(categories []forum.Category) []string {
	out := make([]string, 0, len(categories))
	for _, c := range categories {
		out = append(out, c.Slug)
	}
	return out
}

// newSeededRand 构造一个带随机种子的 *rand.Rand，用于生成可复现的 dataset。
// 使用 math/rand/v2 的 PCG 实现，足够种子场景使用。
func newSeededRand() *rand.Rand {
	return rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano()<<1)))
}

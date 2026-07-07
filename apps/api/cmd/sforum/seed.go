package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Postgres"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

// newSeedCommand 构造 `sforum seed:forum` 子命令。
//
// 它复用 config.Load() 解析数据库连接（只读环境变量，不读 .env，与 cmd/migrate 一致），
// 用 postgres.NewPool 建池，再用 identity/forum 的 PostgresStore + Service 写入数据。
func newSeedCommand() *cobra.Command {
	opts := seedOptions{}
	cmd := &cobra.Command{
		Use:   "seed:forum",
		Short: "批量生成假论坛数据（用户、主题、评论）",
		Long: `批量生成假论坛数据用于本地开发和测试。

复用领域 Service 写入，保证密码哈希、Markdown 渲染、slug、评论树 path_key、
各类计数器与真实用户数据一致。可重复运行，每次追加生成（用户名/邮箱带随机后缀）。

由于 config.Load() 只读环境变量、不读 .env，运行前需先把 .env 导入环境：

  set -a; . ./.env; set +a
  go run ./apps/api/cmd/sforum seed:forum --count=1000

或用 --database-url 显式覆盖连接串。

注意：此命令面向开发/测试环境，请勿在生产数据库运行。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSeedCommand(cmd.Context(), opts, cmd)
		},
	}
	cmd.Flags().IntVar(&opts.Count, "count", 1000, "生成的主题数量")
	cmd.Flags().IntVar(&opts.Users, "users", 50, "预先创建的假用户数量")
	cmd.Flags().IntVar(&opts.CommentsMax, "comments-max", 5, "每个主题最多生成的评论数（0 表示不生成评论）")
	cmd.Flags().StringVar(&opts.CategorySlug, "category-slug", "", "主题发布到的分类 slug（默认 general）")
	cmd.Flags().StringVar(&opts.DatabaseURL, "database-url", "", "覆盖 DATABASE_URL；空则用环境变量")
	cmd.Flags().IntVar(&opts.Batch, "batch", 20, "进度日志频率（每 N 条主题打印一次）")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "只打印计划，不写数据库")
	return cmd
}

func runSeedCommand(ctx context.Context, opts seedOptions, cmd *cobra.Command) error {
	if err := validateSeedOptions(&opts); err != nil {
		return err
	}

	// 1. 先生成内存计划。dry-run 下只打印它就返回，不连库。
	rng := newSeededRand()
	dataset, err := generateSeedDataset(opts, rng)
	if err != nil {
		return fmt.Errorf("generate seed dataset: %w", err)
	}
	if opts.DryRun {
		printDryRun(cmd, opts, dataset)
		return nil
	}

	// 2. 解析数据库 URL：flag 优先，否则走 config.Load() 读环境变量。
	databaseURL := opts.DatabaseURL
	if databaseURL == "" {
		databaseURL = config.Load().DatabaseURL
	}
	if databaseURL == "" {
		return fmt.Errorf("database url is empty: set DATABASE_URL or pass --database-url")
	}

	pool, err := postgres.NewPool(ctx, databaseURL, 10)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	// 3. 装配 stores + services。用不带 events 的 NewService，避免种子数据触发
	// 搜索索引/通知等副作用；publisher=nil 时 Emit 返回默认 OK 不阻塞。
	identityStore := identity.NewPostgresStore(pool)
	forumStore := forum.NewPostgresStore(pool)
	deps := seedDeps{
		identityService: identity.NewService(identityStore),
		forumService:    forum.NewService(forumStore),
	}

	cmd.Printf("seeding forum: %d topics, %d users, up to %d comments/topic\n", opts.Count, opts.Users, opts.CommentsMax)
	result, err := runSeed(ctx, opts, dataset, deps, identityStore, cmd.Printf)
	if err != nil {
		return err
	}
	cmd.Printf("done: %s\n", result.String())
	return nil
}

// validateSeedOptions 规整并校验 flag，给出可读的边界错误。
func validateSeedOptions(opts *seedOptions) error {
	if opts.Count <= 0 {
		return fmt.Errorf("--count must be positive")
	}
	if opts.Users <= 0 {
		return fmt.Errorf("--users must be positive")
	}
	if opts.CommentsMax < 0 {
		return fmt.Errorf("--comments-max must be >= 0")
	}
	if opts.Batch < 0 {
		return fmt.Errorf("--batch must be >= 0")
	}
	return nil
}

// printDryRun 打印内存计划的摘要，不连库也不写任何数据。
func printDryRun(cmd *cobra.Command, opts seedOptions, dataset seedDataset) {
	totalComments := 0
	for _, p := range dataset.Topics {
		totalComments += len(p.Comments)
	}
	cmd.Printf("dry-run plan (no database writes)\n")
	cmd.Printf("  users:    %d\n", len(dataset.Users))
	cmd.Printf("  topics:   %d\n", len(dataset.Topics))
	cmd.Printf("  comments: %d\n", totalComments)
	if len(dataset.Users) > 0 {
		cmd.Printf("  sample user: %s <%s>\n", dataset.Users[0].Username, dataset.Users[0].Email)
	}
	if len(dataset.Topics) > 0 {
		t := dataset.Topics[0].Topic
		cmd.Printf("  sample topic: [user#%d] %s (%d body chars, %d comments)\n",
			t.AuthorIndex, t.Title, len(t.Body), len(dataset.Topics[0].Comments))
	}
}

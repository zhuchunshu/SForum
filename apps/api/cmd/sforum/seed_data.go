package main

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"strings"
)

// seedOptions 对应 seed:forum 子命令的 flags，是 runSeed 的输入。
type seedOptions struct {
	Count        int    // 要生成的主题数量
	Users        int    // 预先创建的假用户数量
	CommentsMax  int    // 每个主题最多生成的评论数（0 表示不生成评论）
	CategorySlug string // 主题发布到的分类 slug，空则用默认分类
	DatabaseURL  string // 覆盖 DATABASE_URL，空则用 config.Load() 读环境变量
	Batch        int    // 进度日志频率，每生成 N 条主题打印一次
	DryRun       bool   // 只打印计划不写库
}

// seedUser 描述一个待注册的假用户。
type seedUser struct {
	Username    string
	Email       string
	DisplayName string
	Password    string
}

// seedTopic 描述一个待创建的假主题（不含评论）。
type seedTopic struct {
	AuthorIndex int // 作者在 userIDs 中的下标
	Title       string
	Body        string // Markdown 正文
}

// seedComment 描述一条待创建的假评论。
type seedComment struct {
	AuthorIndex  int // 评论作者在 userIDs 中的下标
	ParentOffset int // 相对当前主题已建评论起点的偏移；-1 表示顶层评论
	Body         string
}

// seedTopicPlan 是单个主题的完整生成计划：主题 + 它下面的评论。
type seedTopicPlan struct {
	Topic    seedTopic
	Comments []seedComment
}

// seedDataset 是一次 seed 运行的完整内存计划，由 generateSeedDataset 生成。
type seedDataset struct {
	Users  []seedUser
	Topics []seedTopicPlan
}

// 内置中文假数据语料。种子数据用于本地开发与测试，内容力求自然、多样、无意义纠纷。
var (
	seedTopicTitles = []string{
		"关于新版编辑器的使用体验反馈",
		"今天遇到了一个奇怪的白屏问题",
		"分享一个我最近在用的效率工具",
		"新手报到，请大家多多关照",
		"关于社区规则的一些建议",
		"周末线下聚会有人参加吗",
		"如何优雅地组织自己的知识库",
		"请教一个关于数据库索引的问题",
		"推荐几本最近读完的好书",
		"部署到生产环境时踩过的坑",
		"关于暗色模式的一些设计思考",
		"我的 2026 年技术学习计划",
		"讨论：静态语言 vs 动态语言",
		"这个报错有人见过吗？求帮忙看看",
		"分享一份我刚整理的面试题集",
		"新人如何快速融入开源项目",
		"关于积分系统和用户等级的想法",
		"今天升级后发帖功能好像有点问题",
		"如何评价最新的前端框架更新",
		"寻找一起做副业的伙伴",
		"关于广告和推广内容的处理建议",
		"我的家庭服务器搭建笔记",
		"推荐一个好用的 Markdown 编辑器",
		"请教大家如何保持长期学习动力",
		"关于评论区树形结构的交互建议",
		"分享一段我写的糟糕代码，求重构建议",
		"深夜emo：写代码写到怀疑人生",
		"这个功能建议官方考虑一下",
		"记录一次线上事故的复盘过程",
		"祝大家周末愉快",
	}

	// seedTopicBodies 是主题正文片段池，每次随机拼接 2~4 段，组合出多样的正文。
	seedTopicBodies = []string{
		"最近在折腾这个话题，踩了不少坑，想在这里和大家聊聊，也顺便记录一下自己的思路。",
		"如题，我在实际使用过程中发现了一些值得讨论的点，希望能听到更多人的看法。",
		"先说一下背景：我在做一个小项目，遇到了一些技术上和非技术上的问题。",
		"我觉得这个方向挺有意思的，所以开个帖子收集一下大家的经验，避免重复造轮子。",
		"下面我会分几个部分来写，水平有限，如果有不对的地方欢迎指出。",
		"希望能抛砖引玉，大家一起交流。如果你有相关的经验，欢迎在评论区分享。",
		"完整的内容比较多，我先挑重点说，细节可以看后续的讨论。",
		"我用了一段时间，整体感觉还不错，但也有几个不太顺手的地方，下面具体说说。",
		"这次更新之后体验有明显变化，我把自己的感受整理出来，供大家参考。",
		"如果你也遇到类似的问题，希望这个帖子能帮到你，也欢迎补充。",
	}

	seedCommentBodies = []string{
		"说得好，支持一下。",
		"我也遇到了同样的问题，蹲一个解决方案。",
		"感谢分享，收藏了。",
		"Mark，回头仔细看看。",
		"我觉得这个思路挺有意思的。",
		"补充一点：我在另一个场景下表现不太一样。",
		"同感，最近我也在纠结这个。",
		"学到了，感谢楼主。",
		"有不同的看法，不过尊重你的观点。",
		"这个方案我之前也试过，确实可行。",
		"帮顶，希望有大佬来解答。",
		"刚试了一下，确实复现了。",
		"建议加上一些具体的例子会更清楚。",
		"已赞，期待后续更新。",
		"楼主辛苦了，写得很详细。",
		"我的环境和你的略有不同，仅供参考。",
		"这个踩坑经验很实用，感谢记录。",
		"学习了，回头我也试试这个方法。",
	}
)

// randomHex 返回 n 字节的 crypto/rand 十六进制字符串。
// 用于用户名/邮箱后缀，避免唯一约束冲突；用 crypto 而非 math 是为了让
// 随机后缀在多次 seed 运行之间也几乎不可能撞库。
func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := cryptorand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// generateSeedUser 生成第 index 个假用户（index 仅用于显示名序号，不参与唯一性）。
// 唯一性由 6 字节随机后缀保证：username_lower / email_lower 都是大小写不敏感唯一的。
func generateSeedUser(index int) (seedUser, error) {
	suffix, err := randomHex(6)
	if err != nil {
		return seedUser{}, err
	}
	password, err := randomHex(8)
	if err != nil {
		return seedUser{}, err
	}
	return seedUser{
		Username:    "seed_" + suffix,
		Email:       "seed_" + suffix + "@seed.local",
		DisplayName: fmt.Sprintf("种子用户%d", index+1),
		// 种子用户的密码随机生成且不需要登录，但仍走完整 argon2 哈希以贴近真实数据。
		Password: "Seed-" + password,
	}, nil
}

// pickString 从池中随机取一条。
func pickString(rng *rand.Rand, pool []string) string {
	return pool[rng.IntN(len(pool))]
}

// generateTopicBody 随机拼接 2~4 段正文片段，组成一篇 Markdown 主题正文。
func generateTopicBody(rng *rand.Rand) string {
	paragraphCount := 2 + rng.IntN(3) // 2~4
	parts := make([]string, 0, paragraphCount+1)
	parts = append(parts, pickString(rng, seedTopicBodies))
	for i := 0; i < paragraphCount; i++ {
		parts = append(parts, pickString(rng, seedTopicBodies))
	}
	return strings.Join(parts, "\n\n")
}

// generateSeedDataset 生成一次 seed 运行的完整内存计划，不触碰数据库。
// 调用方在 dry-run 模式下可以直接打印它，在真实模式下按计划逐条写入。
func generateSeedDataset(opts seedOptions, rng *rand.Rand) (seedDataset, error) {
	users := make([]seedUser, 0, opts.Users)
	for i := 0; i < opts.Users; i++ {
		u, err := generateSeedUser(i)
		if err != nil {
			return seedDataset{}, err
		}
		users = append(users, u)
	}

	topics := make([]seedTopicPlan, 0, opts.Count)
	for i := 0; i < opts.Count; i++ {
		authorIndex := rng.IntN(opts.Users)
		title := pickString(rng, seedTopicTitles)
		body := generateTopicBody(rng)

		var comments []seedComment
		if opts.CommentsMax > 0 {
			// 每个主题的评论数随机，包含 0 条的可能，平均约为 CommentsMax 的一半。
			count := rng.IntN(opts.CommentsMax + 1)
			comments = make([]seedComment, 0, count)
			for j := 0; j < count; j++ {
				var parentOffset int
				if len(comments) > 0 && rng.IntN(2) == 0 {
					// 50% 概率回复本主题内已有的评论。
					parentOffset = rng.IntN(len(comments))
				} else {
					// 顶层评论。
					parentOffset = -1
				}
				comments = append(comments, seedComment{
					AuthorIndex:  rng.IntN(opts.Users),
					ParentOffset: parentOffset,
					Body:         pickString(rng, seedCommentBodies),
				})
			}
		}

		topics = append(topics, seedTopicPlan{
			Topic: seedTopic{
				AuthorIndex: authorIndex,
				Title:       title,
				Body:        body,
			},
			Comments: comments,
		})
	}

	return seedDataset{Users: users, Topics: topics}, nil
}

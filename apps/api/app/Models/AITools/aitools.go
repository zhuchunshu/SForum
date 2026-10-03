// Package aitools 实现 Core 内置的只读 AI 工具。
//
// 这些工具只做一件事：把「访客本来就能看到的站内内容」按需交给模型，并带上
// 可点击的链接。它们不写任何数据、不触碰审核状态、也不引入新的可见性规则——
// 每个工具都复用对应领域服务的公开读取路径，因此「机器人能看到什么」与
// 「访客能看到什么」由同一套代码决定。
//
// 工具的返回值会进入提示词并可能被复述，因此两条纪律贯穿全包：
//   - 只返回公开可见的内容，越权内容回答「不存在」而不是「无权限」；
//   - 输出按 rune 截断，单个结果保持在几千字节内，成本可预测。
package aitools

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	profile "github.com/zhuchunshu/sforum/apps/api/app/Models/Profile"
	supportai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
	search "github.com/zhuchunshu/sforum/apps/api/app/Support/Search"
)

// Searcher 是站内检索的最小面，由 Support/Search 的服务实现。
type Searcher interface {
	Search(ctx context.Context, input search.SearchInput) (search.SearchResult, error)
}

// TopicReader 读取公开主题详情，由 forum.Service 实现。
type TopicReader interface {
	GetTopic(ctx context.Context, topicID int64) (forum.TopicDetail, error)
}

// CommentReader 读取公开评论列表，由 forum.Service 实现。
type CommentReader interface {
	ListComments(ctx context.Context, input forum.CommentListInput) (forum.CommentList, error)
}

// TopicLister 读取公开主题列表，由 forum.Service 实现。
type TopicLister interface {
	ListTopics(ctx context.Context, input forum.TopicListInput) (forum.TopicList, error)
}

// ProfileReader 读取公开个人资料，由 profile.Service 实现。
type ProfileReader interface {
	GetPublicProfile(ctx context.Context, username string) (profile.PublicProfile, error)
}

// SiteLinks 把站内路径拼成链接。SiteURL 为空时退回相对路径：站内评论里的相对
// 链接同样可点，绝对地址则更适合被复制到站外。
type SiteLinks struct {
	SiteURL func(ctx context.Context) string
}

func (l SiteLinks) Topic(ctx context.Context, topicID int64) string {
	return l.base(ctx) + "/t/" + strconv.FormatInt(topicID, 10)
}

func (l SiteLinks) User(ctx context.Context, username string) string {
	name := strings.TrimSpace(username)
	if name == "" {
		return ""
	}
	return l.base(ctx) + "/u/" + url.PathEscape(name)
}

func (l SiteLinks) base(ctx context.Context) string {
	if l.SiteURL == nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(l.SiteURL(ctx)), "/")
}

// Deps 汇总内置工具的依赖。任一依赖缺失时对应工具不注册：工具不存在比
// 「工具存在但永远失败」更诚实，模型不会把一次调用浪费在必然失败的路径上。
type Deps struct {
	Search   Searcher
	Topics   TopicReader
	Comments CommentReader
	Lister   TopicLister
	Profiles ProfileReader
	SiteURL  func(ctx context.Context) string
	Clock    func() time.Time
}

// NewBuiltinRegistry 注册 Core 内置的五个只读工具。
func NewBuiltinRegistry(deps Deps) (*supportai.ToolRegistry, error) {
	registry := supportai.NewToolRegistry()
	links := SiteLinks{SiteURL: deps.SiteURL}
	register := func(tool supportai.Tool, enabled bool) error {
		if !enabled {
			return nil
		}
		return registry.Register(tool)
	}
	if err := register(NewSearchTool(deps.Search, links), deps.Search != nil); err != nil {
		return nil, err
	}
	if err := register(NewTopicReadTool(deps.Topics, deps.Comments, links), deps.Topics != nil && deps.Comments != nil); err != nil {
		return nil, err
	}
	if err := register(NewTopicListTool(deps.Lister, links), deps.Lister != nil); err != nil {
		return nil, err
	}
	if err := register(NewUserProfileTool(deps.Profiles, links), deps.Profiles != nil); err != nil {
		return nil, err
	}
	if err := register(NewTimeTool(deps.Clock), true); err != nil {
		return nil, err
	}
	return registry, nil
}

// decodeArguments 解析模型给出的参数串。参数不是合法 JSON 对象时返回给模型看的
// 说明，而不是让整次回复失败。
func decodeArguments(raw string, target any) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "{}"
	}
	if err := json.Unmarshal([]byte(trimmed), target); err != nil {
		return errors.New("参数无法解析：需要 JSON 对象，且字段类型要匹配")
	}
	return nil
}

// clip 按 rune 截断，避免在多字节字符中间切断。
func clip(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
}

// authorName 取公开显示名，缺省回落到用户名。
func authorName(username, displayName string) string {
	if name := strings.TrimSpace(displayName); name != "" {
		return name
	}
	return strings.TrimSpace(username)
}

// pageDefaults 归一化页码与每页条数：页码至少 1，每页条数落在 1..max 内。
func pageDefaults(page, perPage, fallback, max int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 {
		perPage = fallback
	}
	if perPage > max {
		perPage = max
	}
	return page, perPage
}

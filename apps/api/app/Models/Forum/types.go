package forum

import (
	"context"
	"errors"
	"time"
)

const (
	SourceFormatMarkdown = "markdown"
	SourceFormatHTML     = "html"
	SourceFormatJSON     = "json"

	EditorTypeMarkdown = "markdown"

	// v2 启用 goldmark GFM 扩展（表格/删除线/自动链接/任务列表）并放开对应 sanitizer 规则。
	// 存量帖子保留 v1 HTML，下次编辑时自然升级到 v2（不做批量重渲染）。
	RenderVersion = "goldmark-bluemonday-v2"

	TopicStatusActive  = "active"
	TopicStatusLocked  = "locked"
	TopicStatusHidden  = "hidden"
	TopicStatusDeleted = "deleted"

	CommentStatusActive  = "active"
	CommentStatusHidden  = "hidden"
	CommentStatusDeleted = "deleted"

	TagStatusActive   = "active"
	TagStatusPending  = "pending"
	TagStatusDisabled = "disabled"

	TagCreationModeControlled = "controlled"
	TagCreationModeReview     = "review"
	TagCreationModeOpen       = "open"

	// 主题生命周期动作。hide/restore 归版主（topic.delete_any），
	// lock/unlock 归 topic.lock，pin/unpin 归 topic.pin。
	TopicActionHide    = "hide"
	TopicActionRestore = "restore"
	TopicActionLock    = "lock"
	TopicActionUnlock  = "unlock"
	TopicActionPin     = "pin"
	TopicActionUnpin   = "unpin"

	CodeInvalidContent    = "forum.content_invalid"
	CodeInvalidTopic      = "forum.topic_invalid"
	CodeTopicNotFound     = "forum.topic_not_found"
	CodeCommentNotFound   = "forum.comment_not_found"
	CodeTopicClosed       = "forum.topic_closed"
	CodeInvalidTag        = "forum.tag_invalid"
	CodeTagNotFound       = "forum.tag_not_found"
	CodeInvalidSettings   = "forum.settings_invalid"
	CodeInvalidAction     = "forum.topic_action_invalid"
	CodeUseSearch         = "forum.use_search_endpoint"
	CodeReindexRunning    = "forum.reindex_running"    // 已有重建在进行
	CodeReindexNoRun      = "forum.reindex_no_run"     // 尚无重建记录
	CodeSearchUnavailable = "forum.search_unavailable" // 搜索服务不可用
)

var (
	ErrInvalidContent  = errors.New("forum: invalid content")
	ErrInvalidTopic    = errors.New("forum: invalid topic")
	ErrTopicNotFound   = errors.New("forum: topic not found")
	ErrCommentNotFound = errors.New("forum: comment not found")
	ErrTopicClosed     = errors.New("forum: topic is closed")
	ErrInvalidTag      = errors.New("forum: invalid tag")
	ErrTagNotFound     = errors.New("forum: tag not found")
	ErrInvalidSettings = errors.New("forum: invalid settings")
	ErrInvalidAction   = errors.New("forum: invalid topic action")
	// ErrUseSearchEndpoint 表示 topics 列表不再支持关键词检索，应改用专用搜索端点。
	ErrUseSearchEndpoint = errors.New("forum: use search endpoint")
)

// TopicSearchIndexer 是 forum 包对搜索索引调度的抽象。
// 由 search 支持包实现并注入 Service，避免 forum 反向依赖 job/meilisearch。
// 实现应异步（入队）且对调用方安全；nil 时 Service 自动降级为不索引。
type TopicSearchIndexer interface {
	// EnqueueIndex 调度重新索引指定主题（用于创建/更新/评论/恢复/置顶等）。
	EnqueueIndex(ctx context.Context, topicID int64) error
	// EnqueueDelete 调度从索引移除指定主题（用于删除/隐藏）。
	EnqueueDelete(ctx context.Context, topicID int64) error
}

type TopicExtensionActionProvider interface {
	TopicExtensionActions(ctx context.Context) ([]TopicExtensionAction, error)
}

type ContentInput struct {
	RawContent    string `json:"rawContent"`
	SourceFormat  string `json:"sourceFormat"`
	EditorType    string `json:"editorType"`
	EditorVersion string `json:"editorVersion"`
}

type RenderedContent struct {
	ID            int64  `json:"id"`
	RawContent    string `json:"rawContent"`
	HTMLContent   string `json:"htmlContent"`
	PlainText     string `json:"plainText"`
	Excerpt       string `json:"excerpt"`
	SourceFormat  string `json:"sourceFormat"`
	EditorType    string `json:"editorType"`
	EditorVersion string `json:"editorVersion"`
	RenderVersion string `json:"renderVersion"`
	ContentHash   string `json:"contentHash"`
}

type Category struct {
	ID           int64     `json:"id"`
	GroupID      int64     `json:"groupId"`
	GroupSlug    string    `json:"groupSlug"`
	GroupName    string    `json:"groupName"`
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Visibility   string    `json:"visibility"`
	Position     int       `json:"position"`
	DefaultSort  string    `json:"defaultSort"`
	TopicCount   int64     `json:"topicCount"`
	CommentCount int64     `json:"commentCount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type CategoryGroup struct {
	ID          int64      `json:"id"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Visibility  string     `json:"visibility"`
	Position    int        `json:"position"`
	Categories  []Category `json:"categories,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type Tag struct {
	ID          int64     `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	TopicCount  int64     `json:"topicCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type UserSummary struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type TopicListInput struct {
	Page         int
	PerPage      int
	CategorySlug string
	TagSlug      string
	Query        string
}

type TopicList struct {
	Items   []TopicSummary `json:"items"`
	Total   int64          `json:"total"`
	Page    int            `json:"page"`
	PerPage int            `json:"perPage"`
}

type TopicSummary struct {
	ID             int64             `json:"id"`
	CategoryID     int64             `json:"categoryId"`
	CategorySlug   string            `json:"categorySlug"`
	CategoryName   string            `json:"categoryName"`
	AuthorUserID   int64             `json:"authorUserId"`
	Author         *UserSummary      `json:"author,omitempty"`
	Title          string            `json:"title"`
	Slug           string            `json:"slug"`
	Status         string            `json:"status"`
	IsPinned       bool              `json:"isPinned"`
	CommentCount   int64             `json:"commentCount"`
	ViewCount      int64             `json:"viewCount"`
	Tags           []TopicTagSummary `json:"tags,omitempty"`
	Excerpt        string            `json:"excerpt"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
	LastActivityAt time.Time         `json:"lastActivityAt"`
}

type TopicDetail struct {
	TopicSummary
	Content          RenderedContent        `json:"content"`
	ExtensionActions []TopicExtensionAction `json:"extensionActions,omitempty"`
}

type TopicExtensionAction struct {
	ExtensionID string            `json:"extensionId"`
	ID          string            `json:"id"`
	Label       map[string]string `json:"label,omitempty"`
	Icon        string            `json:"icon,omitempty"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Confirm     bool              `json:"confirm,omitempty"`
}

type CreateTopicInput struct {
	CategorySlug string       `json:"categorySlug"`
	Title        string       `json:"title"`
	TagSlugs     []string     `json:"tagSlugs,omitempty"`
	Content      ContentInput `json:"content"`
}

type CreateTopicRecord struct {
	ID              int64
	CategoryID      int64
	CategorySlug    string
	AuthorUserID    int64
	Title           string
	Slug            string
	TagSlugs        []string
	TagCreationMode string
	Tags            []TopicTagSummary
	Content         RenderedContent
}

// UpdateTopicInput 是作者或版主更新主题时提交的输入。content 为可选：
// 不传则只更新标题/分类/标签，传则按 triple-storage 规则重新渲染并写入 post_revisions。
type UpdateTopicInput struct {
	TopicID      int64
	CategorySlug *string
	Title        *string
	TagSlugs     []string
	Content      *ContentInput
}

// UpdateTopicRecord 是 store 层更新主题的内部记录。content 为 nil 时表示不改正文。
type UpdateTopicRecord struct {
	TopicID         int64
	EditorUserID    int64
	CategorySlug    string
	Title           string
	Slug            string
	TagSlugs        []string
	TagCreationMode string
	HasContent      bool
	Content         RenderedContent
}

// TopicLifecycleInput 描述一次主题生命周期动作（hide/restore/lock/unlock/pin/unpin）。
type TopicLifecycleInput struct {
	TopicID int64
	Action  string
}

// TopicLifecycleRecord 是 store 层执行生命周期动作后的主题快照。
type TopicLifecycleRecord struct {
	TopicID  int64
	Status   string
	IsPinned bool
}

type TopicTagSummary struct {
	ID     int64  `json:"id"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ForumSettings struct {
	DefaultCategorySlug string `json:"defaultCategorySlug"`
	TagCreationMode     string `json:"tagCreationMode"`
	TagPublicPages      bool   `json:"tagPublicPages"`
	TagMaxPerTopic      int    `json:"tagMaxPerTopic"`
}

type CreateCategoryGroupInput struct {
	Slug        string
	Name        string
	Description string
	Visibility  string
	Position    int
}

type UpdateCategoryGroupInput struct {
	ID          int64
	Slug        *string
	Name        *string
	Description *string
	Visibility  *string
	Position    *int
}

type CreateCategoryInput struct {
	GroupID     int64
	Slug        string
	Name        string
	Description string
	Visibility  string
	Position    int
	DefaultSort string
}

type UpdateCategoryInput struct {
	ID          int64
	GroupID     *int64
	Slug        *string
	Name        *string
	Description *string
	Visibility  *string
	Position    *int
	DefaultSort *string
}

type CreateTagInput struct {
	Slug        string
	Name        string
	Description string
	Status      string
	ActorUserID int64
}

type UpdateTagInput struct {
	ID          int64
	Slug        *string
	Name        *string
	Description *string
	Status      *string
	ActorUserID int64
}

type UpdateForumSettingsInput struct {
	DefaultCategorySlug *string
	TagCreationMode     *string
	TagPublicPages      *bool
	TagMaxPerTopic      *int
}

type CommentListInput struct {
	TopicID int64
	View    string
	Page    int
	PerPage int
}

type CommentList struct {
	Items   []Comment `json:"items"`
	Total   int64     `json:"total"`
	Page    int       `json:"page"`
	PerPage int       `json:"perPage"`
	View    string    `json:"view"`
}

type Comment struct {
	ID            int64           `json:"id"`
	TopicID       int64           `json:"topicId"`
	AuthorUserID  int64           `json:"authorUserId"`
	Author        *UserSummary    `json:"author,omitempty"`
	ParentID      *int64          `json:"parentId,omitempty"`
	RootCommentID int64           `json:"rootCommentId"`
	PathKey       string          `json:"pathKey"`
	Depth         int             `json:"depth"`
	ReplyCount    int64           `json:"replyCount"`
	Status        string          `json:"status"`
	Content       RenderedContent `json:"content"`
	ReplyTo       *ReplyReference `json:"replyTo,omitempty"`
	Children      []Comment       `json:"children,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

type ReplyReference struct {
	ID      int64        `json:"id"`
	Author  *UserSummary `json:"author,omitempty"`
	Excerpt string       `json:"excerpt"`
	Depth   int          `json:"depth"`
}

type CommentSummary struct {
	ID            int64
	TopicID       int64
	AuthorUserID  int64
	ParentID      *int64
	RootCommentID int64
	PathKey       string
	Depth         int
	Status        string
}

type CreateCommentInput struct {
	TopicID  int64        `json:"topicId"`
	ParentID *int64       `json:"parentId,omitempty"`
	Content  ContentInput `json:"content"`
}

type CreateCommentRecord struct {
	ID           int64
	TopicID      int64
	AuthorUserID int64
	ParentID     *int64
	Parent       *CommentSummary
	Content      RenderedContent
}

type UpdateCommentInput struct {
	CommentID int64        `json:"commentId"`
	Content   ContentInput `json:"content"`
}

type UpdateCommentRecord struct {
	CommentID    int64
	EditorUserID int64
	Content      RenderedContent
}

type CommentPosition struct {
	RootCommentID int64
	PathKey       string
	Depth         int
}

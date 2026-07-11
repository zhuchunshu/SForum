package forum

import (
	"context"
	"errors"
	"time"

	avatar "github.com/zhuchunshu/sforum/apps/api/app/Support/Avatar"
)

const (
	SourceFormatMarkdown = "markdown"
	SourceFormatHTML     = "html"
	SourceFormatJSON     = "json"

	EditorTypeMarkdown = "markdown"

	// v2 启用 goldmark GFM 扩展（表格/删除线/自动链接/任务列表）并放开对应 sanitizer 规则。
	// 存量帖子保留 v1 HTML，下次编辑时自然升级到 v2（不做批量重渲染）。
	RenderVersion = "goldmark-bluemonday-v2"

	TopicStatusActive   = "active"
	TopicStatusLocked   = "locked"
	TopicStatusHidden   = "hidden"
	TopicStatusDeleted  = "deleted"
	TopicStatusPending  = "pending"
	TopicStatusRejected = "rejected"

	CommentStatusActive   = "active"
	CommentStatusHidden   = "hidden"
	CommentStatusDeleted  = "deleted"
	CommentStatusPending  = "pending"
	CommentStatusRejected = "rejected"

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
	CodeInvalidSettings      = "forum.settings_invalid"
	CodeInvalidAction        = "forum.topic_action_invalid"
	CodeUseSearch            = "forum.use_search_endpoint"
	CodeReindexRunning       = "forum.reindex_running"    // 已有重建在进行
	CodeReindexNoRun         = "forum.reindex_no_run"     // 尚无重建记录
	CodeSearchUnavailable    = "forum.search_unavailable" // 搜索服务不可用
	CodeTitleTooShort        = "forum.title_too_short"
	CodeTitleTooLong         = "forum.title_too_long"
	CodeContentTooShort      = "forum.content_too_short"
	CodeContentTooLong       = "forum.content_too_long"
	CodeCommentTooShort      = "forum.comment_too_short"
	CodeCommentTooLong       = "forum.comment_too_long"
	CodeCommentNestingDeep   = "forum.comment_nesting_too_deep"
	CodeEditWindowExpired    = "forum.edit_window_expired"
	CodeTopicCooldown         = "forum.topic_cooldown"
	CodeCommentCooldown       = "forum.comment_cooldown"
	CodeDailyTopicLimit      = "forum.daily_topic_limit"
	CodeDailyCommentLimit    = "forum.daily_comment_limit"
	CodeTagMinRequired       = "forum.tag_min_required"
)

var (
	ErrInvalidContent      = errors.New("forum: invalid content")
	ErrInvalidTopic        = errors.New("forum: invalid topic")
	ErrTopicNotFound       = errors.New("forum: topic not found")
	ErrCommentNotFound     = errors.New("forum: comment not found")
	ErrTopicClosed         = errors.New("forum: topic is closed")
	ErrInvalidTag          = errors.New("forum: invalid tag")
	ErrTagNotFound         = errors.New("forum: tag not found")
	ErrInvalidSettings     = errors.New("forum: invalid settings")
	ErrInvalidAction       = errors.New("forum: invalid topic action")
	ErrTitleTooShort       = errors.New("forum: title too short")
	ErrTitleTooLong        = errors.New("forum: title too long")
	ErrContentTooShort     = errors.New("forum: content too short")
	ErrContentTooLong      = errors.New("forum: content too long")
	ErrCommentTooShort     = errors.New("forum: comment too short")
	ErrCommentTooLong      = errors.New("forum: comment too long")
	ErrCommentNestingDeep  = errors.New("forum: comment nesting too deep")
	ErrEditWindowExpired   = errors.New("forum: edit window expired")
	ErrTopicCooldown        = errors.New("forum: topic cooldown")
	ErrCommentCooldown      = errors.New("forum: comment cooldown")
	ErrDailyTopicLimit     = errors.New("forum: daily topic limit")
	ErrDailyCommentLimit   = errors.New("forum: daily comment limit")
	ErrTagMinRequired      = errors.New("forum: tag minimum required")
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
	Icon         string    `json:"icon"`
	IconColor    string    `json:"iconColor"`
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
	Icon        string    `json:"icon"`
	IconColor   string    `json:"iconColor"`
	Status      string    `json:"status"`
	TopicCount  int64     `json:"topicCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type UserSummary struct {
	ID          int64       `json:"id"`
	Username    string      `json:"username"`
	DisplayName string      `json:"displayName"`
	Avatar      avatar.View `json:"avatar"`
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
	ID                 int64
	CategoryID         int64
	CategorySlug       string
	AuthorUserID       int64
	Title              string
	Slug               string
	TagSlugs           []string
	TagCreationMode    string
	Tags               []TopicTagSummary
	Content            RenderedContent
	Status             string
	ModerationTriggers []string
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
	TagMinPerTopic      int    `json:"tagMinPerTopic"`
	TagMaxPerTopic      int    `json:"tagMaxPerTopic"`
	TopicsPerPage       int    `json:"topicsPerPage"`
	CommentsPerPage     int    `json:"commentsPerPage"`

	// 主题内容与节奏限制（rune 计长；0 表示不限，除 min 外）
	TopicTitleMinRunes      int `json:"topicTitleMinRunes"`
	TopicTitleMaxRunes      int `json:"topicTitleMaxRunes"`
	TopicContentMinRunes    int `json:"topicContentMinRunes"`
	TopicContentMaxRunes    int `json:"topicContentMaxRunes"`
	TopicEditWindowMinutes  int `json:"topicEditWindowMinutes"`
	TopicCooldownSeconds     int `json:"topicCooldownSeconds"`
	DailyTopicLimit         int `json:"dailyTopicLimit"`

	// 评论内容、嵌套与节奏限制
	CommentMinRunes         int `json:"commentMinRunes"`
	CommentMaxRunes         int `json:"commentMaxRunes"`
	CommentMaxNestingDepth  int `json:"commentMaxNestingDepth"`
	CommentEditWindowMinutes int `json:"commentEditWindowMinutes"`
	CommentCooldownSeconds    int `json:"commentCooldownSeconds"`
	DailyCommentLimit       int `json:"dailyCommentLimit"`

	// 列表摘要截断长度（写入 posts.excerpt 时生效）
	ExcerptRuneLimit int `json:"excerptRuneLimit"`
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
	Icon        string
	IconColor   string
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
	Icon        *string
	IconColor   *string
	Visibility  *string
	Position    *int
	DefaultSort *string
}

type CreateTagInput struct {
	Slug        string
	Name        string
	Description string
	Icon        string
	IconColor   string
	Status      string
	ActorUserID int64
}

type UpdateTagInput struct {
	ID          int64
	Slug        *string
	Name        *string
	Description *string
	Icon        *string
	IconColor   *string
	Status      *string
	ActorUserID int64
}

type UpdateForumSettingsInput struct {
	DefaultCategorySlug *string
	TagCreationMode     *string
	TagPublicPages      *bool
	TagMinPerTopic      *int
	TagMaxPerTopic      *int
	TopicsPerPage       *int
	CommentsPerPage     *int

	TopicTitleMinRunes       *int
	TopicTitleMaxRunes       *int
	TopicContentMinRunes     *int
	TopicContentMaxRunes     *int
	TopicEditWindowMinutes   *int
	TopicCooldownSeconds      *int
	DailyTopicLimit          *int
	CommentMinRunes          *int
	CommentMaxRunes          *int
	CommentMaxNestingDepth   *int
	CommentEditWindowMinutes *int
	CommentCooldownSeconds     *int
	DailyCommentLimit        *int
	ExcerptRuneLimit         *int
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
	CreatedAt     time.Time
}

type CreateCommentInput struct {
	TopicID  int64        `json:"topicId"`
	ParentID *int64       `json:"parentId,omitempty"`
	Content  ContentInput `json:"content"`
}

type CreateCommentRecord struct {
	ID                 int64
	TopicID            int64
	AuthorUserID       int64
	ParentID           *int64
	Parent             *CommentSummary
	Content            RenderedContent
	Status             string
	ModerationTriggers []string
	MentionedUsernames []string
}

type PublicationInput struct {
	ActorUserID int64
	RawContent  string
}

type PublicationDecision struct {
	Pending  bool
	Triggers []string
}

type AuthorReviewItem struct {
	TargetType string    `json:"targetType"`
	TargetID   int64     `json:"targetId"`
	TopicID    int64     `json:"topicId,omitempty"`
	Title      string    `json:"title"`
	Excerpt    string    `json:"excerpt"`
	Status     string    `json:"status"`
	ReviewNote string    `json:"reviewNote"`
	CreatedAt  time.Time `json:"createdAt"`
}

type AuthorReviewList struct {
	Items []AuthorReviewItem `json:"items"`
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

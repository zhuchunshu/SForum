package forum

import (
	"errors"
	"time"
)

const (
	SourceFormatMarkdown = "markdown"
	SourceFormatHTML     = "html"
	SourceFormatJSON     = "json"

	EditorTypeMarkdown = "markdown"

	RenderVersion = "goldmark-bluemonday-v1"

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

	CodeInvalidContent  = "forum.content_invalid"
	CodeInvalidTopic    = "forum.topic_invalid"
	CodeTopicNotFound   = "forum.topic_not_found"
	CodeCommentNotFound = "forum.comment_not_found"
	CodeTopicClosed     = "forum.topic_closed"
	CodeInvalidTag      = "forum.tag_invalid"
	CodeTagNotFound     = "forum.tag_not_found"
	CodeInvalidSettings = "forum.settings_invalid"
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
)

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
	Content RenderedContent `json:"content"`
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

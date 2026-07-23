package profile

import (
	"context"
	"errors"
	"time"

	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	avatar "github.com/zhuchunshu/sforum/apps/api/app/Support/Avatar"
)

const (
	// 字段长度上限。避免过度校验，只做基本边界保护。
	maxBioLength       = 500
	maxSignatureLength = 200
	maxLocationLength  = 100
	maxWebsiteLength   = 200

	CodeProfileNotFound      = "profile.not_found"
	CodeProfileInvalid       = "profile.invalid"
	CodeAvatarUploadDisabled = "profile.avatar_upload_disabled"
)

var (
	ErrProfileNotFound      = errors.New("profile: not found")
	ErrProfileInvalid       = errors.New("profile: invalid profile input")
	ErrAvatarUploadDisabled = errors.New("profile: avatar upload disabled")
)

const (
	AvatarKindUploaded = "uploaded"
	AvatarKindInitials = "initials"
	AvatarKindGravatar = "gravatar"
	AvatarKindStatic   = "static"
)

// AvatarView 是前端和主题渲染头像所需的稳定视图。
type AvatarView = avatar.View

// Profile 是用户资料行（user_profiles）。
type Profile struct {
	UserID             int64      `json:"userId"`
	Bio                string     `json:"bio"`
	Signature          string     `json:"signature"`
	Location           string     `json:"location"`
	WebsiteURL         string     `json:"websiteUrl"`
	AvatarAttachmentID *int64     `json:"avatarAttachmentId,omitempty"`
	Avatar             AvatarView `json:"avatar"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// PublicProfile 是公开资料页聚合：用户摘要 + 资料 + 统计 + 近期公开主题。
type PublicProfile struct {
	UserID       int64                `json:"userId"`
	Username     string               `json:"username"`
	DisplayName  string               `json:"displayName"`
	Profile      Profile              `json:"profile"`
	TopicCount   int64                `json:"topicCount"`
	CommentCount int64                `json:"commentCount"`
	RecentTopics []forum.TopicSummary `json:"recentTopics"`
	Activities   []ProfileActivity    `json:"activities"`
	JoinedAt     time.Time            `json:"joinedAt"`
	// ExtensionTabs 来自 forum.profile.tabs 贡献（F4.3）；宿主渲染，无插件 HTML。
	ExtensionTabs []ProfileExtensionTab `json:"extensionTabs,omitempty"`
}

// ProfileActivity 是公开个人主页的主题/回复时间线条目。
type ProfileActivity struct {
	Kind      string               `json:"kind"`
	Topic     ProfileActivityTopic `json:"topic"`
	CommentID *int64               `json:"commentId,omitempty"`
	Excerpt   string               `json:"excerpt"`
	CreatedAt time.Time            `json:"createdAt"`
}

// ProfileActivityTopic 是活动条目链接所需的最小公开主题信息。
type ProfileActivityTopic struct {
	ID             int64     `json:"id"`
	Slug           string    `json:"slug"`
	Title          string    `json:"title"`
	Status         string    `json:"status"`
	CategorySlug   string    `json:"categorySlug"`
	CategoryName   string    `json:"categoryName"`
	CommentCount   int64     `json:"commentCount"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	LastActivityAt time.Time `json:"lastActivityAt"`
}

// ProfileCommentActivity 是 store 层回读的公开回复活动。
type ProfileCommentActivity struct {
	CommentID int64
	Topic     ProfileActivityTopic
	Excerpt   string
	CreatedAt time.Time
}

// ProfileExtensionTab 是公开资料页扩展 tab/section 描述符。
type ProfileExtensionTab struct {
	ExtensionID string            `json:"extensionId"`
	ID          string            `json:"id"`
	Order       int               `json:"order"`
	Label       map[string]string `json:"label,omitempty"`
	Icon        string            `json:"icon,omitempty"`
	// Kind: extensionRoute | hostLink
	Kind   string `json:"kind"`
	Method string `json:"method,omitempty"`
	// URL 对 extensionRoute 为 /extensions/{id}{path}；对 hostLink 为站内 href。
	URL string `json:"url"`
}

// ProfileTabProvider 解析 forum.profile.tabs。
type ProfileTabProvider interface {
	ProfileTabs(ctx context.Context) ([]ProfileExtensionTab, error)
}

// UpdateProfileInput 是当前用户更新自己资料的输入。所有指针字段为 nil 表示不改。
type UpdateProfileInput struct {
	Bio                *string
	Signature          *string
	Location           *string
	WebsiteURL         *string
	AvatarAttachmentID *int64
}

// UserProfileSummary 是 store 层返回的用户基本信息（来自 users 表）。
type UserProfileSummary struct {
	UserID      int64
	Username    string
	DisplayName string
	Email       string
	JoinedAt    time.Time
}

// ProfileStats 是用户的公开统计。
type ProfileStats struct {
	TopicCount   int64
	CommentCount int64
}

type AvatarAttachment = avatar.Attachment

// AvatarUser 是头像视图构建所需的最小用户信息。
type AvatarUser = avatar.User

// AvatarSource 是调用方已解析好的头像来源，避免列表页逐项回查 profile。
type AvatarSource = avatar.Source

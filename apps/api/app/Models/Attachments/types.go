package attachments

import (
	"errors"
	"time"

	storage "github.com/zhuchunshu/sforum/apps/api/app/Support/Storage"
)

const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"

	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusDeleted  = "deleted"

	CodeInvalidAttachment  = "attachment.invalid"
	CodeUploadDisabled     = "attachment.upload_disabled"
	CodeReferenced         = "attachment.referenced"
	CodeStorageUnavailable = "attachment.storage_unavailable"
)

var (
	ErrInvalidAttachment  = errors.New("attachments: invalid attachment")
	ErrUploadDisabled     = errors.New("attachments: upload disabled")
	ErrAttachmentNotFound = errors.New("attachments: not found")
	ErrReferenced         = errors.New("attachments: attachment is referenced")
	ErrStorageUnavailable = errors.New("attachments: storage unavailable")
)

type OwnerSummary struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type Attachment struct {
	ID             int64         `json:"id"`
	PublicID       string        `json:"publicId"`
	Owner          *OwnerSummary `json:"owner,omitempty"`
	Provider       string        `json:"provider"`
	ObjectKey      string        `json:"objectKey"`
	OriginalName   string        `json:"name"`
	ContentType    string        `json:"contentType"`
	Extension      string        `json:"extension"`
	SizeBytes      int64         `json:"size"`
	SHA256         string        `json:"sha256"`
	ImageWidth     *int          `json:"imageWidth,omitempty"`
	ImageHeight    *int          `json:"imageHeight,omitempty"`
	Visibility     string        `json:"visibility"`
	Status         string        `json:"status"`
	ReferenceCount int           `json:"referenceCount"`
	URL            string        `json:"url"`
	CreatedAt      time.Time     `json:"createdAt"`
	UpdatedAt      time.Time     `json:"updatedAt"`
	DeletedAt      *time.Time    `json:"deletedAt,omitempty"`
}

type AttachmentReference struct {
	ID           int64     `json:"id"`
	AttachmentID int64     `json:"attachmentId"`
	ResourceType string    `json:"resourceType"`
	ResourceID   int64     `json:"resourceId"`
	Context      string    `json:"context"`
	CreatedAt    time.Time `json:"createdAt"`
}

// ReferenceAccess 是附件引用对应内容资源的授权快照。
// TopicStatus 对 comment/post 表示其所属主题状态。
type ReferenceAccess struct {
	AttachmentReference
	AuthorUserID       int64
	ResourceStatus     string
	TopicStatus        string
	CategoryVisibility string
	Exists             bool
}

// ReadGuardSubject is the authoritative resource snapshot used before a
// trusted replacement handles an attachment read route.
type ReadGuardSubject struct {
	PublicID    string
	OwnerUserID int64
	Status      string
	Visibility  string
	Exists      bool
	References  []ReferenceAccess
}

type AttachmentDetail struct {
	Attachment
	References []AttachmentReference `json:"references"`
}

type AttachmentListInput struct {
	Page            int
	PerPage         int
	Query           string
	Provider        string
	Status          string
	ContentType     string
	OwnerUserID     int64
	ReferenceStatus string
	CreatedFrom     time.Time
	CreatedTo       time.Time
}

type AttachmentList struct {
	Items   []Attachment `json:"items"`
	Total   int64        `json:"total"`
	Page    int          `json:"page"`
	PerPage int          `json:"perPage"`
}

type CreateAttachmentInput struct {
	PublicID     string
	OwnerUserID  int64
	Provider     string
	ObjectKey    string
	OriginalName string
	ContentType  string
	Extension    string
	SizeBytes    int64
	SHA256       string
	ImageWidth   *int
	ImageHeight  *int
	Visibility   string
}

type UploadInput struct {
	OriginalName string
	ContentType  string
	SizeBytes    int64
	File         ReadSeekCloser
}

type ReadSeekCloser interface {
	Read([]byte) (int, error)
	Seek(offset int64, whence int) (int64, error)
	Close() error
}

type AttachmentSettings struct {
	// ProviderSlot 是宿主契约名 attachment.storage.provider（F3.5 / E6）。
	ProviderSlot string `json:"providerSlot"`
	// Drivers 列出 core 内置驱动 id（兼容字段；完整列表见 Candidates）。
	Drivers []string `json:"drivers"`
	// Candidates 为 core 驱动 + 已启用且声明槽位的插件（E6.1）。
	Candidates []storage.Candidate `json:"candidates"`
	// Provider 为 attachment.provider：core 驱动 id 或 plugin:<extensionId>。
	Provider               string        `json:"provider"`
	UploadEnabled          bool          `json:"uploadEnabled"`
	PathTemplate           string        `json:"pathTemplate"`
	PublicBaseURL          string        `json:"publicBaseUrl"`
	MaxFileSizeMB          int           `json:"maxFileSizeMb"`
	AllowedExtensions      []string      `json:"allowedExtensions"`
	AllowedMIMETypes       []string      `json:"allowedMimeTypes"`
	DefaultVisibility      string        `json:"defaultVisibility"`
	CleanupOrphanAfterDays int           `json:"cleanupOrphanAfterDays"`
	Local                  LocalSettings `json:"local"`
}

type LocalSettings struct {
	Root         string `json:"root"`
	PublicPrefix string `json:"publicPrefix"`
}

type CleanupResult struct {
	Deleted int `json:"deleted"`
	Failed  int `json:"failed"`
}

type ProbeResult struct {
	Provider string `json:"provider"`
	OK       bool   `json:"ok"`
	Message  string `json:"message"`
	// Reason 稳定机器码（插件 RPC reason 或 attachment.storage_unavailable），便于运营与 i18n。
	Reason string `json:"reason,omitempty"`
}

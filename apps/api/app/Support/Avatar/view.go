package avatar

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	KindUploaded = "uploaded"
	KindInitials = "initials"
	KindGravatar = "gravatar"
	KindStatic   = "static"

	// 与 Attachments.StatusActive 保持同值，但不导入 Attachments，避免 Avatar 成为模型包循环依赖入口。
	attachmentStatusActive = "active"

	ProviderInitials = "initials"
	ProviderGravatar = "gravatar"
	ProviderStatic   = "static"

	HashMD5    = "md5"
	HashSHA256 = "sha256"
)

type View struct {
	Kind         string `json:"kind"`
	URL          string `json:"url"`
	AttachmentID *int64 `json:"attachmentId,omitempty"`
	Alt          string `json:"alt"`
}

type User struct {
	UserID      int64
	Username    string
	DisplayName string
	Email       string
}

type Attachment struct {
	ID          int64
	PublicID    string
	OwnerUserID int64
	ContentType string
	Status      string
	URL         string
}

type Source struct {
	AttachmentID *int64
	Attachment   *Attachment
}

type Options struct {
	AllowUpload           bool
	DefaultProvider       string
	GravatarBaseURL       string
	GravatarHashAlgorithm string
	DefaultStaticURL      string
	MaxSizeKB             int
	MaxDimension          int
	AllowGIF              bool
	CompressEnabled       bool
	TargetDimension       int
	CompressQuality       int
}

type OptionResolver interface {
	AvatarOptions(ctx context.Context) (Options, error)
}

type ViewBuilder struct {
	avatarOptions OptionResolver
}

func NewViewBuilder(avatarOptions OptionResolver) *ViewBuilder {
	return &ViewBuilder{avatarOptions: avatarOptions}
}

func (b *ViewBuilder) AvatarView(ctx context.Context, user User, source Source) View {
	alt := Alt(user)
	if source.AttachmentID != nil && *source.AttachmentID > 0 && source.Attachment != nil && source.Attachment.Status == attachmentStatusActive {
		id := source.Attachment.ID
		url := strings.TrimSpace(source.Attachment.URL)
		if url == "" && source.Attachment.PublicID != "" {
			url = "/api/v1/attachments/" + source.Attachment.PublicID + "/content"
		}
		return View{Kind: KindUploaded, URL: url, AttachmentID: &id, Alt: alt}
	}

	avatarOptions, err := b.resolveAvatarOptions(ctx)
	if err != nil {
		return View{Kind: KindInitials, Alt: alt}
	}
	switch avatarOptions.DefaultProvider {
	case ProviderGravatar:
		if hash := GravatarHash(user.Email, avatarOptions.GravatarHashAlgorithm); hash != "" {
			return View{Kind: KindGravatar, URL: avatarOptions.GravatarBaseURL + hash, Alt: alt}
		}
	case ProviderStatic:
		if url := strings.TrimSpace(avatarOptions.DefaultStaticURL); url != "" {
			return View{Kind: KindStatic, URL: url, Alt: alt}
		}
	}
	return View{Kind: KindInitials, Alt: alt}
}

func (b *ViewBuilder) resolveAvatarOptions(ctx context.Context) (Options, error) {
	if b == nil || b.avatarOptions == nil {
		return DefaultOptions(), nil
	}
	return b.avatarOptions.AvatarOptions(ctx)
}

func DefaultOptions() Options {
	return Options{
		AllowUpload:           true,
		DefaultProvider:       ProviderInitials,
		GravatarBaseURL:       "https://gravatar.com/avatar/",
		GravatarHashAlgorithm: HashSHA256,
		MaxSizeKB:             2048,
		MaxDimension:          2048,
		AllowGIF:              false,
		CompressEnabled:       true,
		TargetDimension:       256,
		CompressQuality:       85,
	}
}

func Alt(user User) string {
	if value := strings.TrimSpace(user.DisplayName); value != "" {
		return value
	}
	if value := strings.TrimSpace(user.Username); value != "" {
		return value
	}
	return "User"
}

func GravatarHash(email string, algorithm string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return ""
	}
	if algorithm == HashMD5 {
		sum := md5.Sum([]byte(normalized))
		return hex.EncodeToString(sum[:])
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

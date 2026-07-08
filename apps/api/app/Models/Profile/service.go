package profile

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
)

type Service struct {
	store          Store
	avatarUploader AvatarUploader
	avatarOptions  AvatarOptionResolver
}

type AvatarUploader interface {
	UploadAvatar(ctx context.Context, actor identity.Actor, input attachments.UploadInput) (attachments.Attachment, error)
}

type AvatarOptionResolver interface {
	AvatarOptions(ctx context.Context) (options.AvatarOptions, error)
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func NewServiceWithAvatar(store Store, uploader AvatarUploader, avatarOptions AvatarOptionResolver) *Service {
	return &Service{store: store, avatarUploader: uploader, avatarOptions: avatarOptions}
}

// GetPublicProfile 返回公开资料页聚合数据。
func (s *Service) GetPublicProfile(ctx context.Context, username string) (PublicProfile, error) {
	normalized := strings.TrimSpace(username)
	if normalized == "" {
		return PublicProfile{}, ErrProfileNotFound
	}
	user, err := s.store.GetUserSummaryByUsername(ctx, normalized)
	if err != nil {
		return PublicProfile{}, err
	}
	profile, err := s.store.GetProfile(ctx, user.UserID)
	if err != nil {
		return PublicProfile{}, err
	}
	stats, err := s.store.GetProfileStats(ctx, user.UserID)
	if err != nil {
		return PublicProfile{}, err
	}
	recent, err := s.store.ListRecentTopics(ctx, user.UserID, 5)
	if err != nil {
		return PublicProfile{}, err
	}
	profile = s.decorateProfile(ctx, user, profile)
	return PublicProfile{
		UserID:       user.UserID,
		Username:     user.Username,
		DisplayName:  user.DisplayName,
		Profile:      profile,
		TopicCount:   stats.TopicCount,
		CommentCount: stats.CommentCount,
		RecentTopics: recent,
		JoinedAt:     user.JoinedAt,
	}, nil
}

// GetMyProfile 读取当前用户可编辑的资料。
func (s *Service) GetMyProfile(ctx context.Context, userID int64) (PublicProfile, error) {
	if userID <= 0 {
		return PublicProfile{}, identity.ErrPermissionDenied
	}
	user, err := s.store.GetUserSummaryByID(ctx, userID)
	if err != nil {
		return PublicProfile{}, err
	}
	profile, err := s.store.GetProfile(ctx, userID)
	if err != nil {
		return PublicProfile{}, err
	}
	profile = s.decorateProfile(ctx, user, profile)
	return PublicProfile{
		UserID:      user.UserID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Profile:     profile,
		JoinedAt:    user.JoinedAt,
	}, nil
}

// UpdateMyProfile 更新当前用户自己的资料。仅限当前用户操作自己。
func (s *Service) UpdateMyProfile(ctx context.Context, actor identity.Actor, input UpdateProfileInput) (Profile, error) {
	if actor.ID <= 0 {
		return Profile{}, identity.ErrPermissionDenied
	}
	normalized, err := normalizeUpdateProfileInput(input)
	if err != nil {
		return Profile{}, err
	}
	existing, err := s.store.GetProfile(ctx, actor.ID)
	if err != nil {
		return Profile{}, err
	}
	// 合并：nil 字段保留原值。
	merged := Profile{
		UserID:             actor.ID,
		Bio:                existing.Bio,
		Signature:          existing.Signature,
		Location:           existing.Location,
		WebsiteURL:         existing.WebsiteURL,
		AvatarAttachmentID: existing.AvatarAttachmentID,
		CreatedAt:          existing.CreatedAt,
	}
	if normalized.Bio != nil {
		merged.Bio = *normalized.Bio
	}
	if normalized.Signature != nil {
		merged.Signature = *normalized.Signature
	}
	if normalized.Location != nil {
		merged.Location = *normalized.Location
	}
	if normalized.WebsiteURL != nil {
		merged.WebsiteURL = *normalized.WebsiteURL
	}
	updated, err := s.store.UpsertProfile(ctx, merged)
	if err != nil {
		return Profile{}, err
	}
	if normalized.AvatarAttachmentID != nil {
		updated, err = s.store.SetAvatarAttachment(ctx, actor.ID, normalized.AvatarAttachmentID, actor.ID)
		if err != nil {
			return Profile{}, err
		}
	}
	user, err := s.store.GetUserSummaryByID(ctx, actor.ID)
	if err != nil {
		return updated, nil
	}
	return s.decorateProfile(ctx, user, updated), nil
}

func (s *Service) UploadAvatar(ctx context.Context, actor identity.Actor, input attachments.UploadInput) (Profile, error) {
	if actor.ID <= 0 || !actor.IsActive() {
		return Profile{}, identity.ErrPermissionDenied
	}
	if !actor.Can(identity.PermissionAttachmentUpload) {
		return Profile{}, identity.ErrPermissionDenied
	}
	avatarOptions, err := s.resolveAvatarOptions(ctx)
	if err != nil {
		return Profile{}, err
	}
	if !avatarOptions.AllowUpload {
		return Profile{}, ErrAvatarUploadDisabled
	}
	if s.avatarUploader == nil {
		return Profile{}, ErrAvatarUploadDisabled
	}
	attachment, err := s.avatarUploader.UploadAvatar(ctx, actor, input)
	if err != nil {
		return Profile{}, err
	}
	updated, err := s.store.SetAvatarAttachment(ctx, actor.ID, &attachment.ID, actor.ID)
	if err != nil {
		return Profile{}, err
	}
	user, err := s.store.GetUserSummaryByID(ctx, actor.ID)
	if err != nil {
		return updated, nil
	}
	return s.decorateProfile(ctx, user, updated), nil
}

func (s *Service) DeleteAvatar(ctx context.Context, actor identity.Actor) (Profile, error) {
	if actor.ID <= 0 || !actor.IsActive() {
		return Profile{}, identity.ErrPermissionDenied
	}
	updated, err := s.store.SetAvatarAttachment(ctx, actor.ID, nil, actor.ID)
	if err != nil {
		return Profile{}, err
	}
	user, err := s.store.GetUserSummaryByID(ctx, actor.ID)
	if err != nil {
		return updated, nil
	}
	return s.decorateProfile(ctx, user, updated), nil
}

func (s *Service) decorateProfile(ctx context.Context, user UserProfileSummary, profile Profile) Profile {
	profile.Avatar = s.avatarView(ctx, user, profile)
	return profile
}

func (s *Service) avatarView(ctx context.Context, user UserProfileSummary, profile Profile) AvatarView {
	alt := avatarAlt(user)
	if profile.AvatarAttachmentID != nil && *profile.AvatarAttachmentID > 0 {
		if attachment, err := s.store.GetAvatarAttachment(ctx, *profile.AvatarAttachmentID); err == nil && attachment.Status == attachments.StatusActive {
			id := attachment.ID
			url := strings.TrimSpace(attachment.URL)
			if url == "" && attachment.PublicID != "" {
				url = "/api/v1/attachments/" + attachment.PublicID + "/content"
			}
			return AvatarView{Kind: AvatarKindUploaded, URL: url, AttachmentID: &id, Alt: alt}
		}
	}

	avatarOptions, err := s.resolveAvatarOptions(ctx)
	if err != nil {
		return AvatarView{Kind: AvatarKindInitials, Alt: alt}
	}
	switch avatarOptions.DefaultProvider {
	case options.AvatarProviderGravatar:
		if hash := gravatarHash(user.Email, avatarOptions.GravatarHashAlgorithm); hash != "" {
			return AvatarView{Kind: AvatarKindGravatar, URL: avatarOptions.GravatarBaseURL + hash, Alt: alt}
		}
	case options.AvatarProviderStatic:
		if url := strings.TrimSpace(avatarOptions.DefaultStaticURL); url != "" {
			return AvatarView{Kind: AvatarKindStatic, URL: url, Alt: alt}
		}
	}
	return AvatarView{Kind: AvatarKindInitials, Alt: alt}
}

func (s *Service) resolveAvatarOptions(ctx context.Context) (options.AvatarOptions, error) {
	if s.avatarOptions == nil {
		return options.AvatarOptions{
			AllowUpload:           true,
			DefaultProvider:       options.AvatarProviderInitials,
			GravatarBaseURL:       "https://gravatar.com/avatar/",
			GravatarHashAlgorithm: options.AvatarHashSHA256,
			MaxSizeKB:             2048,
			MaxDimension:          2048,
			AllowGIF:              false,
			CompressEnabled:       true,
			TargetDimension:       256,
			CompressQuality:       85,
		}, nil
	}
	return s.avatarOptions.AvatarOptions(ctx)
}

func avatarAlt(user UserProfileSummary) string {
	if value := strings.TrimSpace(user.DisplayName); value != "" {
		return value
	}
	if value := strings.TrimSpace(user.Username); value != "" {
		return value
	}
	return "User"
}

func gravatarHash(email string, algorithm string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return ""
	}
	if algorithm == options.AvatarHashMD5 {
		sum := md5.Sum([]byte(normalized))
		return hex.EncodeToString(sum[:])
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// normalizeUpdateProfileInput 规范化并校验输入字段。
func normalizeUpdateProfileInput(input UpdateProfileInput) (NormalizeProfile, error) {
	result := NormalizeProfile{
		Bio:                trimPtr(input.Bio, maxBioLength),
		Signature:          trimPtr(input.Signature, maxSignatureLength),
		Location:           trimPtr(input.Location, maxLocationLength),
		WebsiteURL:         trimPtr(input.WebsiteURL, maxWebsiteLength),
		AvatarAttachmentID: input.AvatarAttachmentID,
	}
	if result.Bio != nil && len(*result.Bio) > maxBioLength {
		return NormalizeProfile{}, ErrProfileInvalid
	}
	if result.Signature != nil && len(*result.Signature) > maxSignatureLength {
		return NormalizeProfile{}, ErrProfileInvalid
	}
	if result.Location != nil && len(*result.Location) > maxLocationLength {
		return NormalizeProfile{}, ErrProfileInvalid
	}
	if result.WebsiteURL != nil {
		url := *result.WebsiteURL
		if url != "" && !isValidWebsiteURL(url) {
			return NormalizeProfile{}, ErrProfileInvalid
		}
	}
	if result.AvatarAttachmentID != nil && *result.AvatarAttachmentID <= 0 {
		return NormalizeProfile{}, ErrProfileInvalid
	}
	return result, nil
}

// trimPtr 裁剪空白并校验长度上限；nil 保持 nil。
func trimPtr(value *string, _ int) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

// isValidWebsiteURL 做最简单的 URL 形态校验，避免过度校验。
func isValidWebsiteURL(value string) bool {
	if len(value) > maxWebsiteLength {
		return false
	}
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// 保留 forum 包引用，避免未来扩展公开资料时的引用丢失（近期主题等依赖 forum 类型）。
var _ = forum.TopicSummary{}

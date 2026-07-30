package attachments

import (
	"context"
	"errors"
	"strings"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
)

// CodeGuestLoginRequired 与论坛游客阅读策略对齐：匿名访问受保护附件内容。
const CodeGuestLoginRequired = "forum.guest_login_required"

// ErrGuestLoginRequired 论坛 guest.read=login_required 时匿名访问帖子媒体。
var ErrGuestLoginRequired = errors.New("attachments: guest login required")

// 显式站点公开用途：头像、SEO/品牌图；不受 forum.guest.read 限制。
const (
	ResourceTypeUser = "user"
	ResourceTypeSEO  = "seo"
	ResourceTypeSite = "site"

	ContextAvatar         = "avatar"
	ContextLogo           = "logo"
	ContextFavicon        = "favicon"
	ContextAppleTouchIcon = "apple-touch-icon"
)

// isSitePublicReference 是否为应始终匿名可读的站点资产引用。
func isSitePublicReference(ref AttachmentReference) bool {
	switch strings.TrimSpace(ref.ResourceType) {
	case ResourceTypeSEO:
		return true
	case ResourceTypeUser:
		return strings.TrimSpace(ref.Context) == ContextAvatar
	case ResourceTypeSite:
		ctx := strings.TrimSpace(ref.Context)
		return ctx == ContextLogo || ctx == ContextFavicon || ctx == ContextAppleTouchIcon || ctx == "brand" || ctx == "chrome"
	default:
		return false
	}
}

func isForumReference(resourceType string) bool {
	switch strings.TrimSpace(resourceType) {
	case "topic", "comment", "post":
		return true
	default:
		return false
	}
}

func canModerateAttachmentReference(actor identity.Actor) bool {
	return actor.Can(identity.PermissionAttachmentManage) || actor.Can(identity.PermissionModerationReview)
}

func canViewForumReference(actor identity.Actor, ref ReferenceAccess) bool {
	if !ref.Exists || ref.CategoryVisibility != "public" {
		return canModerateAttachmentReference(actor)
	}
	if ref.TopicStatus != "active" && ref.TopicStatus != "locked" {
		if ref.TopicStatus == "pending" && actor.ID > 0 && actor.ID == ref.AuthorUserID {
			return true
		}
		return canModerateAttachmentReference(actor)
	}
	switch ref.ResourceStatus {
	case "active", "locked":
		return true
	case "pending":
		return (actor.ID > 0 && actor.ID == ref.AuthorUserID) || canModerateAttachmentReference(actor)
	default:
		return canModerateAttachmentReference(actor)
	}
}

func (s *Service) guestReadLoginRequired(ctx context.Context) bool {
	if s == nil || s.options == nil {
		return false
	}
	value, err := s.options.WebOption(ctx, options.NameForumGuestRead)
	if err != nil {
		// 读配置失败：保守按 login_required，避免误公开。
		return true
	}
	return strings.TrimSpace(value) == "login_required"
}

// authorizeAttachmentView 在 canViewAttachment 之上叠加论坛游客读策略。
func (s *Service) authorizeAttachmentView(ctx context.Context, actor identity.Actor, attachment Attachment) error {
	if !canViewAttachment(actor, attachment) {
		return identity.ErrPermissionDenied
	}
	if actor.Can(identity.PermissionAttachmentManage) ||
		attachment.Status != StatusActive || attachment.Visibility != VisibilityPublic {
		return nil
	}
	refs, err := s.store.ListReferenceAccess(ctx, attachment.ID)
	if err != nil {
		return identity.ErrPermissionDenied
	}
	ownerUserID := int64(0)
	if attachment.Owner != nil {
		ownerUserID = attachment.Owner.ID
	}
	return AuthorizeReadGuardSubject(actor, ReadGuardSubject{
		PublicID: attachment.PublicID, OwnerUserID: ownerUserID,
		Status: attachment.Status, Visibility: attachment.Visibility,
		Exists: true, References: refs,
	}, s.guestReadLoginRequired(ctx))
}

// AuthorizeReadGuardSubject keeps core and trusted replacement reads on the
// same attachment/reference policy without exposing storage internals.
func AuthorizeReadGuardSubject(actor identity.Actor, subject ReadGuardSubject, guestLoginRequired bool) error {
	if !subject.Exists || strings.TrimSpace(subject.PublicID) == "" {
		return ErrAttachmentNotFound
	}
	if actor.Can(identity.PermissionAttachmentManage) {
		return nil
	}
	if subject.Status != StatusActive {
		return identity.ErrPermissionDenied
	}
	if subject.Visibility != VisibilityPublic {
		if actor.IsActive() && subject.OwnerUserID > 0 && actor.ID == subject.OwnerUserID {
			return nil
		}
		return identity.ErrPermissionDenied
	}
	if len(subject.References) == 0 {
		if actor.IsActive() && subject.OwnerUserID > 0 && actor.ID == subject.OwnerUserID {
			return nil
		}
		return identity.ErrPermissionDenied
	}
	for _, ref := range subject.References {
		if isSitePublicReference(ref.AttachmentReference) {
			return nil
		}
		if !isForumReference(ref.ResourceType) || !canViewForumReference(actor, ref) {
			continue
		}
		if !actor.IsActive() && guestLoginRequired {
			return ErrGuestLoginRequired
		}
		return nil
	}
	return identity.ErrPermissionDenied
}

// contentURLPath API 内容代理路径（可执行会话策略）。
func contentURLPath(publicID string) string {
	return "/api/v1/attachments/" + publicID + "/content"
}

func displayVariantURLPath(publicID string) string {
	return "/api/v1/attachments/" + publicID + "/variants/" + CompressionVariantDisplay + "/content"
}

// shouldProxyAuthorizedURL 只有站点公开资产可返回永久 URL；论坛与未引用媒体始终走代理。
func (s *Service) shouldProxyAuthorizedURL(ctx context.Context, attachment Attachment) bool {
	refs, err := s.store.ListReferenceAccess(ctx, attachment.ID)
	if err != nil {
		return true
	}
	for _, ref := range refs {
		if isSitePublicReference(ref.AttachmentReference) {
			return false
		}
	}
	return true
}

func (s *Service) decorateURL(ctx context.Context, attachment Attachment) Attachment {
	if s.shouldProxyAuthorizedURL(ctx, attachment) {
		if attachment.ContentType == "image/jpeg" || attachment.ContentType == "image/png" {
			attachment.URL = displayVariantURLPath(attachment.PublicID)
		} else {
			attachment.URL = contentURLPath(attachment.PublicID)
		}
		return attachment
	}
	settings, err := s.runtimeSettings(ctx)
	if err != nil {
		attachment.URL = contentURLPath(attachment.PublicID)
		return attachment
	}
	adapter, err := s.adapterForSettings(ctx, settings, attachment.Provider)
	if err == nil {
		// 远程 provider 若仅有永久公网 URL，在需授权时仍回退代理（上面已处理）。
		// 此处 public 模式可直接用 PublicURL；有 SignedURL 能力时优先短时签名。
		if attachment.Visibility == VisibilityPrivate {
			if signed, signErr := adapter.SignedURL(ctx, attachment.ObjectKey, defaultSignedURLTTL); signErr == nil && signed != "" {
				attachment.URL = signed
			}
		} else {
			attachment.URL = adapter.PublicURL(attachment.ObjectKey)
		}
	}
	if attachment.URL == "" {
		attachment.URL = contentURLPath(attachment.PublicID)
	}
	return attachment
}

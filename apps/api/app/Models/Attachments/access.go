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

	ContextAvatar  = "avatar"
	ContextLogo    = "logo"
	ContextFavicon = "favicon"
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
		return ctx == ContextLogo || ctx == ContextFavicon || ctx == "brand" || ctx == "chrome"
	default:
		return false
	}
}

// requiresForumReadGate 帖子/未分类媒体在 login_required 模式下需登录。
// 仅当全部引用均为站点公开用途时才跳过论坛门禁；无引用时 fail closed。
func requiresForumReadGate(refs []AttachmentReference) bool {
	if len(refs) == 0 {
		return true
	}
	for _, ref := range refs {
		if !isSitePublicReference(ref) {
			return true
		}
	}
	return false
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
	// 管理权限或私有所有者路径已由 canViewAttachment 处理；
	// 仅对「公开可见」附件再套 guest.read。
	if attachment.Status != StatusActive || attachment.Visibility != VisibilityPublic {
		return nil
	}
	if actor.IsActive() {
		// 已登录活跃用户与论坛公开读接口一致，始终可看公开附件。
		return nil
	}
	if !s.guestReadLoginRequired(ctx) {
		return nil
	}
	refs, err := s.store.ListReferences(ctx, attachment.ID)
	if err != nil {
		// 引用解析失败：fail closed。
		return ErrGuestLoginRequired
	}
	if requiresForumReadGate(refs) {
		return ErrGuestLoginRequired
	}
	return nil
}

// contentURLPath API 内容代理路径（可执行会话策略）。
func contentURLPath(publicID string) string {
	return "/api/v1/attachments/" + publicID + "/content"
}

// shouldProxyAuthorizedURL login_required 下论坛媒体不得返回永久 CDN 公网 URL。
func (s *Service) shouldProxyAuthorizedURL(ctx context.Context, attachment Attachment) bool {
	if attachment.Status != StatusActive || attachment.Visibility != VisibilityPublic {
		return false
	}
	if !s.guestReadLoginRequired(ctx) {
		return false
	}
	refs, err := s.store.ListReferences(ctx, attachment.ID)
	if err != nil {
		return true
	}
	return requiresForumReadGate(refs)
}

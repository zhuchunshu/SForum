package sitechrome

import (
	"context"
	"net/url"
	"strings"
	"unicode/utf8"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

const (
	labelMaxRunes       = 40
	hrefMaxRunes        = 500
	nameMaxRunes        = 80
	descriptionMaxRunes = 200
	titleMaxRunes       = 120
	bodyMaxRunes        = 2000
	positionMin         = -100000
	positionMax         = 100000
)

var announcementStyles = []string{StyleInfo, StyleSuccess, StyleWarning, StyleDanger}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) canManage(actor identity.Actor) bool {
	// 品牌壳与站点设置同属运营配置；兼容旧 settings.manage 父权限。
	return actor.Can(identity.PermissionSettingsSiteManage)
}

// --- Nav ---

func (s *Service) ListPublicNavItems(ctx context.Context) ([]NavItem, error) {
	return s.store.ListNavItems(ctx, true)
}

func (s *Service) ListAdminNavItems(ctx context.Context, actor identity.Actor) ([]NavItem, error) {
	if !s.canManage(actor) {
		return nil, identity.ErrPermissionDenied
	}
	return s.store.ListNavItems(ctx, false)
}

func (s *Service) CreateNavItem(ctx context.Context, actor identity.Actor, input CreateNavItemInput) (NavItem, error) {
	if !s.canManage(actor) {
		return NavItem{}, identity.ErrPermissionDenied
	}
	normalized, err := normalizeCreateNavItem(input)
	if err != nil {
		return NavItem{}, err
	}
	return s.store.CreateNavItem(ctx, normalized)
}

func (s *Service) UpdateNavItem(ctx context.Context, actor identity.Actor, input UpdateNavItemInput) (NavItem, error) {
	if !s.canManage(actor) {
		return NavItem{}, identity.ErrPermissionDenied
	}
	if input.ID <= 0 {
		return NavItem{}, ErrInvalid
	}
	normalized, err := normalizeUpdateNavItem(input)
	if err != nil {
		return NavItem{}, err
	}
	return s.store.UpdateNavItem(ctx, normalized)
}

func (s *Service) DeleteNavItem(ctx context.Context, actor identity.Actor, id int64) error {
	if !s.canManage(actor) {
		return identity.ErrPermissionDenied
	}
	if id <= 0 {
		return ErrInvalid
	}
	return s.store.DeleteNavItem(ctx, id)
}

// --- Friend links ---

func (s *Service) ListPublicFriendLinks(ctx context.Context) ([]FriendLink, error) {
	return s.store.ListFriendLinks(ctx, true)
}

func (s *Service) ListAdminFriendLinks(ctx context.Context, actor identity.Actor) ([]FriendLink, error) {
	if !s.canManage(actor) {
		return nil, identity.ErrPermissionDenied
	}
	return s.store.ListFriendLinks(ctx, false)
}

func (s *Service) CreateFriendLink(ctx context.Context, actor identity.Actor, input CreateFriendLinkInput) (FriendLink, error) {
	if !s.canManage(actor) {
		return FriendLink{}, identity.ErrPermissionDenied
	}
	normalized, err := normalizeCreateFriendLink(input)
	if err != nil {
		return FriendLink{}, err
	}
	return s.store.CreateFriendLink(ctx, normalized)
}

func (s *Service) UpdateFriendLink(ctx context.Context, actor identity.Actor, input UpdateFriendLinkInput) (FriendLink, error) {
	if !s.canManage(actor) {
		return FriendLink{}, identity.ErrPermissionDenied
	}
	if input.ID <= 0 {
		return FriendLink{}, ErrInvalid
	}
	normalized, err := normalizeUpdateFriendLink(input)
	if err != nil {
		return FriendLink{}, err
	}
	return s.store.UpdateFriendLink(ctx, normalized)
}

func (s *Service) DeleteFriendLink(ctx context.Context, actor identity.Actor, id int64) error {
	if !s.canManage(actor) {
		return identity.ErrPermissionDenied
	}
	if id <= 0 {
		return ErrInvalid
	}
	return s.store.DeleteFriendLink(ctx, id)
}

// --- Announcements ---

func (s *Service) ListPublicAnnouncements(ctx context.Context) ([]Announcement, error) {
	return s.store.ListAnnouncements(ctx, true, true)
}

func (s *Service) ListAdminAnnouncements(ctx context.Context, actor identity.Actor) ([]Announcement, error) {
	if !s.canManage(actor) {
		return nil, identity.ErrPermissionDenied
	}
	return s.store.ListAnnouncements(ctx, false, false)
}

func (s *Service) CreateAnnouncement(ctx context.Context, actor identity.Actor, input CreateAnnouncementInput) (Announcement, error) {
	if !s.canManage(actor) {
		return Announcement{}, identity.ErrPermissionDenied
	}
	normalized, err := normalizeCreateAnnouncement(input)
	if err != nil {
		return Announcement{}, err
	}
	return s.store.CreateAnnouncement(ctx, normalized)
}

func (s *Service) UpdateAnnouncement(ctx context.Context, actor identity.Actor, input UpdateAnnouncementInput) (Announcement, error) {
	if !s.canManage(actor) {
		return Announcement{}, identity.ErrPermissionDenied
	}
	if input.ID <= 0 {
		return Announcement{}, ErrInvalid
	}
	normalized, err := normalizeUpdateAnnouncement(input)
	if err != nil {
		return Announcement{}, err
	}
	return s.store.UpdateAnnouncement(ctx, normalized)
}

func (s *Service) DeleteAnnouncement(ctx context.Context, actor identity.Actor, id int64) error {
	if !s.canManage(actor) {
		return identity.ErrPermissionDenied
	}
	if id <= 0 {
		return ErrInvalid
	}
	return s.store.DeleteAnnouncement(ctx, id)
}

// --- normalize helpers ---

func normalizeCreateNavItem(input CreateNavItemInput) (CreateNavItemInput, error) {
	labelZh, ok := normalizeBounded(input.LabelZhCN, 1, labelMaxRunes)
	if !ok {
		return CreateNavItemInput{}, ErrInvalid
	}
	labelEn, ok := normalizeBounded(input.LabelEnUS, 1, labelMaxRunes)
	if !ok {
		return CreateNavItemInput{}, ErrInvalid
	}
	href, ok := normalizeHref(input.Href)
	if !ok {
		return CreateNavItemInput{}, ErrInvalid
	}
	position, ok := normalizePosition(input.Position)
	if !ok {
		return CreateNavItemInput{}, ErrInvalid
	}
	return CreateNavItemInput{
		LabelZhCN:    labelZh,
		LabelEnUS:    labelEn,
		Href:         href,
		OpenInNewTab: input.OpenInNewTab,
		Position:     position,
		Enabled:      input.Enabled,
	}, nil
}

func normalizeUpdateNavItem(input UpdateNavItemInput) (UpdateNavItemInput, error) {
	out := UpdateNavItemInput{ID: input.ID, OpenInNewTab: input.OpenInNewTab, Enabled: input.Enabled}
	if input.LabelZhCN != nil {
		value, ok := normalizeBounded(*input.LabelZhCN, 1, labelMaxRunes)
		if !ok {
			return UpdateNavItemInput{}, ErrInvalid
		}
		out.LabelZhCN = &value
	}
	if input.LabelEnUS != nil {
		value, ok := normalizeBounded(*input.LabelEnUS, 1, labelMaxRunes)
		if !ok {
			return UpdateNavItemInput{}, ErrInvalid
		}
		out.LabelEnUS = &value
	}
	if input.Href != nil {
		value, ok := normalizeHref(*input.Href)
		if !ok {
			return UpdateNavItemInput{}, ErrInvalid
		}
		out.Href = &value
	}
	if input.Position != nil {
		value, ok := normalizePosition(*input.Position)
		if !ok {
			return UpdateNavItemInput{}, ErrInvalid
		}
		out.Position = &value
	}
	return out, nil
}

func normalizeCreateFriendLink(input CreateFriendLinkInput) (CreateFriendLinkInput, error) {
	name, ok := normalizeBounded(input.Name, 1, nameMaxRunes)
	if !ok {
		return CreateFriendLinkInput{}, ErrInvalid
	}
	linkURL, ok := normalizeExternalOrPathURL(input.URL, false)
	if !ok {
		return CreateFriendLinkInput{}, ErrInvalid
	}
	description, ok := normalizeBounded(input.Description, 0, descriptionMaxRunes)
	if !ok {
		return CreateFriendLinkInput{}, ErrInvalid
	}
	logoURL, ok := normalizeExternalOrPathURL(input.LogoURL, true)
	if !ok {
		return CreateFriendLinkInput{}, ErrInvalid
	}
	position, ok := normalizePosition(input.Position)
	if !ok {
		return CreateFriendLinkInput{}, ErrInvalid
	}
	return CreateFriendLinkInput{
		Name:        name,
		URL:         linkURL,
		Description: description,
		LogoURL:     logoURL,
		Position:    position,
		Enabled:     input.Enabled,
	}, nil
}

func normalizeUpdateFriendLink(input UpdateFriendLinkInput) (UpdateFriendLinkInput, error) {
	out := UpdateFriendLinkInput{ID: input.ID, Enabled: input.Enabled}
	if input.Name != nil {
		value, ok := normalizeBounded(*input.Name, 1, nameMaxRunes)
		if !ok {
			return UpdateFriendLinkInput{}, ErrInvalid
		}
		out.Name = &value
	}
	if input.URL != nil {
		value, ok := normalizeExternalOrPathURL(*input.URL, false)
		if !ok {
			return UpdateFriendLinkInput{}, ErrInvalid
		}
		out.URL = &value
	}
	if input.Description != nil {
		value, ok := normalizeBounded(*input.Description, 0, descriptionMaxRunes)
		if !ok {
			return UpdateFriendLinkInput{}, ErrInvalid
		}
		out.Description = &value
	}
	if input.LogoURL != nil {
		value, ok := normalizeExternalOrPathURL(*input.LogoURL, true)
		if !ok {
			return UpdateFriendLinkInput{}, ErrInvalid
		}
		out.LogoURL = &value
	}
	if input.Position != nil {
		value, ok := normalizePosition(*input.Position)
		if !ok {
			return UpdateFriendLinkInput{}, ErrInvalid
		}
		out.Position = &value
	}
	return out, nil
}

func normalizeCreateAnnouncement(input CreateAnnouncementInput) (CreateAnnouncementInput, error) {
	titleZh, ok := normalizeBounded(input.TitleZhCN, 0, titleMaxRunes)
	if !ok {
		return CreateAnnouncementInput{}, ErrInvalid
	}
	titleEn, ok := normalizeBounded(input.TitleEnUS, 0, titleMaxRunes)
	if !ok {
		return CreateAnnouncementInput{}, ErrInvalid
	}
	bodyZh, ok := normalizeBounded(input.BodyZhCN, 0, bodyMaxRunes)
	if !ok {
		return CreateAnnouncementInput{}, ErrInvalid
	}
	bodyEn, ok := normalizeBounded(input.BodyEnUS, 0, bodyMaxRunes)
	if !ok {
		return CreateAnnouncementInput{}, ErrInvalid
	}
	// 至少一侧有标题或正文，避免空横幅。
	if titleZh == "" && titleEn == "" && bodyZh == "" && bodyEn == "" {
		return CreateAnnouncementInput{}, ErrInvalid
	}
	style, ok := normalizeStyle(input.Style)
	if !ok {
		return CreateAnnouncementInput{}, ErrInvalid
	}
	href, ok := normalizeHrefOptional(input.Href)
	if !ok {
		return CreateAnnouncementInput{}, ErrInvalid
	}
	position, ok := normalizePosition(input.Position)
	if !ok {
		return CreateAnnouncementInput{}, ErrInvalid
	}
	if input.StartsAt != nil && input.EndsAt != nil && input.EndsAt.Before(*input.StartsAt) {
		return CreateAnnouncementInput{}, ErrInvalid
	}
	return CreateAnnouncementInput{
		TitleZhCN:   titleZh,
		TitleEnUS:   titleEn,
		BodyZhCN:    bodyZh,
		BodyEnUS:    bodyEn,
		Style:       style,
		Href:        href,
		Dismissible: input.Dismissible,
		Position:    position,
		Enabled:     input.Enabled,
		StartsAt:    input.StartsAt,
		EndsAt:      input.EndsAt,
	}, nil
}

func normalizeUpdateAnnouncement(input UpdateAnnouncementInput) (UpdateAnnouncementInput, error) {
	out := UpdateAnnouncementInput{
		ID:            input.ID,
		Dismissible:   input.Dismissible,
		Enabled:       input.Enabled,
		ClearStartsAt: input.ClearStartsAt,
		ClearEndsAt:   input.ClearEndsAt,
	}
	if input.TitleZhCN != nil {
		value, ok := normalizeBounded(*input.TitleZhCN, 0, titleMaxRunes)
		if !ok {
			return UpdateAnnouncementInput{}, ErrInvalid
		}
		out.TitleZhCN = &value
	}
	if input.TitleEnUS != nil {
		value, ok := normalizeBounded(*input.TitleEnUS, 0, titleMaxRunes)
		if !ok {
			return UpdateAnnouncementInput{}, ErrInvalid
		}
		out.TitleEnUS = &value
	}
	if input.BodyZhCN != nil {
		value, ok := normalizeBounded(*input.BodyZhCN, 0, bodyMaxRunes)
		if !ok {
			return UpdateAnnouncementInput{}, ErrInvalid
		}
		out.BodyZhCN = &value
	}
	if input.BodyEnUS != nil {
		value, ok := normalizeBounded(*input.BodyEnUS, 0, bodyMaxRunes)
		if !ok {
			return UpdateAnnouncementInput{}, ErrInvalid
		}
		out.BodyEnUS = &value
	}
	if input.Style != nil {
		value, ok := normalizeStyle(*input.Style)
		if !ok {
			return UpdateAnnouncementInput{}, ErrInvalid
		}
		out.Style = &value
	}
	if input.Href != nil {
		value, ok := normalizeHrefOptional(*input.Href)
		if !ok {
			return UpdateAnnouncementInput{}, ErrInvalid
		}
		out.Href = &value
	}
	if input.Position != nil {
		value, ok := normalizePosition(*input.Position)
		if !ok {
			return UpdateAnnouncementInput{}, ErrInvalid
		}
		out.Position = &value
	}
	if !input.ClearStartsAt {
		out.StartsAt = input.StartsAt
	}
	if !input.ClearEndsAt {
		out.EndsAt = input.EndsAt
	}
	if out.StartsAt != nil && out.EndsAt != nil && out.EndsAt.Before(*out.StartsAt) {
		return UpdateAnnouncementInput{}, ErrInvalid
	}
	return out, nil
}

func normalizeBounded(value string, minRunes, maxRunes int) (string, bool) {
	value = strings.TrimSpace(value)
	n := utf8.RuneCountInString(value)
	if n < minRunes || n > maxRunes {
		return "", false
	}
	return value, true
}

func normalizePosition(value int) (int, bool) {
	if value < positionMin || value > positionMax {
		return 0, false
	}
	return value, true
}

func normalizeStyle(value string) (string, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return StyleInfo, true
	}
	for _, allowed := range announcementStyles {
		if value == allowed {
			return value, true
		}
	}
	return "", false
}

// normalizeHref 导航链接：相对路径或 http(s)。
func normalizeHref(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	return normalizeExternalOrPathURL(value, false)
}

func normalizeHrefOptional(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	return normalizeExternalOrPathURL(value, false)
}

func normalizeExternalOrPathURL(value string, allowEmpty bool) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", allowEmpty
	}
	if utf8.RuneCountInString(value) > hrefMaxRunes {
		return "", false
	}
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return value, true
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed == nil || parsed.Host == "" {
		return "", false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	return value, true
}

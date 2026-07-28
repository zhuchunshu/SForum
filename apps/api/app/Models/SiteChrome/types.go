package sitechrome

import (
	"context"
	"errors"
	"time"
)

const (
	StyleInfo    = "info"
	StyleSuccess = "success"
	StyleWarning = "warning"
	StyleDanger  = "danger"

	CodeInvalid  = "site_chrome.invalid"
	CodeNotFound = "site_chrome.not_found"
	CodeConflict = "site_chrome.conflict"
)

var (
	ErrInvalid  = errors.New("site_chrome: invalid input")
	ErrNotFound = errors.New("site_chrome: not found")
	ErrConflict = errors.New("site_chrome: revision conflict")
)

// NavItem 顶部导航项（双语标签）。
type NavItem struct {
	ID           int64     `json:"id"`
	LabelZhCN    string    `json:"labelZhCN"`
	LabelEnUS    string    `json:"labelEnUS"`
	Href         string    `json:"href"`
	OpenInNewTab bool      `json:"openInNewTab"`
	Position     int       `json:"position"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// ExtensionNavItem 是 forum.nav.items 宿主安全描述符（E2.3）。
// Kind: hostLink | extensionRoute；URL 对 extensionRoute 为 /extensions/{id}{path}。
// 公开顶栏专用：禁止 /admin 与 /api。
type ExtensionNavItem struct {
	ExtensionID string            `json:"extensionId"`
	ID          string            `json:"id"`
	Order       int               `json:"order"`
	Label       map[string]string `json:"label,omitempty"`
	Icon        string            `json:"icon,omitempty"`
	Kind        string            `json:"kind"`
	Method      string            `json:"method,omitempty"`
	URL         string            `json:"url"`
}

// ExtensionNavItemProvider 解析 forum.nav.items；nil 时公开导航不含扩展项。
type ExtensionNavItemProvider interface {
	ExtensionNavItems(ctx context.Context) ([]ExtensionNavItem, error)
}

type CreateNavItemInput struct {
	LabelZhCN    string
	LabelEnUS    string
	Href         string
	OpenInNewTab bool
	Position     int
	Enabled      bool
}

type UpdateNavItemInput struct {
	ID           int64
	LabelZhCN    *string
	LabelEnUS    *string
	Href         *string
	OpenInNewTab *bool
	Position     *int
	Enabled      *bool
}

// FriendLink 页脚/友情链接区条目。
type FriendLink struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	LogoURL     string    `json:"logoUrl"`
	Position    int       `json:"position"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateFriendLinkInput struct {
	Name        string
	URL         string
	Description string
	LogoURL     string
	Position    int
	Enabled     bool
}

type UpdateFriendLinkInput struct {
	ID          int64
	Name        *string
	URL         *string
	Description *string
	LogoURL     *string
	Position    *int
	Enabled     *bool
}

// Announcement 首页公告横幅。
type Announcement struct {
	ID          int64      `json:"id"`
	TitleZhCN   string     `json:"titleZhCN"`
	TitleEnUS   string     `json:"titleEnUS"`
	BodyZhCN    string     `json:"bodyZhCN"`
	BodyEnUS    string     `json:"bodyEnUS"`
	Style       string     `json:"style"`
	Href        string     `json:"href"`
	Dismissible bool       `json:"dismissible"`
	Position    int        `json:"position"`
	Enabled     bool       `json:"enabled"`
	StartsAt    *time.Time `json:"startsAt,omitempty"`
	EndsAt      *time.Time `json:"endsAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type CreateAnnouncementInput struct {
	TitleZhCN   string
	TitleEnUS   string
	BodyZhCN    string
	BodyEnUS    string
	Style       string
	Href        string
	Dismissible bool
	Position    int
	Enabled     bool
	StartsAt    *time.Time
	EndsAt      *time.Time
}

type UpdateAnnouncementInput struct {
	ID          int64
	TitleZhCN   *string
	TitleEnUS   *string
	BodyZhCN    *string
	BodyEnUS    *string
	Style       *string
	Href        *string
	Dismissible *bool
	Position    *int
	Enabled     *bool
	StartsAt    *time.Time
	// ClearStartsAt 为 true 时清空 starts_at（与 nil 指针「不改」区分）。
	ClearStartsAt bool
	EndsAt        *time.Time
	ClearEndsAt   bool
}

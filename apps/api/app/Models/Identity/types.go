package identity

import (
	"errors"
	"time"

	avatar "github.com/zhuchunshu/sforum/apps/api/app/Support/Avatar"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusBanned   UserStatus = "banned"
)

const (
	CodeRegisterInvalid     = "auth.register_invalid"
	CodeRegisterDisabled    = "auth.register_disabled"
	CodeSessionUnavailable  = "auth.session_unavailable"
	CodeSessionPolicyDenied = "auth.session_policy_denied"
	AuditActionLogin        = "auth.login.success"
	AuditActionRegister     = "auth.register.success"

	FieldUsername          = "username"
	FieldEmail             = "email"
	FieldPassword          = "password"
	FieldHumanVerification = "humanVerification"

	MessageUsernameRequired  = "auth.username_required"
	MessageEmailRequired     = "auth.email_required"
	MessageEmailInvalid      = "auth.email_invalid"
	MessagePasswordMin       = "auth.password_min_length"
	MessagePasswordMax       = "auth.password_max_length"
	MessagePasswordLowercase = "auth.password_lowercase"
	MessagePasswordUppercase = "auth.password_uppercase"
	MessagePasswordNumber    = "auth.password_number"
	MessagePasswordSymbol    = "auth.password_symbol"
	MessageUsernameTaken     = "auth.username_taken"
	MessageEmailTaken        = "auth.email_taken"
	MessageUsernameTooShort  = "auth.username_too_short"
	MessageUsernameTooLong   = "auth.username_too_long"
	MessageUsernameReserved  = "auth.username_reserved"
	MessageUsernameCharset   = "auth.username_invalid_charset"
	CodeLoginLocked          = "auth.login_locked"
)

var (
	ErrInvalidCredentials = errors.New("identity: invalid credentials")
	// ErrLoginLocked 连续登录失败触发临时锁定。
	ErrLoginLocked = errors.New("identity: login temporarily locked")
	// ErrLoginVerificationRequired 密码正确，但账号累计风险要求额外人机验证。
	ErrLoginVerificationRequired = errors.New("identity: login verification required")
	// ErrRegistrationDisabled 表示运营已关闭开放注册，且当前不在首用户 bootstrap 窗口。
	ErrRegistrationDisabled       = errors.New("identity: registration disabled")
	ErrCredentialNotFound         = errors.New("identity: credential not found")
	ErrPermissionDenied           = errors.New("identity: permission denied")
	ErrInvalidPermission          = errors.New("identity: invalid permission")
	ErrInvalidRole                = errors.New("identity: invalid role")
	ErrInvalidRoleInput           = errors.New("identity: invalid role input")
	ErrPermissionOverrideConflict = errors.New("identity: permission override conflict")
	ErrSystemRoleLocked           = errors.New("identity: system role is locked")
	ErrDefaultRoleLocked          = errors.New("identity: default role is locked")
	ErrInitialSuperAdminLocked    = errors.New("identity: initial super admin is locked")
	ErrSuperAdminOverridesLocked  = errors.New("identity: super admin permission overrides are locked")
	ErrSelfRoleChange             = errors.New("identity: actors cannot change their own roles or overrides")
	ErrSuperAdminGrantRestricted  = errors.New("identity: only super admin can grant or manage super_admin role")
	ErrUsernameOrEmailNotUnique   = errors.New("identity: username or email is not unique")
	ErrPasswordDoesNotMeetPolicy  = errors.New("identity: password does not meet policy")
	ErrPasswordResetTokenNotFound = errors.New("identity: password reset token not found or expired")
	// 密码重置请求过于频繁（按邮箱/IP 限流）。
	ErrPasswordResetRateLimited = errors.New("identity: password reset rate limited")
	// 会话目录：要操作的会话不存在或不属于当前用户（含越权访问别人的 sid）。
	ErrSessionNotFound = errors.New("identity: session not found")
	// 管理员试图强制下线自己的全部设备（应改用 logout）。
	ErrSelfSessionRevoke = errors.New("identity: cannot revoke own sessions via admin path")
	// 非超管管理员试图强制下线超管用户的设备，超管账户受保护（与 ReplaceUserRoles 的保护对称）。
	ErrSuperAdminSessionLocked = errors.New("identity: super admin sessions cannot be revoked by non-super-admin")
	// 目标用户不存在（管理路径）。
	ErrUserNotFound = errors.New("identity: user not found")
	// 管理员更新账户/资料时字段不合法。
	ErrInvalidUserUpdate = errors.New("identity: invalid user update")
	// 禁止通过管理路径修改自己的账号状态（避免自锁）。
	ErrSelfStatusChange = errors.New("identity: cannot change own status via admin path")
)

// 会话下线原因（写入 user_sessions.revoke_reason）。
const (
	RevokeReasonLogout          = "logout"
	RevokeReasonDevice          = "revoke_device"
	RevokeReasonOthers          = "revoke_others"
	RevokeReasonMaxExceeded     = "max_exceeded"
	RevokeReasonPasswordReset   = "password_reset"
	RevokeReasonSessionReplaced = "session_replaced"
)

// SessionRecord 是 user_sessions 一行的领域视图，用于设备列表/历史展示。
// 注意：不包含任何可用于劫持会话的凭证（无 cookie session id、无 raw token）；
// SID 是 server 生成的 opaque 标识，仅用于指定「下线哪一条」，无法用来登录。
type SessionRecord struct {
	ID           string     `json:"id"`           // opaque 会话标识，前端用来指定下线哪一条
	DeviceName   string     `json:"deviceName"`   // 由 UA 解析的展示名，如 "Chrome on macOS"
	Browser      string     `json:"browser"`      // 浏览器名
	OS           string     `json:"os"`           // 操作系统名
	IPPrefix     string     `json:"ipPrefix"`     // 脱敏 IP 前缀，如 "1.2.3.*"
	CreatedAt    time.Time  `json:"createdAt"`    // 登录时间
	LastSeenAt   time.Time  `json:"lastSeenAt"`   // 最后活跃时间
	IsCurrent    bool       `json:"isCurrent"`    // 是否当前请求所在设备
	RevokedAt    *time.Time `json:"revokedAt"`    // 历史记录里可见的下线时间
	RevokeReason string     `json:"revokeReason"` // 下线原因
}

// SessionListResult 是设备列表/历史的分页结果。
type SessionListResult struct {
	Items   []SessionRecord `json:"items"`
	Total   int64           `json:"total"`
	Page    int             `json:"page"`
	PerPage int             `json:"perPage"`
}

// ClearUserClientIPsResult 是清空用户相关真实 IP 后的影响行数（隐私合规）。
type ClearUserClientIPsResult struct {
	SessionsCleared int `json:"sessionsCleared"`
	TopicsCleared   int `json:"topicsCleared"`
	CommentsCleared int `json:"commentsCleared"`
}

type FieldMessages map[string][]string

type RegisterInvalidError struct {
	Fields FieldMessages
}

func (e *RegisterInvalidError) Error() string {
	return "identity: registration input is invalid"
}

func NewRegisterInvalid(fields FieldMessages) *RegisterInvalidError {
	return &RegisterInvalidError{Fields: fields}
}

type Actor struct {
	ID          int64
	Status      UserStatus
	RoleKeys    []string
	Permissions map[string]bool
	// CreatedAt 用于新人信任阶梯；零值表示未知（跳过新人限制）。
	CreatedAt time.Time
}

type PostSummary struct {
	ID           int64
	AuthorUserID int64
}

type CurrentUser struct {
	ID                  int64       `json:"id"`
	Username            string      `json:"username"`
	DisplayName         string      `json:"displayName"`
	Avatar              avatar.View `json:"avatar"`
	Locale              string      `json:"locale"`
	Status              UserStatus  `json:"status"`
	IsInitialSuperAdmin bool        `json:"isInitialSuperAdmin"`
	RoleKeys            []string    `json:"roleKeys"`
	Permissions         []string    `json:"permissions"`
	// CreatedAt 注册时间；不强制暴露给所有前端，但 Actor 构建需要。
	CreatedAt time.Time `json:"-"`
	// CurrentTokenVersion binds a successful credential proof to the authority
	// revision that existed when the credential row was read.
	CurrentTokenVersion int64 `json:"-"`
}

type RegistrationStatus struct {
	// 始终为 false：不向公开端点泄露 bootstrap 窗口信息。
	NextUserIsInitialSuperAdmin bool `json:"nextUserIsInitialSuperAdmin"`
	// 前端是否应展示注册入口/允许提交。已含 bootstrap 覆盖（无用户时恒为 true）。
	RegistrationEnabled bool `json:"registrationEnabled"`
}

type LoginAudit struct {
	UserID      int64
	Action      string
	IPAddress   string
	UserAgent   string
	SessionHash string
}

type Permission struct {
	Key         string `json:"key"`
	Module      string `json:"module"`
	Description string `json:"description"`
}

type RolePermissionSet struct {
	RoleKey        string   `json:"roleKey"`
	PermissionKeys []string `json:"permissionKeys"`
}

type PermissionMatrix struct {
	Permissions []Permission        `json:"permissions"`
	Roles       []RolePermissionSet `json:"roles"`
}

type PermissionOverrides struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

type AdminUserSummary struct {
	ID                  int64      `json:"id"`
	Username            string     `json:"username"`
	Email               string     `json:"email"`
	DisplayName         string     `json:"displayName"`
	Locale              string     `json:"locale"`
	Status              UserStatus `json:"status"`
	IsInitialSuperAdmin bool       `json:"isInitialSuperAdmin"`
	RoleKeys            []string   `json:"roleKeys"`
	// CreatedAt / UpdatedAt 列表可选展示；旧客户端忽略未知字段。
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AdminUserProfile 是后台可查看/编辑的公开资料字段（来自 user_profiles）。
type AdminUserProfile struct {
	Bio        string `json:"bio"`
	Signature  string `json:"signature"`
	Location   string `json:"location"`
	WebsiteURL string `json:"websiteUrl"`
}

type AdminUserDetail struct {
	AdminUserSummary
	Permissions         []string            `json:"permissions"`
	PermissionOverrides PermissionOverrides `json:"permissionOverrides"`
	Profile             AdminUserProfile    `json:"profile"`
}

// AdminUpdateUserInput 管理员更新账户/资料。指针字段为 nil 表示不改。
type AdminUpdateUserInput struct {
	Username    *string
	Email       *string
	DisplayName *string
	Locale      *string
	Status      *UserStatus
	Bio         *string
	Signature   *string
	Location    *string
	WebsiteURL  *string
}

type UserListInput struct {
	Page    int
	PerPage int
	Query   string
	Status  string
	RoleKey string
}

type AdminUserList struct {
	Items   []AdminUserSummary `json:"items"`
	Total   int64              `json:"total"`
	Page    int                `json:"page"`
	PerPage int                `json:"perPage"`
}

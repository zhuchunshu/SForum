package identity

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// 昵称（users.display_name）是展示标签，不是身份：唯一性只属于 username。
// 本文件只放与「改名」直接相关的纯逻辑（策略视图、归一化、校验、冷却计算），
// 存储与编排分别在 nickname_postgres_store.go / nickname_service.go。
const (
	FieldDisplayName = "displayName"

	MessageNicknameInvalidChars  = "profile.nickname_invalid_chars"
	MessageNicknameTooShort      = "profile.nickname_too_short"
	MessageNicknameTooLong       = "profile.nickname_too_long"
	MessageNicknameReserved      = "profile.nickname_reserved"
	MessageNicknameUsernameTaken = "profile.nickname_username_taken"
	MessageNicknameAtPrefix      = "profile.nickname_at_prefix"

	NicknameSourceSelf  = "self"
	NicknameSourceAdmin = "admin"
)

// ErrNicknameDisabled 表示运营关闭了自助改名。它是策略拒绝而不是权限不足，
// 因此调用方应把它映射成 422 而不是 403。
var ErrNicknameDisabled = errors.New("identity: nickname change is disabled")

// NicknamePolicy 是昵称运行时策略（由 Options 服务实现）。
type NicknamePolicy struct {
	AllowUserChange bool
	CooldownMinutes int
	MinLength       int
	MaxLength       int
	HistoryPublic   bool
}

type NicknamePolicyResolver interface {
	NicknamePolicy(ctx context.Context) (NicknamePolicy, error)
}

// NicknameSnapshot 是存储层返回的昵称事实，不含策略判断。
type NicknameSnapshot struct {
	UserID      int64
	Username    string
	DisplayName string
	ChangedAt   *time.Time
}

// NicknameState 是对外暴露的昵称状态。CanChange 已经合并了开关与冷却，
// 前端不需要自己再算一遍；NextChangeAt 为空表示没有冷却限制。
type NicknameState struct {
	DisplayName  string     `json:"displayName"`
	ChangedAt    *time.Time `json:"changedAt,omitempty"`
	NextChangeAt *time.Time `json:"nextChangeAt,omitempty"`
	CanChange    bool       `json:"canChange"`
	Duplicates   int        `json:"duplicates"`
}

// NicknameAvailability 是改名前的预检结果。重名不算「不可用」——昵称本来
// 就不唯一，Duplicates 只用于提示；Conflict 才是会拒绝保存的原因。
type NicknameAvailability struct {
	Available  bool   `json:"available"`
	Duplicates int    `json:"duplicates"`
	Conflict   string `json:"conflict,omitempty"`
	Normalized string `json:"normalized,omitempty"`
}

// NicknameChange 是一条改名历史。
type NicknameChange struct {
	ID        int64     `json:"id"`
	OldName   string    `json:"oldName"`
	NewName   string    `json:"newName"`
	Source    string    `json:"source"`
	ChangedBy *int64    `json:"changedBy,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// NicknameInvalidError 携带字段级原因（值是 i18n key，与注册校验一致）。
type NicknameInvalidError struct {
	Fields FieldMessages
}

func (e *NicknameInvalidError) Error() string {
	return "identity: nickname input is invalid"
}

// NicknameCooldownError 表示冷却未到期；RetryAt 是允许再次改名的时刻。
type NicknameCooldownError struct {
	RetryAt time.Time
}

func (e *NicknameCooldownError) Error() string {
	return "identity: nickname cooldown is active"
}

// normalizeNicknameValue 折叠空白并拒绝不可见字符。
//
// 先拒绝再折叠：换行、制表符、零宽字符都是输入噪声或伪造手段，不能靠
// 「折叠成空格」蒙混过去；NBSP 与全角空格属于合法空白，交给 strings.Fields
// 归一成半角单空格。
func normalizeNicknameValue(raw string) (string, bool) {
	for _, r := range raw {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return "", false
		}
	}
	return strings.Join(strings.Fields(raw), " "), true
}

func nicknameReserved(value string, reserved []string) bool {
	candidate := strings.ToLower(strings.TrimSpace(value))
	for _, item := range reserved {
		if strings.ToLower(strings.TrimSpace(item)) == candidate {
			return true
		}
	}
	return false
}

// validateNicknameInput 返回最终要写入的值与字段级错误。
//
// 留空是合法输入，表示「显示用户名」：它不是用户自选的字符串，所以跳过
// 长度、保留词与撞名校验，直接回落到 username。
func validateNicknameInput(raw string, username string, policy NicknamePolicy, reserved []string) (string, FieldMessages) {
	value, ok := normalizeNicknameValue(raw)
	if !ok {
		return "", FieldMessages{FieldDisplayName: {MessageNicknameInvalidChars}}
	}
	if value == "" {
		return username, nil
	}
	if strings.HasPrefix(value, "@") {
		return "", FieldMessages{FieldDisplayName: {MessageNicknameAtPrefix}}
	}
	length := utf8.RuneCountInString(value)
	if length < policy.MinLength {
		return "", FieldMessages{FieldDisplayName: {MessageNicknameTooShort}}
	}
	if length > policy.MaxLength {
		return "", FieldMessages{FieldDisplayName: {MessageNicknameTooLong}}
	}
	if nicknameReserved(value, reserved) {
		return "", FieldMessages{FieldDisplayName: {MessageNicknameReserved}}
	}
	return value, nil
}

// nextNicknameChangeAt 计算冷却到期时刻；0 分钟表示不限制。
func nextNicknameChangeAt(changedAt *time.Time, cooldownMinutes int) *time.Time {
	if changedAt == nil || cooldownMinutes <= 0 {
		return nil
	}
	next := changedAt.Add(time.Duration(cooldownMinutes) * time.Minute)
	return &next
}

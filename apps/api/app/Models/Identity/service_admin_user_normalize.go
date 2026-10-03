package identity

import (
	"context"
	"strings"
)

// 管理端用户更新的字段校验。它体积不小且与其余账户逻辑耦合很少，拆到独立文件
// 是为了让 service.go 保持在架构门禁的行数上限内，而不是给它加新职责。
func (s *Service) normalizeAdminUpdateUserInput(ctx context.Context, input AdminUpdateUserInput) (AdminUpdateUserInput, error) {
	out := AdminUpdateUserInput{}
	fields := FieldMessages{}

	// 必须写进 out：调用方用的是返回值，写回 input 会被静默丢弃——
	// 那个错误的表现是「保存成功但值没变」，不会有任何报错。
	if input.Kind != nil {
		// 与用户名、邮箱一致：先归一化再校验。契约里 kind 是小写字面量，
		// 大小写差异是输入噪声而不是另一个值。
		kind := UserKind(strings.ToLower(strings.TrimSpace(string(*input.Kind))))
		if !ValidUserKind(kind) {
			return AdminUpdateUserInput{}, ErrInvalidUserUpdate
		}
		out.Kind = &kind
	}

	if input.Username != nil {
		username := strings.TrimSpace(*input.Username)
		usernamePolicy, err := s.resolveUsernamePolicy(ctx)
		if err != nil {
			return AdminUpdateUserInput{}, err
		}
		if username == "" {
			addFieldMessage(fields, FieldUsername, MessageUsernameRequired)
		} else if usernamePolicy.MinLength > 0 || usernamePolicy.MaxLength > 0 || usernamePolicy.Charset != "" || len(usernamePolicy.Reserved) > 0 {
			if ok, reason := usernamePolicy.Validate(username); !ok {
				addFieldMessage(fields, FieldUsername, reason)
			}
		}
		out.Username = &username
	}
	if input.Email != nil {
		email := strings.TrimSpace(*input.Email)
		if email == "" {
			addFieldMessage(fields, FieldEmail, MessageEmailRequired)
		} else if !isValidEmail(email) {
			addFieldMessage(fields, FieldEmail, MessageEmailInvalid)
		}
		out.Email = &email
	}
	if input.DisplayName != nil {
		displayName := strings.TrimSpace(*input.DisplayName)
		if displayName == "" {
			return AdminUpdateUserInput{}, ErrInvalidUserUpdate
		}
		if len([]rune(displayName)) > maxAdminDisplayNameLength {
			return AdminUpdateUserInput{}, ErrInvalidUserUpdate
		}
		out.DisplayName = &displayName
	}
	if input.Locale != nil {
		locale := strings.TrimSpace(*input.Locale)
		if locale == "" {
			locale = "zh-CN"
		}
		// 仅接受当前产品支持的语言码，避免写入任意字符串。
		if locale != "zh-CN" && locale != "en-US" {
			return AdminUpdateUserInput{}, ErrInvalidUserUpdate
		}
		out.Locale = &locale
	}
	if input.Status != nil {
		status := *input.Status
		switch status {
		case UserStatusActive, UserStatusDisabled, UserStatusBanned:
			out.Status = &status
		default:
			return AdminUpdateUserInput{}, ErrInvalidUserUpdate
		}
	}
	if input.Bio != nil {
		bio := strings.TrimSpace(*input.Bio)
		if len([]rune(bio)) > maxAdminBioLength {
			return AdminUpdateUserInput{}, ErrInvalidUserUpdate
		}
		out.Bio = &bio
	}
	if input.Signature != nil {
		signature := strings.TrimSpace(*input.Signature)
		if len([]rune(signature)) > maxAdminSignatureLength {
			return AdminUpdateUserInput{}, ErrInvalidUserUpdate
		}
		out.Signature = &signature
	}
	if input.Location != nil {
		location := strings.TrimSpace(*input.Location)
		if len([]rune(location)) > maxAdminLocationLength {
			return AdminUpdateUserInput{}, ErrInvalidUserUpdate
		}
		out.Location = &location
	}
	if input.WebsiteURL != nil {
		url := strings.TrimSpace(*input.WebsiteURL)
		if len(url) > maxAdminWebsiteLength {
			return AdminUpdateUserInput{}, ErrInvalidUserUpdate
		}
		if url != "" {
			lower := strings.ToLower(url)
			if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
				return AdminUpdateUserInput{}, ErrInvalidUserUpdate
			}
		}
		out.WebsiteURL = &url
	}

	if len(fields) > 0 {
		return AdminUpdateUserInput{}, NewRegisterInvalid(fields)
	}
	// 至少要改一个字段。新增可更新字段时必须同步加进这里，否则「只改这个字段」
	// 的请求会被判成什么都没改而拒绝。
	if out.Username == nil && out.Email == nil && out.DisplayName == nil && out.Locale == nil &&
		out.Status == nil && out.Kind == nil &&
		out.Bio == nil && out.Signature == nil && out.Location == nil && out.WebsiteURL == nil {
		return AdminUpdateUserInput{}, ErrInvalidUserUpdate
	}
	return out, nil
}

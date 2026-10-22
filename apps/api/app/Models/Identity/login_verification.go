package identity

import "context"

// LoginInput 是一次密码登录的输入。ClientIP/DeviceFingerprint 只用于节流与
// login_risk 判定；HumanVerified/SkipHumanVerification 由 HTTP 层设置。
type LoginInput struct {
	Login    string
	Password string
	// ClientIP 用于分层登录节流（account+IP / IP）；可空。
	ClientIP string
	// DeviceFingerprint 是与会话登记同口径的 UA 指纹（useragent.UserAgentRaw），
	// 仅用于 login_risk 的新设备判定；空值视为已知设备。
	DeviceFingerprint string
	// HumanVerified 仅由 HTTP 层在 login_risk 验证成功后设置。
	HumanVerified bool
	// SkipHumanVerification 由 HTTP 层在「登录同时签发 API Token」时设置。
	// 这是已知取舍：该路径仍会签发浏览器会话，等同一处可绕过的验证豁免面，
	// 见 knowledge/decisions/2026-10-22-login-human-verification-triggers.md。
	SkipHumanVerification bool
}

// WithLoginVerification 注入登录强化策略（可选；总是/新设备/特权账号）。
func (s *Service) WithLoginVerification(policy LoginVerificationPolicyResolver) *Service {
	s.loginVerification = policy
	return s
}

// LoginVerificationPolicy 是运营配置的登录强化触发策略。
//
// 四个触发条件相互独立，但都受 Enabled（human_verification.scenarios.login_risk）
// 总闸约束；关闭总闸时 Always/NewDevice/Privileged 一律不生效。
type LoginVerificationPolicy struct {
	// Enabled 是 human_verification.scenarios.login_risk 总闸。
	Enabled bool
	// Always 要求每次密码登录都先完成人机验证。
	Always bool
	// NewDevice 要求从未见过的设备（UA 指纹未命中活跃会话）先完成验证。
	NewDevice bool
	// Privileged 要求管理面账号先完成验证。
	Privileged bool
}

// LoginVerificationPolicyResolver 读取登录强化策略（通常由 Options 实现）。
type LoginVerificationPolicyResolver interface {
	LoginVerificationPolicy(ctx context.Context) (LoginVerificationPolicy, error)
}

// loginRequiresVerification 判定本次登录是否必须先完成人机验证。
//
// 判定顺序：显式豁免 → 失败锁定（历史行为，与运营配置无关）→ 总闸 →
// 总是验证 → 新设备 → 特权账号。判定发生在密码校验通过之后，错误密码既不会
// 消耗验证令牌，也不会产生账号存在性信号。
//
// 失败姿态：策略读取或设备指纹查询出错时 fail open，避免配置/存储抖动把合法
// 用户挡在门外；失败锁定本身仍由 Redis 计数决定。
func loginRequiresVerification(s *Service, ctx context.Context, loginKey string, credential CredentialUser, input LoginInput) (bool, error) {
	if input.SkipHumanVerification {
		return false, nil
	}
	if s.loginLockout != nil {
		required, err := s.loginLockout.RequiresVerification(ctx, loginKey)
		if err != nil {
			return false, nil
		}
		if required {
			return true, nil
		}
	}
	policy := loginVerificationPolicy(s, ctx)
	if !policy.Enabled {
		return false, nil
	}
	if policy.Always {
		return true, nil
	}
	if policy.NewDevice {
		known, err := s.HasKnownDevice(ctx, credential.ID, input.DeviceFingerprint)
		if err == nil && !known {
			return true, nil
		}
	}
	if policy.Privileged && privilegedAccount(credential.CurrentUser) {
		return true, nil
	}
	return false, nil
}

func loginVerificationPolicy(s *Service, ctx context.Context) LoginVerificationPolicy {
	if s.loginVerification == nil {
		return LoginVerificationPolicy{}
	}
	policy, err := s.loginVerification.LoginVerificationPolicy(ctx)
	if err != nil {
		return LoginVerificationPolicy{}
	}
	return policy
}

// privilegedAccount 判定账号是否属于需要强化的管理面账号：
// 初始超管标记、super_admin 角色，或拥有后台访问权限。
func privilegedAccount(current CurrentUser) bool {
	if current.IsInitialSuperAdmin {
		return true
	}
	for _, roleKey := range current.RoleKeys {
		if roleKey == RoleSuperAdmin {
			return true
		}
	}
	for _, permission := range current.Permissions {
		if permission == PermissionAdminAccess {
			return true
		}
	}
	return false
}

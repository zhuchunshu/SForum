package identity

import (
	"context"
)

// RecommendedMaxDevices 是最大活跃浏览器会话数的推荐默认值。
// 论坛场景下普通用户极少同时使用超过 5 台设备（手机/电脑/平板等），
// 取较保守值以降低旧设备残留会话的风险。admin 可在 1-20 间调整。
const RecommendedMaxDevices = 5

// RecommendedSessionsKeepDays 是历史会话（已下线）的默认保留天数。
// 保留 30 天供用户回看登录历史，超过后由 periodic job 清理。
const RecommendedSessionsKeepDays = 30

// NameSessionsMaxDevices 是「最大活跃浏览器会话数」的 runtime option 名。
// admin 可在 settings 页调整；非 public（仅后端登录时读取）。
const NameSessionsMaxDevices = "identity.sessions.max_devices"

// NameSessionsKeepDays 是「已下线历史会话保留天数」的 runtime option 名。
// 超过后由 periodic job 清理；非 public。
const NameSessionsKeepDays = "identity.sessions.keep_days"

// maxDevicesLowerBound / maxDevicesUpperBound 是 identity.sessions.max_devices 的取值上下限。
const (
	maxDevicesLowerBound = 1
	maxDevicesUpperBound = 20
)

// NormalizeMaxDevices 将任意输入归一化到合法的最大设备数区间。
// 非法值（<=0 或超出上限）回退到推荐默认值，遵循 beginner-friendly 原则：
// 配置错误不应让功能失效，而应回落到安全的推荐值。
func NormalizeMaxDevices(value int) int {
	if value < maxDevicesLowerBound || value > maxDevicesUpperBound {
		return RecommendedMaxDevices
	}
	return value
}

// ListSessions 列出当前用户的会话/设备。
// actor 是操作者（自服务，只能看自己的会话）；currentSID 用于标记 isCurrent。
// includeHistory=true 时含已下线的历史记录。
func (s *Service) ListSessions(ctx context.Context, userID int64, currentSID string, includeHistory bool, page int, perPage int) (SessionListResult, error) {
	return s.store.ListUserSessions(ctx, userID, currentSID, includeHistory, page, perPage)
}

// RevokeSession 下线当前用户的单个设备。
// 越权保护：store 层 WHERE user_id = actor.ID 强制只能操作自己的会话，
// 传别人的 sid 会因不匹配返回 ErrSessionNotFound（不泄漏该 sid 是否属于他人）。
func (s *Service) RevokeSession(ctx context.Context, userID int64, sid string) error {
	return s.store.RevokeSession(ctx, userID, sid, RevokeReasonDevice)
}

// RevokeOtherSessions 下线除当前设备外的所有其他设备，返回被下线的条数。
func (s *Service) RevokeOtherSessions(ctx context.Context, userID int64, currentSID string) (int, error) {
	return s.store.RevokeOtherSessions(ctx, userID, currentSID, RevokeReasonOthers)
}

// AdminRevokeUserSessions 管理员强制下线目标用户的全部活跃设备。
// 权限边界：
//   - actor 必须持有 user.manage；
//   - 禁止对自己操作（管理员下线自己请用 logout）；
//   - 目标若是 super_admin，则只有同样是 super_admin 的管理员才能下线其设备，
//     防止普通 user.manage 持有者把超管账户踢下线（与 ReplaceUserRoles 的保护对称）。
//
// 这不依赖 sid（管理员不知道目标用户的 sid），按 user_id 全量下线。
func (s *Service) AdminRevokeUserSessions(ctx context.Context, actor Actor, targetUserID int64) (int, error) {
	if !actor.Can(PermissionUserManage) {
		return 0, ErrPermissionDenied
	}
	if actor.ID == targetUserID {
		return 0, ErrSelfSessionRevoke
	}
	// 加载目标用户以检查是否为受保护的 super_admin。
	target, err := s.store.GetAdminUser(ctx, targetUserID)
	if err != nil {
		return 0, err
	}
	if containsString(target.RoleKeys, RoleSuperAdmin) && !actor.IsSuperAdmin() {
		return 0, ErrSuperAdminSessionLocked
	}
	return s.store.RevokeUserSessions(ctx, targetUserID, "admin_revoke")
}

// CleanupRevokedSessions 清理已下线超过保留期的历史会话行（periodic job 调用）。
// 返回删除条数。keepDays<=0 时用推荐默认 30 天。
func (s *Service) CleanupRevokedSessions(ctx context.Context, keepDays int) (int, error) {
	if keepDays < 1 {
		keepDays = RecommendedSessionsKeepDays
	}
	return s.store.DeleteOldRevokedSessions(ctx, keepDays)
}

// ClearUserClientIPs 管理员清空目标用户相关的真实客户端 IP（隐私合规）。
// 需要 user.manage；禁止对 super_admin 操作（除非操作者自己是 super_admin）。
// 账号删除/封禁流程实现后应在状态变更事务中复用本方法。
func (s *Service) ClearUserClientIPs(ctx context.Context, actor Actor, targetUserID int64) (ClearUserClientIPsResult, error) {
	if !actor.Can(PermissionUserManage) {
		return ClearUserClientIPsResult{}, ErrPermissionDenied
	}
	if targetUserID <= 0 {
		return ClearUserClientIPsResult{}, ErrUserNotFound
	}
	target, err := s.store.GetAdminUser(ctx, targetUserID)
	if err != nil {
		return ClearUserClientIPsResult{}, err
	}
	if containsString(target.RoleKeys, RoleSuperAdmin) && !actor.IsSuperAdmin() {
		return ClearUserClientIPsResult{}, ErrSuperAdminSessionLocked
	}
	return s.store.ClearUserClientIPs(ctx, targetUserID)
}

// HasKnownDevice 判断该用户是否已有记录的活跃设备（指纹命中）。
//
// 设计用途：风险登录判断——若设备指纹是新出现的，可触发 login_risk 人机验证。
// 现状（2026-07-10）：本方法已实现并测试，但尚未在 login 流程中接通（login controller
// 暂未调用）。保留接口供后续 login_risk 场景接入，勿误以为风险检测已生效。
//
// 指纹当前按 user_agent_raw 等值匹配（调用方传截断后的 UA）；空指纹视为已知（跳过风险检查）。
func (s *Service) HasKnownDevice(ctx context.Context, userID int64, fingerprint string) (bool, error) {
	if fingerprint == "" {
		return true, nil
	}
	return s.store.HasSessionFingerprint(ctx, userID, fingerprint)
}

// EnforceMaxSessions 在登录后强制最大活跃设备数，踢出最旧的超出设备。
// currentSID 是本次登录的会话标识，一定不会被踢，保证刚登录的设备立即可用。
// 由 controller 在 Begin/Save 成功后调用。
func (s *Service) EnforceMaxSessions(ctx context.Context, userID int64, currentSID string, maxDevices int) (int, error) {
	return s.store.EnforceMaxSessions(ctx, userID, currentSID, NormalizeMaxDevices(maxDevices))
}

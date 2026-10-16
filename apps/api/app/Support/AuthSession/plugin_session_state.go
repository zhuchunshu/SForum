package authsession

import (
	"context"
	"strings"
)

// PluginSessionStateInvalidator 是宿主「插件命名空间会话状态」存储的最窄形状。
//
// 为什么接口声明在消费方而不是导入存储包：
//  1. 避免 Support/AuthSession 依赖存储实现（以及它的 Redis 客户端），AuthSession 只应
//     负责身份；会话状态的删除是一个副作用，由装配层注入。
//  2. 本包的测试因此不需要起 Redis。
//
// 语义要点：这是**擦除**钩子，不是访问控制。一个被撤销/替换/过期的 sid 根本无法再
// 发起请求，所以它的插件状态本来就不可达；调用这里的目的是不让字节在会话失效后继续
// 留着（隐私与卫生）。因此：
//   - 必须 best-effort：失败不得让登出或登录失败；
//   - 传入的是原始 sid，指纹化由装配层完成，避免两处各写一遍派生逻辑。
type PluginSessionStateInvalidator interface {
	InvalidateSessionBySID(ctx context.Context, sid string) (int, error)
}

// WithPluginSessionStateInvalidator 注入插件会话状态擦除钩子。未注入时登出/登录
// 仍完全正常，只是不会主动擦除那些字节（它们仍会随会话寿命到期不可达）。
func (m *Manager) WithPluginSessionStateInvalidator(invalidator PluginSessionStateInvalidator) *Manager {
	if m == nil {
		return m
	}
	m.pluginSessionState = invalidator
	return m
}

// dropPluginSessionState 尽力擦除某个 sid 的插件状态。
//
// 刻意不返回错误：调用点（登出、登录替换）不能因为插件状态的清理失败而失败，
// 那会把一个后台卫生问题升级成用户可见的登录/登出故障。失败只记录为日志由调用方
// 的审计路径承担；这里保证不 panic、不阻塞。
func (m *Manager) dropPluginSessionState(ctx context.Context, sid string) {
	if m == nil || m.pluginSessionState == nil {
		return
	}
	sid = strings.TrimSpace(sid)
	if sid == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	func() {
		defer func() { _ = recover() }()
		_, _ = m.pluginSessionState.InvalidateSessionBySID(ctx, sid)
	}()
}

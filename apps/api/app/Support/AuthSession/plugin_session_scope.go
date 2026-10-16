package authsession

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v3"
)

// RequestScope 是宿主在一次 HTTP 请求里解析出的 Core 会话范围（S9）。
//
// 它只属于宿主：既不上线协议，也不交给插件。插件能拿到的只有一份**不透明**的
// 委派令牌，令牌里不含会话指纹；指纹只存在于宿主自己的签发记录里（S8：插件永远
// 看不到会话标识符）。
type RequestScope struct {
	// SessionFingerprint 是会话标识符的 sha256hex。已认证会话用稳定的 sid
	// （跨 24h 轮换不变，S4/S5）；匿名会话用 cookie 会话 id（匿名会话不参与轮换，
	// 因此同样在会话寿命内稳定）。原始标识符永不离开本包。
	SessionFingerprint string
	// IdleExpiresAt / AbsoluteExpiresAt 是 Core 会话的两条寿命边界。零值表示
	// 该边界未知；未知不会被当成无限（存储层只做收紧，不放宽）。
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
	// ActorUserID 为 0 表示匿名会话。
	ActorUserID int64
}

type pluginSessionScopeKey struct{}

// WithPluginSessionScope 把会话范围挂到请求的 Go context 上。只有宿主自己在 HTTP
// 请求里发起的插件调用会经过这里；后台任务（job/队列）拿不到范围，因此也签发不出
// 会话委派。
func WithPluginSessionScope(ctx context.Context, scope RequestScope) context.Context {
	if ctx == nil {
		return ctx
	}
	return context.WithValue(ctx, pluginSessionScopeKey{}, scope)
}

// PluginSessionScopeFromContext 读取会话范围。缺失表示「本次调用没有会话范围」，
// 调用方必须 fail closed（不签发委派），而不是伪造一个。
func PluginSessionScopeFromContext(ctx context.Context) (RequestScope, bool) {
	if ctx == nil {
		return RequestScope{}, false
	}
	scope, ok := ctx.Value(pluginSessionScopeKey{}).(RequestScope)
	if !ok || scope.SessionFingerprint == "" {
		return RequestScope{}, false
	}
	return scope, true
}

// PluginSessionScope 解析当前请求的会话范围。
//
// 与 CurrentSID 的区别：CurrentSID 只回答「这是哪个 sid」，而这里要回答「插件状态
// 能活多久、按什么指纹隔离」。两个边界都取自会话自身的事实而不是配置愿望：
//
//   - idle 边界 = now + IdleTimeout，即下一次 Save 会写进 Redis 的那个 TTL；
//   - 绝对边界 = fiber 会话存储实际保存的绝对过期时间（唯一权威），取不到时退化为
//     created_at + AbsoluteTimeout（只可能更早，因此只会更保守）。
//
// 任一必需事实缺失（没有会话对象、指纹算不出、绝对边界在配置了绝对超时却读不到）时
// 返回 false：宁可不给插件会话状态，也不给一个可能比 Core 会话更长的寿命。
func (m *Manager) PluginSessionScope(c fiber.Ctx) (RequestScope, bool) {
	if m == nil || c == nil || m.store == nil {
		return RequestScope{}, false
	}
	state, err := m.requestSessionState(c)
	if err != nil || state == nil || state.session == nil {
		return RequestScope{}, false
	}
	if state.pending.Load() {
		// 会话正在签发/销毁，此刻的 sid 与寿命都还没定下来。
		return RequestScope{}, false
	}
	sess := state.session
	state.lifecycleMu.Lock()
	if state.pendingActive {
		state.lifecycleMu.Unlock()
		return RequestScope{}, false
	}
	userID, _ := sessionUserID(sess.Get(sessionUserIDKey))
	sid, _ := sess.Get(sessionSIDKey).(string)
	createdAt := sessionTime(sess.Get(sessionCreatedAtKey))
	absolute := storedSessionAbsoluteExpiration(sess)
	state.lifecycleMu.Unlock()

	identifier := sid
	if identifier == "" {
		// 匿名会话没有 sid（sid 只在登录签发时写入）。这里刻意要求会话**已经被持久化**
		// 才给出范围：本轮实测，Core 并没有挂全局 session 中间件，匿名会话只由签发/续期
		// 路径写 Redis，因此「新生成的匿名会话」在本请求结束后就再也拿不到——按它的 id
		// 建键只会留下一份永远读不到的字节。未持久化时 fail closed（不给范围），
		// 插件会得到「没有会话委派」这个预期结果。
		if sess.Fresh() {
			return RequestScope{}, false
		}
		identifier = sess.ID()
	}
	if identifier == "" {
		return RequestScope{}, false
	}
	if absolute.IsZero() && m.store.AbsoluteTimeout > 0 {
		if createdAt.IsZero() {
			// 配置了绝对超时却读不到任何绝对边界：无法给出比 Core 会话更短的上界，
			// 于是不给会话范围（fail closed）。
			return RequestScope{}, false
		}
		absolute = createdAt.Add(m.store.AbsoluteTimeout)
	}
	scope := RequestScope{
		SessionFingerprint: fingerprintSessionIdentifier(identifier),
		ActorUserID:        userID,
	}
	if m.store.IdleTimeout > 0 {
		scope.IdleExpiresAt = m.currentTime().Add(m.store.IdleTimeout)
	}
	scope.AbsoluteExpiresAt = absolute
	return scope, true
}

// storedSessionAbsoluteExpiration 读取 fiber 会话存储保存的绝对过期时间。
//
// 为什么需要它：绝对寿命由 session.Store 在载荷里维护（键是未导出的类型），本包无法
// 直接取键；但该键是本包之外**唯一**的非字符串键，而时间值只有它一个来自存储层。
// 用「非字符串键 + time.Time 值」定位既不需要反射访问未导出类型，也不会把
// created_at/renewed_at 这类字符串键误认成绝对边界。读不到就返回零值，由调用方决定
// 退化策略。
func storedSessionAbsoluteExpiration(sess interface {
	Keys() []any
	Get(any) any
}) time.Time {
	if sess == nil {
		return time.Time{}
	}
	for _, key := range sess.Keys() {
		if _, isString := key.(string); isString {
			continue
		}
		if value, ok := sess.Get(key).(time.Time); ok {
			return value
		}
	}
	return time.Time{}
}

func fingerprintSessionIdentifier(identifier string) string {
	sum := sha256.Sum256([]byte(identifier))
	return hex.EncodeToString(sum[:])
}

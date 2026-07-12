package authsession

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	clientip "github.com/zhuchunshu/sforum/apps/api/app/Support/ClientIP"
)

const (
	sessionUserIDKey          = "user_id"
	sessionCreatedAtKey       = "created_at"
	sessionRenewedAtKey       = "renewed_at"
	sessionLoginIPKey         = "login_ip"
	sessionLoginAgentKey      = "login_user_agent"
	sessionTokenVersionKey    = "token_version"
	sessionSIDKey             = "sid"
	sessionLastSeenTouchedKey = "last_seen_touched"
)

// lastSeenInterval 限制刷新 user_sessions.last_seen_at 的写频率，避免每个请求都写。
const lastSeenInterval = time.Hour

// TokenVersionSource 返回用户当前的令牌版本号。
// 用于会话失效校验（M8）：session 存储创建时的版本号，校验时与用户当前版本比对，
// 不一致（如密码重置后递增了版本号）则视为会话失效。
// 为 nil 时跳过版本校验（向后兼容测试场景）。
type TokenVersionSource func(ctx context.Context, userID int64) (int64, error)

// SessionStore 是会话目录的存储契约（user_sessions 表）。
// 由 identity.PostgresStore 实现，authsession 通过本接口读写目录，
// 避免直接依赖 identity 包（防循环依赖）。
// 为 nil 时跳过设备目录功能（向后兼容测试场景）。
type SessionStore interface {
	// CreateSession 登记一条新会话/设备。
	CreateSession(ctx context.Context, input SessionRecordInput) error
	// IsSessionRevoked 判断 sid 是否已被下线（实现「下次请求失效」）。
	IsSessionRevoked(ctx context.Context, userID int64, sid string) (bool, error)
	// TouchSessionLastSeen 刷新最后活跃时间（节流调用）。
	TouchSessionLastSeen(ctx context.Context, userID int64, sid string) error
	// RevokeSession 标记单条会话下线（logout 时调用）。
	RevokeSession(ctx context.Context, userID int64, sid string, reason string) error
}

// SessionRecordInput 是登记会话时所需的展示字段（已由调用方从 UA/IP 解析好）。
type SessionRecordInput struct {
	UserID       int64
	SID          string
	SessionHash  string
	DeviceName   string
	Browser      string
	OS           string
	UserAgentRaw string
	// IPAddress 是规范化后的真实客户端 IP（全文，管理/风控用）。
	IPAddress string
	// IPPrefix 是脱敏展示前缀（用户「我的设备」列表）。
	IPPrefix string
}

type Config struct {
	RenewalInterval time.Duration
	HashSecret      string
	TokenVersion    TokenVersionSource
	SessionStore    SessionStore
}

type Manager struct {
	store           *session.Store
	renewalInterval time.Duration
	hashSecret      []byte
	tokenVersion    TokenVersionSource
	sessions        SessionStore
	now             func() time.Time
}

type Info struct {
	ID        string
	Hash      string
	SID       string
	CreatedAt time.Time
	RenewedAt time.Time
}

// Pending 是 Begin 后尚未持久化的会话；调用方可设置设备展示信息后再 Save。
type Pending struct {
	manager *Manager
	session *session.Session
	info    Info
	// deviceInfo 由调用方在 Save 前设置；非空时 Save 会写入会话目录。
	deviceInfo *SessionRecordInput
}

func NewManager(store *session.Store, cfg Config) *Manager {
	if store == nil {
		store = session.NewStore()
	}
	store.RegisterType(time.Time{})

	return &Manager{
		store:           store,
		renewalInterval: cfg.RenewalInterval,
		hashSecret:      []byte(cfg.HashSecret),
		tokenVersion:    cfg.TokenVersion,
		sessions:        cfg.SessionStore,
		now:             func() time.Time { return time.Now().UTC() },
	}
}

func (m *Manager) Start(c fiber.Ctx, userID int64) (Info, error) {
	pending, err := m.Begin(c, userID)
	if err != nil {
		return Info{}, err
	}
	if err := pending.Save(); err != nil {
		return Info{}, err
	}
	return pending.Info(), nil
}

func (m *Manager) Begin(c fiber.Ctx, userID int64) (*Pending, error) {
	sess, err := m.store.Get(c)
	if err != nil {
		return nil, err
	}

	if err := sess.Reset(); err != nil {
		return nil, err
	}

	now := m.currentTime()
	sid, err := generateSID()
	if err != nil {
		return nil, err
	}
	sess.Set(sessionUserIDKey, userID)
	sess.Set(sessionCreatedAtKey, now)
	sess.Set(sessionRenewedAtKey, now)
	// 登录 IP 写入 session payload（全文），供审计/风控；解析走 clientip 防代理伪造。
	sess.Set(sessionLoginIPKey, truncate(clientip.FromCtx(c), 128))
	sess.Set(sessionLoginAgentKey, truncate(c.Get(fiber.HeaderUserAgent), 512))
	sess.Set(sessionSIDKey, sid)
	// 记录创建会话时的令牌版本号，供后续 CurrentUserID 比对以实现会话失效（M8）。
	if m.tokenVersion != nil {
		if version, err := m.tokenVersion(c.Context(), userID); err == nil {
			sess.Set(sessionTokenVersionKey, version)
		}
	}

	return &Pending{
		manager: m,
		session: sess,
		info:    m.info(sess, now, now, sid),
	}, nil
}

func (m *Manager) CurrentUserID(c fiber.Ctx) (int64, bool, error) {
	sess, err := m.store.Get(c)
	if err != nil {
		return 0, false, err
	}

	userID, ok := sessionUserID(sess.Get(sessionUserIDKey))
	if !ok {
		return 0, false, nil
	}

	// 令牌版本号校验（M8）：session 版本与用户当前版本不一致则视为会话失效，
	// 用于密码重置/封禁后使旧会话立即失效。版本号缺失（旧 session 或未配置 source）时跳过。
	if m.tokenVersion != nil {
		sessionVersion, _ := sess.Get(sessionTokenVersionKey).(int64)
		currentVersion, err := m.tokenVersion(c.Context(), userID)
		if err != nil {
			return 0, false, nil
		}
		if sessionVersion != currentVersion {
			return 0, false, nil
		}
	}

	// 设备目录撤销校验：若 sid 已被下线（revoke_device/revoke_others/max_exceeded），
	// 视为未登录，实现「被下线设备在下次请求即失效」。
	if m.sessions != nil {
		sid, _ := sess.Get(sessionSIDKey).(string)
		if sid != "" {
			revoked, err := m.sessions.IsSessionRevoked(c.Context(), userID, sid)
			if err != nil || revoked {
				return 0, false, nil
			}
		}
	}

	if err := m.refresh(sess, userID); err != nil {
		return 0, false, err
	}
	return userID, true, nil
}

// CurrentSID 返回当前请求所在会话的 opaque sid。
// 用于设备列表标记 isCurrent 与「下线其他设备」时排除当前会话。
// 未配置会话目录或无有效会话时返回空串。
func (m *Manager) CurrentSID(c fiber.Ctx) (string, error) {
	sess, err := m.store.Get(c)
	if err != nil {
		return "", err
	}
	sid, _ := sess.Get(sessionSIDKey).(string)
	return sid, nil
}

func (m *Manager) Destroy(c fiber.Ctx) error {
	sess, err := m.store.Get(c)
	if err != nil {
		return err
	}
	// 登出时在目录标记本会话已下线（保留为历史记录），best-effort，不阻塞登出。
	if m.sessions != nil {
		userID, ok := sessionUserID(sess.Get(sessionUserIDKey))
		if ok && userID != 0 {
			sid, _ := sess.Get(sessionSIDKey).(string)
			if sid != "" {
				_ = m.sessions.RevokeSession(c.Context(), userID, sid, "logout")
			}
		}
	}
	return sess.Destroy()
}

func (m *Manager) Hash(sessionID string) string {
	mac := hmac.New(sha256.New, m.hashSecret)
	_, _ = mac.Write([]byte(sessionID))
	return hex.EncodeToString(mac.Sum(nil))
}

func (p *Pending) Info() Info {
	if p == nil {
		return Info{}
	}
	return p.info
}

// SetDeviceInfo 设置设备展示信息（device_name/browser/os/ip_prefix 等，由调用方解析 UA/IP 得到）。
// 必须在 Save 前调用；Save 时会据此写入会话目录。
func (p *Pending) SetDeviceInfo(info SessionRecordInput) {
	if p == nil {
		return
	}
	// 用 pending 已有的 sid/hash 补齐，确保一致。
	info.SID = p.info.SID
	info.SessionHash = p.info.Hash
	p.deviceInfo = &info
}

func (p *Pending) Save() error {
	if p == nil || p.session == nil {
		return nil
	}
	if err := p.session.Save(); err != nil {
		return err
	}
	// 目录写入失败不能静默成功：后续鉴权依赖目录判断远程下线状态。
	if p.manager.sessions != nil && p.deviceInfo != nil {
		if err := p.manager.sessions.CreateSession(context.Background(), *p.deviceInfo); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) refresh(sess *session.Session, userID int64) error {
	now := m.currentTime()
	createdAt := sessionTime(sess.Get(sessionCreatedAtKey))
	if createdAt.IsZero() {
		createdAt = now
		sess.Set(sessionCreatedAtKey, createdAt)
	}

	renewedAt := sessionTime(sess.Get(sessionRenewedAtKey))
	if renewedAt.IsZero() {
		renewedAt = createdAt
	}

	// 定期换 session id，降低长期复用同一个 id 带来的泄露影响。
	// sid 不随 cookie session id 轮换而变（保留在 payload），设备列表稳定。
	if m.renewalInterval > 0 && !renewedAt.After(now) && now.Sub(renewedAt) >= m.renewalInterval {
		if err := sess.Regenerate(); err != nil {
			return err
		}
		renewedAt = now
		sess.Set(sessionRenewedAtKey, renewedAt)
	}

	// 节流刷新 last_seen：距上次刷新超过 lastSeenInterval 时才写库，避免每请求都写。
	// 与 session 续期解耦：续期是安全相关（轮换 id），last_seen 是展示相关（最后活跃）。
	shouldTouchLastSeen := false
	touchSID := ""
	if m.sessions != nil {
		lastSeenTouched := sessionTime(sess.Get(sessionLastSeenTouchedKey))
		if lastSeenTouched.IsZero() || now.Sub(lastSeenTouched) >= lastSeenInterval {
			sid, _ := sess.Get(sessionSIDKey).(string)
			if sid != "" {
				shouldTouchLastSeen = true
				touchSID = sid
				sess.Set(sessionLastSeenTouchedKey, now)
			}
		}
	}

	if err := sess.Save(); err != nil {
		return err
	}
	if shouldTouchLastSeen {
		_ = m.sessions.TouchSessionLastSeen(context.Background(), userID, touchSID)
	}
	return nil
}

func (m *Manager) info(sess *session.Session, createdAt time.Time, renewedAt time.Time, sid string) Info {
	id := sess.ID()
	return Info{
		ID:        id,
		Hash:      m.Hash(id),
		SID:       sid,
		CreatedAt: createdAt,
		RenewedAt: renewedAt,
	}
}

func (m *Manager) currentTime() time.Time {
	if m == nil || m.now == nil {
		return time.Now().UTC()
	}
	return m.now().UTC()
}

// generateSID 生成稳定的 opaque 会话标识（与 cookie session id 独立）。
// 32 字节 crypto/rand → hex，用作 user_sessions 唯一键与前端「下线哪一条」的句柄。
// 注意：它是非认证凭证，泄漏也无法登录。
//
// 失败时返回 error 而非回退到时间戳派生值：后者可预测且可能碰撞（同纳秒并发），
// 会破坏 user_sessions 唯一性约束。crypto/rand 失败属于极端系统异常，此时拒绝登录
// 是正确的安全行为——用可预测值静默继续比登录失败更危险。
func generateSID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func sessionUserID(value any) (int64, bool) {
	switch id := value.(type) {
	case int64:
		return id, id != 0
	case int:
		return int64(id), id != 0
	default:
		return 0, false
	}
}

func sessionTime(value any) time.Time {
	parsed, ok := value.(time.Time)
	if !ok {
		return time.Time{}
	}
	return parsed
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len([]rune(value)) <= max {
		return value
	}

	runes := []rune(value)
	return string(runes[:max])
}

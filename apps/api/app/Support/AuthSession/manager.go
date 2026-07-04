package authsession

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

const (
	sessionUserIDKey     = "user_id"
	sessionCreatedAtKey  = "created_at"
	sessionRenewedAtKey  = "renewed_at"
	sessionLoginIPKey    = "login_ip"
	sessionLoginAgentKey = "login_user_agent"
)

type Config struct {
	RenewalInterval time.Duration
	HashSecret      string
}

type Manager struct {
	store           *session.Store
	renewalInterval time.Duration
	hashSecret      []byte
	now             func() time.Time
}

type Info struct {
	ID        string
	Hash      string
	CreatedAt time.Time
	RenewedAt time.Time
}

type Pending struct {
	manager *Manager
	session *session.Session
	info    Info
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
	sess.Set(sessionUserIDKey, userID)
	sess.Set(sessionCreatedAtKey, now)
	sess.Set(sessionRenewedAtKey, now)
	sess.Set(sessionLoginIPKey, truncate(c.IP(), 128))
	sess.Set(sessionLoginAgentKey, truncate(c.Get(fiber.HeaderUserAgent), 512))

	return &Pending{
		manager: m,
		session: sess,
		info:    m.info(sess, now, now),
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

	if err := m.refresh(sess); err != nil {
		return 0, false, err
	}
	return userID, true, nil
}

func (m *Manager) Destroy(c fiber.Ctx) error {
	sess, err := m.store.Get(c)
	if err != nil {
		return err
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

func (p *Pending) Save() error {
	if p == nil || p.session == nil {
		return nil
	}
	return p.session.Save()
}

func (m *Manager) refresh(sess *session.Session) error {
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
	if m.renewalInterval > 0 && !renewedAt.After(now) && now.Sub(renewedAt) >= m.renewalInterval {
		if err := sess.Regenerate(); err != nil {
			return err
		}
		renewedAt = now
		sess.Set(sessionRenewedAtKey, renewedAt)
	}

	return sess.Save()
}

func (m *Manager) info(sess *session.Session, createdAt time.Time, renewedAt time.Time) Info {
	id := sess.ID()
	return Info{
		ID:        id,
		Hash:      m.Hash(id),
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

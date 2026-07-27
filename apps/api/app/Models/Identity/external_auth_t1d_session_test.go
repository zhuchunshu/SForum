package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// T1D：session-bound recent-auth、unlink 幂等键、password 策略与权威注册校验。

// memoryRecentAuth 是进程内 session-bound recent-auth，供跨 session 隔离测试。
type memoryRecentAuth struct {
	mu   sync.Mutex
	rows map[string]time.Time // key: userID|fingerprint
}

func newMemoryRecentAuth() *memoryRecentAuth {
	return &memoryRecentAuth{rows: map[string]time.Time{}}
}

func (m *memoryRecentAuth) key(userID int64, fp string) string {
	return strings.TrimSpace(fp) + "|" + itoa64(userID)
}

func itoa64(v int64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := v < 0
	if neg {
		v = -v
	}
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func (m *memoryRecentAuth) MarkSessionRecentlyAuthenticated(
	_ context.Context, userID int64, sessionFingerprint, method, providerID string, ttl time.Duration,
) error {
	_ = method
	_ = providerID
	fp := strings.ToLower(strings.TrimSpace(sessionFingerprint))
	if userID <= 0 || !validSessionFingerprint(fp) {
		return errors.New("invalid mark input")
	}
	if ttl <= 0 {
		ttl = RecentAuthDefaultTTL
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rows[m.key(userID, fp)] = time.Now().Add(ttl)
	return nil
}

func (m *memoryRecentAuth) IsSessionRecentlyAuthenticated(
	_ context.Context, userID int64, sessionFingerprint string,
) (bool, error) {
	fp := strings.ToLower(strings.TrimSpace(sessionFingerprint))
	if userID <= 0 || !validSessionFingerprint(fp) {
		return false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	exp, ok := m.rows[m.key(userID, fp)]
	return ok && time.Now().Before(exp), nil
}

func TestSessionFingerprintNonReversible(t *testing.T) {
	sid := "opaque-session-id-example"
	fp := SessionFingerprint(sid)
	if len(fp) != 64 {
		t.Fatalf("fingerprint length: %d", len(fp))
	}
	sum := sha256.Sum256([]byte(sid))
	if fp != hex.EncodeToString(sum[:]) {
		t.Fatalf("fingerprint mismatch")
	}
	if SessionFingerprint("") != "" {
		t.Fatalf("empty sid must yield empty fingerprint")
	}
	if SessionFingerprint(sid) == sid {
		t.Fatalf("fingerprint must not equal raw sid")
	}
}

// TestT1D_RecentAuthCrossSessionIsolation：同一 user 的另一 session 不能继承 recent-auth。
func TestT1D_RecentAuthCrossSessionIsolation(t *testing.T) {
	store := newMemoryRecentAuth()
	ctx := context.Background()
	userID := int64(42)
	fpA := SessionFingerprint("session-A")
	fpB := SessionFingerprint("session-B")
	if err := store.MarkSessionRecentlyAuthenticated(ctx, userID, fpA, "password", "", RecentAuthDefaultTTL); err != nil {
		t.Fatalf("mark: %v", err)
	}
	okA, err := store.IsSessionRecentlyAuthenticated(ctx, userID, fpA)
	if err != nil || !okA {
		t.Fatalf("session A should be recent: %v %v", okA, err)
	}
	okB, err := store.IsSessionRecentlyAuthenticated(ctx, userID, fpB)
	if err != nil || okB {
		t.Fatalf("session B must NOT inherit recent-auth: %v %v", okB, err)
	}
	// 不同 user 同 fingerprint 也不共享。
	okOther, err := store.IsSessionRecentlyAuthenticated(ctx, 99, fpA)
	if err != nil || okOther {
		t.Fatalf("other user must not share fingerprint: %v %v", okOther, err)
	}
}

// TestT1D_AuthorizeLinkRequiresSessionBoundRecentAuth：空 fingerprint / 错误 session fail closed。
func TestT1D_AuthorizeLinkRequiresSessionBoundRecentAuth(t *testing.T) {
	ResetIdentitySubjectHMACKeyForTest()
	t.Setenv(IdentitySubjectHMACSecretEnv, "")
	t.Setenv("SFORUM_IN_PRODUCTION", "")

	store := newMemoryRecentAuth()
	digest := strings.Repeat("d", 64)
	providerID := "demo.auth"
	userID := int64(7)
	fpGood := SessionFingerprint("good-session")
	_ = store.MarkSessionRecentlyAuthenticated(context.Background(), userID, fpGood, "password", "", RecentAuthDefaultTTL)

	svc := NewExternalAuthService(ExternalAuthDeps{
		RecentAuth: store,
	})
	tx := t1aBaseTX(providerID, digest, ExternalAuthOperationLink, userID)
	ctx := context.Background()

	if err := svc.AuthorizeLinkBeforePersist(ctx, tx, userID, ""); !errors.Is(err, ErrExternalAuthRecentAuthRequired) {
		t.Fatalf("empty fingerprint: %v", err)
	}
	if err := svc.AuthorizeLinkBeforePersist(ctx, tx, userID, SessionFingerprint("other")); !errors.Is(err, ErrExternalAuthRecentAuthRequired) {
		t.Fatalf("other session: %v", err)
	}
	if err := svc.AuthorizeLinkBeforePersist(ctx, tx, userID, fpGood); err != nil {
		t.Fatalf("good session: %v", err)
	}
}

// TestT1D_UnlinkIdempotencyKeyBindsUserLinkRevisionRequest：不绑定 client IP。
func TestT1D_UnlinkIdempotencyKeyBindsUserLinkRevisionRequest(t *testing.T) {
	k1 := unlinkIdempotencyKey(10, 20, 3, "req-a")
	k2 := unlinkIdempotencyKey(10, 20, 3, "req-b")
	k3 := unlinkIdempotencyKey(10, 20, 4, "req-a")
	k4 := unlinkIdempotencyKey(11, 20, 3, "req-a")
	if k1 == k2 || k1 == k3 || k1 == k4 {
		t.Fatalf("keys must differ by request/revision/user: %q %q %q %q", k1, k2, k3, k4)
	}
	if strings.Contains(k1, "127.0.0.1") || strings.Contains(k1, "ip") {
		t.Fatalf("key must not bind client IP: %q", k1)
	}
	if !validExternalIdentityIdempotencyKey(k1) {
		t.Fatalf("key must be valid: %q", k1)
	}
	// 过长 requestID 仍产生合法 key。
	long := strings.Repeat("x", 200)
	kLong := unlinkIdempotencyKey(1, 2, 1, long)
	if !validExternalIdentityIdempotencyKey(kLong) {
		t.Fatalf("long key shortened invalid: %q", kLong)
	}
}

// TestT1D_ValidateRegisterIdentityFieldsShared：外部注册复用密码无关权威字段校验。
func TestT1D_ValidateRegisterIdentityFieldsShared(t *testing.T) {
	policy := UsernamePolicy{MinLength: 3, MaxLength: 20, Charset: "ascii", Reserved: []string{"admin"}}
	fields := validateRegisterIdentityFields("ab", "bad", policy)
	if len(fields[FieldUsername]) == 0 || len(fields[FieldEmail]) == 0 {
		t.Fatalf("expected username+email errors: %#v", fields)
	}
	fields = validateRegisterIdentityFields("admin", "user@example.com", policy)
	if len(fields[FieldUsername]) == 0 {
		t.Fatalf("expected reserved username: %#v", fields)
	}
	fields = validateRegisterIdentityFields("alice", "user@example.com", policy)
	if len(fields) != 0 {
		t.Fatalf("expected clean: %#v", fields)
	}
	// 密码路径在 identity 字段之上叠加 password 校验。
	withPass := validateRegisterInputWithUsername("alice", "user@example.com", "short", RecommendedPasswordPolicy(), policy)
	if len(withPass[FieldPassword]) == 0 {
		t.Fatalf("password path must still validate password")
	}
}

// TestT1D_ExternalRegistrationInputNormalized：displayName/locale 默认。
func TestT1D_ExternalRegistrationInputNormalized(t *testing.T) {
	n := ExternalRegistrationInput{Username: "  bob  ", Email: "  a@b.co "}.Normalized()
	if n.Username != "bob" || n.Email != "a@b.co" || n.DisplayName != "bob" || n.Locale != "zh-CN" {
		t.Fatalf("normalized: %#v", n)
	}
}

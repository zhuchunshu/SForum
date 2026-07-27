package identity

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// Callback state 与 registration ticket 的内存实现测试。
// 覆盖：并发安全、原子一次性消费、TTL、hash key、过期、重放拒绝、绑定强制、PKCE。

func validRegistrationTicket(token string, expiresAt time.Time) RegistrationTicket {
	now := time.Now()
	created := now
	if !expiresAt.IsZero() && expiresAt.Before(now) {
		created = expiresAt.Add(-time.Minute)
	}
	return RegistrationTicket{
		Token:              token,
		ProviderID:         "sforum.auth-github.auth",
		OwnerExtensionID:   "sforum.auth-github",
		OwnerPackageDigest: strings.Repeat("a", 64),
		Operation:          ExternalAuthOperationRegistration,
		ProviderSubject:    "raw-subject-123",
		CreatedAt:          created,
		ExpiresAt:          expiresAt,
	}
}

func TestInMemoryCallbackStateStore_ConsumeOnceAndReplayRejected(t *testing.T) {
	store := NewInMemoryCallbackStateStore()
	ctx := context.Background()
	tx := CallbackTransaction{
		State:      "state-1",
		ProviderID: "sforum.auth-github.auth",
		Operation:  ExternalAuthOperationLogin,
		ExpiresAt:  time.Now().Add(time.Minute),
	}
	if err := store.Save(ctx, tx); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := store.Consume(ctx, "state-1")
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if got.ProviderID != tx.ProviderID || got.Operation != tx.Operation {
		t.Fatalf("consumed tx mismatch: %+v", got)
	}
	// 再次消费：重放必须失败。
	if _, err := store.Consume(ctx, "state-1"); err != ErrCallbackStateReplayed {
		t.Fatalf("replay consume err = %v, want ErrCallbackStateReplayed", err)
	}
	// 未知 state：失败。
	if _, err := store.Consume(ctx, "unknown"); err != ErrCallbackStateInvalid {
		t.Fatalf("unknown state consume err = %v, want ErrCallbackStateInvalid", err)
	}
}

func TestInMemoryCallbackStateStore_ExpiredRejected(t *testing.T) {
	store := NewInMemoryCallbackStateStore()
	ctx := context.Background()
	tx := CallbackTransaction{
		State:     "state-expired",
		ExpiresAt: time.Now().Add(-time.Second), // 已过期
	}
	if err := store.Save(ctx, tx); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := store.Consume(ctx, "state-expired"); err != ErrCallbackStateExpired {
		t.Fatalf("expired consume err = %v, want ErrCallbackStateExpired", err)
	}
	// 过期后再消费应是 replayed（state 已被记为已消费）。
	if _, err := store.Consume(ctx, "state-expired"); err != ErrCallbackStateReplayed {
		t.Fatalf("post-expire consume err = %v, want ErrCallbackStateReplayed", err)
	}
}

func TestInMemoryCallbackStateStore_FillsTimestampsAndClampsTTL(t *testing.T) {
	store := NewInMemoryCallbackStateStoreWithTTL(time.Minute)
	ctx := context.Background()
	// 未填时间戳：Save 必须填充；超长 ExpiresAt 必须夹紧到 TTL。
	tx := CallbackTransaction{State: "state-ttl"}
	if err := store.Save(ctx, tx); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := store.Peek(ctx, "state-ttl")
	if err != nil {
		t.Fatalf("peek: %v", err)
	}
	if got.CreatedAt.IsZero() || got.ExpiresAt.IsZero() {
		t.Fatalf("timestamps must be populated: created=%v expires=%v", got.CreatedAt, got.ExpiresAt)
	}
	if got.ExpiresAt.Sub(got.CreatedAt) > time.Minute+time.Second {
		t.Fatalf("expires must clamp to TTL: %v", got.ExpiresAt.Sub(got.CreatedAt))
	}

	// 显式超长 ExpiresAt 夹紧。
	long := CallbackTransaction{
		State:     "state-long",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Save(ctx, long); err != nil {
		t.Fatalf("save long: %v", err)
	}
	got, err = store.Peek(ctx, "state-long")
	if err != nil {
		t.Fatalf("peek long: %v", err)
	}
	if got.ExpiresAt.Sub(time.Now()) > time.Minute+2*time.Second {
		t.Fatalf("long expiry must clamp to ~1m, remaining=%v", time.Until(got.ExpiresAt))
	}
}

func TestInMemoryCallbackStateStore_StoresUnderHashNotRawToken(t *testing.T) {
	store := NewInMemoryCallbackStateStore()
	ctx := context.Background()
	raw := "browser-visible-state-token"
	if err := store.Save(ctx, CallbackTransaction{
		State:     raw,
		ExpiresAt: time.Now().Add(time.Minute),
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	// 内部 map 不得以 raw token 为 key。
	store.mu.Lock()
	_, rawPresent := store.entries[raw]
	hashKey := opaqueTokenHash(raw)
	_, hashPresent := store.entries[hashKey]
	store.mu.Unlock()
	if rawPresent {
		t.Fatal("store must not use raw browser token as map key")
	}
	if !hashPresent {
		t.Fatal("store must use sha256(token) as map key")
	}
	// Consume 仍用浏览器 token。
	if _, err := store.Consume(ctx, raw); err != nil {
		t.Fatalf("consume by browser token: %v", err)
	}
}

func TestInMemoryCallbackStateStore_ConcurrentConsumeOnce(t *testing.T) {
	store := NewInMemoryCallbackStateStore()
	ctx := context.Background()
	if err := store.Save(ctx, CallbackTransaction{
		State:     "state-race",
		ExpiresAt: time.Now().Add(time.Minute),
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	const n = 32
	var wg sync.WaitGroup
	errs := make(chan error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, err := store.Consume(ctx, "state-race")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	ok, replayed, other := 0, 0, 0
	for err := range errs {
		switch err {
		case nil:
			ok++
		case ErrCallbackStateReplayed, ErrCallbackStateInvalid:
			replayed++
		default:
			other++
			t.Errorf("unexpected err: %v", err)
		}
	}
	if ok != 1 {
		t.Fatalf("exactly one consume must succeed, got ok=%d replayed=%d other=%d", ok, replayed, other)
	}
}

func TestCallbackTransaction_MatchesProviderAndActor(t *testing.T) {
	tx := CallbackTransaction{
		ProviderID:         "p1",
		Operation:          ExternalAuthOperationLink,
		ActorUserID:        7,
		OwnerPackageDigest: "digest-xyz",
	}
	if !tx.MatchesProvider("p1", ExternalAuthOperationLink, "digest-xyz") {
		t.Fatalf("provider/operation/digest should match")
	}
	if tx.MatchesProvider("p2", ExternalAuthOperationLink, "digest-xyz") {
		t.Fatalf("provider mismatch should fail")
	}
	if tx.MatchesProvider("p1", ExternalAuthOperationLogin, "digest-xyz") {
		t.Fatalf("operation mismatch should fail")
	}
	if tx.MatchesProvider("p1", ExternalAuthOperationLink, "different") {
		t.Fatalf("digest mismatch should fail")
	}
	// 空 digest 必须 fail closed（禁止与自身空字段“自匹配”）。
	emptyDigest := CallbackTransaction{ProviderID: "p1", Operation: ExternalAuthOperationLink}
	if emptyDigest.MatchesProvider("p1", ExternalAuthOperationLink, "") {
		t.Fatalf("empty digest must not match")
	}
	// link actor 校验
	if !tx.MatchesActor(ExternalAuthOperationLink, 7) {
		t.Fatalf("actor should match for link")
	}
	if tx.MatchesActor(ExternalAuthOperationLink, 8) {
		t.Fatalf("different actor should fail for link")
	}
	// login/registration 必须 actorless
	loginTx := CallbackTransaction{Operation: ExternalAuthOperationLogin, ActorUserID: 0}
	if !loginTx.MatchesActor(ExternalAuthOperationLogin, 0) {
		t.Fatalf("actorless login should match")
	}
}

func TestAbsoluteExternalAuthCallbackURL(t *testing.T) {
	got, err := AbsoluteExternalAuthCallbackURL("https://forum.example.com/site", "Demo.Auth", false)
	if err != nil {
		t.Fatalf("absolute url: %v", err)
	}
	want := "https://forum.example.com/api/v1/auth/providers/demo.auth/callback"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	// 生产非 HTTPS 拒绝。
	if _, err := AbsoluteExternalAuthCallbackURL("http://forum.example.com", "p", true); err == nil {
		t.Fatalf("production http must fail")
	}
	// 缺 scheme/host 拒绝。
	if _, err := AbsoluteExternalAuthCallbackURL("not-a-url", "p", false); err == nil {
		t.Fatalf("invalid app url must fail")
	}
	// 绝不接受相对路径。
	if _, err := AbsoluteExternalAuthCallbackURL("/local", "p", false); err == nil {
		t.Fatalf("relative app url must fail")
	}
}

func TestValidateSafeRedirectPathAndRegistrationContinuation(t *testing.T) {
	if !ValidateSafeRedirectPath("/settings/security") {
		t.Fatalf("local path should be safe")
	}
	for _, bad := range []string{
		"", "https://evil.example/", "//evil.example", "/api/v1/secret", "relative",
	} {
		if ValidateSafeRedirectPath(bad) {
			t.Fatalf("path %q must be rejected", bad)
		}
	}
	path := ExternalRegistrationContinuationPath("ticket-abc", "/topics/1")
	if !strings.HasPrefix(path, "/register?") {
		t.Fatalf("registration continuation must use fixed Host route: %s", path)
	}
	if !strings.Contains(path, "ticket=ticket-abc") {
		t.Fatalf("ticket missing: %s", path)
	}
	if !strings.Contains(path, "redirect=%2Ftopics%2F1") && !strings.Contains(path, "redirect=/topics/1") {
		t.Fatalf("safe redirect must be independent query: %s", path)
	}
	// 危险 redirect 不得进入 query。
	path = ExternalRegistrationContinuationPath("t", "https://evil.example")
	if strings.Contains(path, "redirect=") {
		t.Fatalf("unsafe redirect must not appear: %s", path)
	}
}

func TestInMemoryRegistrationTicketStore_ConsumeOnce(t *testing.T) {
	store := NewInMemoryRegistrationTicketStore()
	ctx := context.Background()
	ticket := validRegistrationTicket("ticket-1", time.Now().Add(time.Minute))
	if err := store.Save(ctx, ticket); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := store.Consume(ctx, "ticket-1")
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if got.ProviderSubject != "raw-subject-123" {
		t.Fatalf("consumed ticket subject mismatch")
	}
	// 重放拒绝（公开契约：invalid）
	if _, err := store.Consume(ctx, "ticket-1"); err != ErrRegistrationTicketInvalid {
		t.Fatalf("ticket replay err = %v, want ErrRegistrationTicketInvalid", err)
	}
}

func TestInMemoryRegistrationTicketStore_Expired(t *testing.T) {
	store := NewInMemoryRegistrationTicketStore()
	ctx := context.Background()
	if err := store.Save(ctx, validRegistrationTicket("expired", time.Now().Add(-time.Second))); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := store.Consume(ctx, "expired"); err != ErrRegistrationTicketExpired {
		t.Fatalf("expired ticket err = %v, want ErrRegistrationTicketExpired", err)
	}
}

func TestInMemoryRegistrationTicketStore_RequiresBindingAndTimestamps(t *testing.T) {
	store := NewInMemoryRegistrationTicketStore()
	ctx := context.Background()

	// 缺 provider/artifact/operation → Save 拒绝。
	if err := store.Save(ctx, RegistrationTicket{
		Token:           "t1",
		ProviderSubject: "x",
		ExpiresAt:       time.Now().Add(time.Minute),
	}); err != ErrRegistrationTicketInvalid {
		t.Fatalf("missing binding save err = %v, want invalid", err)
	}
	// 错误 operation。
	badOp := validRegistrationTicket("t2", time.Now().Add(time.Minute))
	badOp.Operation = ExternalAuthOperationLogin
	if err := store.Save(ctx, badOp); err != ErrRegistrationTicketInvalid {
		t.Fatalf("wrong operation save err = %v, want invalid", err)
	}
	// 合法绑定通过；时间戳由 Save 填充。
	ok := RegistrationTicket{
		Token:              "t3",
		ProviderID:         "p1",
		OwnerExtensionID:   "ext",
		OwnerPackageDigest: strings.Repeat("b", 64),
		Operation:          ExternalAuthOperationRegistration,
		ProviderSubject:    "42",
	}
	if err := store.Save(ctx, ok); err != nil {
		t.Fatalf("valid ticket save: %v", err)
	}
	got, err := store.Consume(ctx, "t3")
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if got.CreatedAt.IsZero() || got.ExpiresAt.IsZero() {
		t.Fatalf("timestamps must be filled on save")
	}
	if got.ExpiresAt.Sub(got.CreatedAt) > RegistrationTicketDefaultTTL+time.Second {
		t.Fatalf("expires must clamp to default TTL")
	}
}

func TestInMemoryRegistrationTicketStore_ConcurrentConsumeOnce(t *testing.T) {
	store := NewInMemoryRegistrationTicketStore()
	ctx := context.Background()
	if err := store.Save(ctx, validRegistrationTicket("race-ticket", time.Now().Add(time.Minute))); err != nil {
		t.Fatalf("save: %v", err)
	}

	const n = 32
	var wg sync.WaitGroup
	errs := make(chan error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, err := store.Consume(ctx, "race-ticket")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	ok := 0
	for err := range errs {
		if err == nil {
			ok++
		}
	}
	if ok != 1 {
		t.Fatalf("exactly one ticket consume must succeed, got %d", ok)
	}
}

func TestGeneratePKCE_S256Challenge(t *testing.T) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("generate pkce: %v", err)
	}
	if len(verifier) < 43 {
		t.Fatalf("verifier too short: %d", len(verifier))
	}
	if challenge == "" {
		t.Fatalf("challenge empty")
	}
	// 不同调用应产生不同 verifier
	v2, _, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("generate pkce 2: %v", err)
	}
	if verifier == v2 {
		t.Fatalf("verifiers must differ")
	}
}

func TestGenerateCallbackState_And_OpaqueToken(t *testing.T) {
	s1, err := GenerateCallbackState()
	if err != nil || len(s1) != 64 {
		t.Fatalf("state = %v len=%d err=%v", s1, len(s1), err)
	}
	tok, err := GenerateOpaqueToken()
	if err != nil || tok == "" || len(tok) < 16 {
		t.Fatalf("token = %v err=%v", tok, err)
	}
	if strings.Contains(tok, "+") || strings.Contains(tok, "/") || strings.Contains(tok, "=") {
		t.Fatalf("opaque token must be base64url without padding: %s", tok)
	}
}

func TestParseExternalAuthOperation(t *testing.T) {
	cases := []struct {
		in   string
		want ExternalAuthOperation
		ok   bool
	}{
		{"login", ExternalAuthOperationLogin, true},
		{"registration", ExternalAuthOperationRegistration, true},
		{"link", ExternalAuthOperationLink, true},
		{"recovery", "", false},
		{"", "", false},
		{"bogus", "", false},
	}
	for _, c := range cases {
		got, err := ParseExternalAuthOperation(c.in)
		if c.ok && err != nil {
			t.Fatalf("parse %q: unexpected err %v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Fatalf("parse %q: expected error, got %v", c.in, got)
		}
		if c.ok && got != c.want {
			t.Fatalf("parse %q: got %v want %v", c.in, got, c.want)
		}
	}
}

func TestOpaqueTokenHash_StableAndNotRaw(t *testing.T) {
	h1 := opaqueTokenHash("abc")
	h2 := opaqueTokenHash("abc")
	if h1 != h2 || len(h1) != 64 {
		t.Fatalf("hash should be stable 64-hex, got %q / %q", h1, h2)
	}
	if h1 == "abc" {
		t.Fatal("hash must not equal raw token")
	}
}

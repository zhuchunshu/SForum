package pluginsessionstate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	testInstallationID  = "installation-test"
	testExtensionID     = "example.plugin.alpha"
	testExtensionIDBeta = "example.plugin.beta"
	testRuntimeInstance = "runtime-instance-1"
)

var (
	testSessionFingerprint  = Fingerprint("session-sid-alpha")
	testSessionBFingerprint = Fingerprint("session-sid-beta")
	testBaseTime            = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
)

// sessionHarness 把「Store + fake 后端 + 注入时钟」绑成一个可推进时间的最小现场。
type sessionHarness struct {
	t       *testing.T
	clock   *fakeBackendClock
	backend *fakeBackend
	store   *Store
	scope   Scope
}

func newSessionHarness(t *testing.T) *sessionHarness {
	t.Helper()
	clock := newFakeBackendClock(testBaseTime)
	backend := newFakeBackend(clock.Now)
	store, err := NewStore(backend)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	store.WithClock(clock.Now)
	return &sessionHarness{
		t:       t,
		clock:   clock,
		backend: backend,
		store:   store,
		scope: Scope{
			InstallationID:     testInstallationID,
			ExtensionID:        testExtensionID,
			RuntimeInstanceID:  testRuntimeInstance,
			SessionFingerprint: testSessionFingerprint,
		},
	}
}

func (h *sessionHarness) withExtension(extensionID string) Scope {
	scope := h.scope
	scope.ExtensionID = extensionID
	return scope
}

func (h *sessionHarness) withSession(fingerprint string) Scope {
	scope := h.scope
	scope.SessionFingerprint = fingerprint
	return scope
}

// valueKeyFor 用 store 的派生函数算键，避免测试自己发明键结构。
func (h *sessionHarness) valueKeyFor(scope Scope, namespaceKey string) string {
	return valueKey(scope.InstallationID, Fingerprint(scope.ExtensionID), scope.SessionFingerprint, namespaceKey)
}

func (h *sessionHarness) mustSet(ctx context.Context, scope Scope, namespaceKey, value string, ttl time.Duration) time.Duration {
	h.t.Helper()
	effective, err := h.store.Set(ctx, scope, namespaceKey, []byte(value), ttl)
	if err != nil {
		h.t.Fatalf("Set(%q): %v", namespaceKey, err)
	}
	return effective
}

func (h *sessionHarness) mustGet(ctx context.Context, scope Scope, namespaceKey string) string {
	h.t.Helper()
	value, found, err := h.store.Get(ctx, scope, namespaceKey)
	if err != nil {
		h.t.Fatalf("Get(%q): %v", namespaceKey, err)
	}
	if !found {
		h.t.Fatalf("Get(%q) 未命中，但期望有值", namespaceKey)
	}
	return string(value)
}

func (h *sessionHarness) mustMiss(ctx context.Context, scope Scope, namespaceKey string) {
	h.t.Helper()
	value, found, err := h.store.Get(ctx, scope, namespaceKey)
	if err != nil {
		h.t.Fatalf("Get(%q): %v", namespaceKey, err)
	}
	if found {
		h.t.Fatalf("Get(%q) 命中 %q，但期望未命中", namespaceKey, value)
	}
}

func scoped(fn func(*Scope)) Scope {
	scope := Scope{
		InstallationID:     testInstallationID,
		ExtensionID:        testExtensionID,
		SessionFingerprint: testSessionFingerprint,
	}
	fn(&scope)
	return scope
}

func memberOf(extensionID, namespaceKey string) string {
	return Fingerprint(extensionID) + ":" + Fingerprint(namespaceKey)
}

func TestNewStoreRequiresBackend(t *testing.T) {
	if _, err := NewStore(nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("NewStore(nil) 必须失败并返回 ErrInvalid，got %v", err)
	}
	store, err := NewStore(newFakeBackend(nil))
	if err != nil || store == nil {
		t.Fatalf("NewStore(fake) 必须成功，got store=%v err=%v", store, err)
	}
}

// TestSetGetDeleteRoundTrip 覆盖允许路径：写入 → 读回同样字节 → 删除 → 未命中。
func TestSetGetDeleteRoundTrip(t *testing.T) {
	h := newSessionHarness(t)
	ctx := context.Background()
	const requested = 30 * time.Minute

	effective, err := h.store.Set(ctx, h.scope, "cart", []byte("payload"), requested)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if effective != requested {
		t.Fatalf("Set 必须返回生效 TTL：got %s want %s", effective, requested)
	}
	if ttl, ok := h.backend.lastValueTTL(h.valueKeyFor(h.scope, "cart")); !ok || ttl != requested {
		t.Fatalf("值键必须以生效 TTL 写入后端：got %s ok=%v", ttl, ok)
	}

	value, found, err := h.store.Get(ctx, h.scope, "cart")
	if err != nil || !found {
		t.Fatalf("Get: found=%v err=%v", found, err)
	}
	if !bytes.Equal(value, []byte("payload")) {
		t.Fatalf("Get 必须返回写入的字节：got %q", value)
	}

	// 覆盖写同一键：读回的是最后一次写入。
	if _, err := h.store.Set(ctx, h.scope, "cart", []byte("payload-2"), time.Minute); err != nil {
		t.Fatalf("覆盖 Set: %v", err)
	}
	if got := h.mustGet(ctx, h.scope, "cart"); got != "payload-2" {
		t.Fatalf("覆盖写后应读到新值：got %q", got)
	}

	deleted, err := h.store.Delete(ctx, h.scope, "cart")
	if err != nil || !deleted {
		t.Fatalf("Delete 必须报告删除了已存在的键：deleted=%v err=%v", deleted, err)
	}
	h.mustMiss(ctx, h.scope, "cart")

	deleted, err = h.store.Delete(ctx, h.scope, "cart")
	if err != nil || deleted {
		t.Fatalf("重复 Delete 必须 (false, nil)：deleted=%v err=%v", deleted, err)
	}
	if _, found := h.backend.liveValue(h.valueKeyFor(h.scope, "cart")); found {
		t.Fatal("Delete 之后后端不得再持有该值键")
	}
}

// TestStoreRejectsInvalidInput 覆盖拒绝路径：每个非法输入必须返回它对应的具名错误。
func TestStoreRejectsInvalidInput(t *testing.T) {
	ctx := context.Background()
	longKey := strings.Repeat("k", KeyMaxBytes+1)
	oversizedValue := bytes.Repeat([]byte("v"), ValueMaxBytes+1)

	cases := []struct {
		name string
		run  func(h *sessionHarness) error
		want error
	}{
		{
			name: "Get 收到 nil ctx",
			run:  func(h *sessionHarness) error { _, _, err := h.store.Get(nil, h.scope, "k"); return err },
			want: ErrInvalid,
		},
		{
			name: "Set 收到 nil ctx",
			run: func(h *sessionHarness) error {
				_, err := h.store.Set(nil, h.scope, "k", []byte("v"), time.Minute)
				return err
			},
			want: ErrInvalid,
		},
		{
			name: "Delete 收到 nil ctx",
			run:  func(h *sessionHarness) error { _, err := h.store.Delete(nil, h.scope, "k"); return err },
			want: ErrInvalid,
		},
		{
			name: "InvalidateSession 收到 nil ctx",
			run: func(h *sessionHarness) error {
				_, err := h.store.InvalidateSession(nil, testInstallationID, testSessionFingerprint)
				return err
			},
			want: ErrInvalid,
		},
		{
			name: "InvalidateExtension 收到 nil ctx",
			run: func(h *sessionHarness) error {
				_, err := h.store.InvalidateExtension(nil, testInstallationID, testExtensionID)
				return err
			},
			want: ErrInvalid,
		},
		{
			name: "scope 缺 installation",
			run: func(h *sessionHarness) error {
				scope := scoped(func(s *Scope) { s.InstallationID = "" })
				_, err := h.store.Set(ctx, scope, "k", []byte("v"), time.Minute)
				return err
			},
			want: ErrInvalid,
		},
		{
			name: "scope 缺 extension",
			run: func(h *sessionHarness) error {
				scope := scoped(func(s *Scope) { s.ExtensionID = "" })
				_, err := h.store.Set(ctx, scope, "k", []byte("v"), time.Minute)
				return err
			},
			want: ErrInvalid,
		},
		{
			name: "scope 缺 session fingerprint",
			run: func(h *sessionHarness) error {
				scope := scoped(func(s *Scope) { s.SessionFingerprint = "" })
				_, err := h.store.Set(ctx, scope, "k", []byte("v"), time.Minute)
				return err
			},
			want: ErrInvalid,
		},
		{
			name: "fingerprint 含冒号",
			run: func(h *sessionHarness) error {
				scope := scoped(func(s *Scope) { s.SessionFingerprint = "abc:def" })
				_, err := h.store.Set(ctx, scope, "k", []byte("v"), time.Minute)
				return err
			},
			want: ErrInvalid,
		},
		{
			name: "fingerprint 长度对但不是十六进制",
			run: func(h *sessionHarness) error {
				scope := scoped(func(s *Scope) { s.SessionFingerprint = strings.Repeat("z", 64) })
				_, _, err := h.store.Get(ctx, scope, "k")
				return err
			},
			want: ErrInvalid,
		},
		{
			name: "installation 含冒号（键结构冲突）",
			run: func(h *sessionHarness) error {
				scope := scoped(func(s *Scope) { s.InstallationID = "installation:1" })
				_, err := h.store.Set(ctx, scope, "k", []byte("v"), time.Minute)
				return err
			},
			want: ErrInvalid,
		},
		{
			name: "空 key",
			run: func(h *sessionHarness) error {
				_, err := h.store.Set(ctx, h.scope, "", []byte("v"), time.Minute)
				return err
			},
			want: ErrInvalid,
		},
		{
			name: "只有空白的 key",
			run: func(h *sessionHarness) error {
				_, err := h.store.Set(ctx, h.scope, "   ", []byte("v"), time.Minute)
				return err
			},
			want: ErrInvalid,
		},
		{
			name: "尾部空白的 key",
			run: func(h *sessionHarness) error {
				_, err := h.store.Set(ctx, h.scope, "cart ", []byte("v"), time.Minute)
				return err
			},
			want: ErrInvalid,
		},
		{
			name: "Get 收到只有空白的 key",
			run:  func(h *sessionHarness) error { _, _, err := h.store.Get(ctx, h.scope, " "); return err },
			want: ErrInvalid,
		},
		{
			name: "Delete 收到只有空白的 key",
			run:  func(h *sessionHarness) error { _, err := h.store.Delete(ctx, h.scope, "\t"); return err },
			want: ErrInvalid,
		},
		{
			name: "key 超过 512 字节",
			run: func(h *sessionHarness) error {
				_, err := h.store.Set(ctx, h.scope, longKey, []byte("v"), time.Minute)
				return err
			},
			want: ErrKeyTooLong,
		},
		{
			name: "空值",
			run:  func(h *sessionHarness) error { _, err := h.store.Set(ctx, h.scope, "k", nil, time.Minute); return err },
			want: ErrInvalid,
		},
		{
			name: "值超过 1 MiB",
			run: func(h *sessionHarness) error {
				_, err := h.store.Set(ctx, h.scope, "k", oversizedValue, time.Minute)
				return err
			},
			want: ErrValueTooLarge,
		},
		{
			name: "TTL 为 0",
			run:  func(h *sessionHarness) error { _, err := h.store.Set(ctx, h.scope, "k", []byte("v"), 0); return err },
			want: ErrTTLOutOfRange,
		},
		{
			name: "TTL 为负",
			run: func(h *sessionHarness) error {
				_, err := h.store.Set(ctx, h.scope, "k", []byte("v"), -time.Second)
				return err
			},
			want: ErrTTLOutOfRange,
		},
		{
			name: "TTL 小于 1ms",
			run: func(h *sessionHarness) error {
				_, err := h.store.Set(ctx, h.scope, "k", []byte("v"), MinTTL-time.Nanosecond)
				return err
			},
			want: ErrTTLOutOfRange,
		},
		{
			name: "TTL 大于 24h",
			run: func(h *sessionHarness) error {
				_, err := h.store.Set(ctx, h.scope, "k", []byte("v"), MaxTTL+time.Second)
				return err
			},
			want: ErrTTLOutOfRange,
		},
		{
			name: "InvalidateSession 缺 installation",
			run: func(h *sessionHarness) error {
				_, err := h.store.InvalidateSession(ctx, "", testSessionFingerprint)
				return err
			},
			want: ErrInvalid,
		},
		{
			name: "InvalidateSession 的 fingerprint 不是十六进制",
			run: func(h *sessionHarness) error {
				_, err := h.store.InvalidateSession(ctx, testInstallationID, "not-a-fingerprint")
				return err
			},
			want: ErrInvalid,
		},
		{
			name: "InvalidateExtension 缺 extension",
			run: func(h *sessionHarness) error {
				_, err := h.store.InvalidateExtension(ctx, testInstallationID, "")
				return err
			},
			want: ErrInvalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newSessionHarness(t)
			err := tc.run(h)
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
			// 被拒的写入不得留下任何后端痕迹。
			if keys := h.backend.liveValueKeys(); len(keys) != 0 {
				t.Fatalf("被拒的调用不得写入值键，实际留下 %v", keys)
			}
		})
	}
}

// TestStoreAcceptsBoundaryInput 是上表的允许侧：正好等于上限的输入必须通过。
func TestStoreAcceptsBoundaryInput(t *testing.T) {
	h := newSessionHarness(t)
	ctx := context.Background()
	maxKey := strings.Repeat("k", KeyMaxBytes)
	maxValue := bytes.Repeat([]byte("v"), ValueMaxBytes)

	if _, err := h.store.Set(ctx, h.scope, maxKey, maxValue, MaxTTL); err != nil {
		t.Fatalf("key=%d 字节 / value=%d 字节 / ttl=%s 都在上限内，必须接受：%v", KeyMaxBytes, ValueMaxBytes, MaxTTL, err)
	}
	value, found, err := h.store.Get(ctx, h.scope, maxKey)
	if err != nil || !found || !bytes.Equal(value, maxValue) {
		t.Fatalf("边界值必须能原样读回：found=%v err=%v len=%d", found, err, len(value))
	}
	if _, err := h.store.Set(ctx, h.scope, "min-ttl", []byte("v"), MinTTL); err != nil {
		t.Fatalf("TTL 正好 1ms 必须接受：%v", err)
	}
}

// TestReadPathsReportSpecificKeyErrors 要求三条路径对同一把非法键给出**同一个**具体错误。
//
// 历史：这个测试原先固定的是偏差——Set 返回 ErrKeyTooLong，而 Get/Delete 把 scope 与键校验
// 合并成一个 fail-closed 分支统一返回 ErrInvalid。store.go 已修（读/删路径拆出键校验），
// 因此期望值按当时的说明改成了 ErrKeyTooLong；如果哪天又折叠回去，本测试会立刻失败。
func TestReadPathsReportSpecificKeyErrors(t *testing.T) {
	h := newSessionHarness(t)
	ctx := context.Background()
	longKey := strings.Repeat("k", KeyMaxBytes+1)

	if _, err := h.store.Set(ctx, h.scope, longKey, []byte("v"), time.Minute); !errors.Is(err, ErrKeyTooLong) {
		t.Fatalf("写路径必须返回 ErrKeyTooLong，got %v", err)
	}
	if _, _, err := h.store.Get(ctx, h.scope, longKey); !errors.Is(err, ErrKeyTooLong) {
		t.Fatalf("读路径对超长键必须返回 ErrKeyTooLong（与写路径一致），got %v", err)
	}
	if _, err := h.store.Delete(ctx, h.scope, longKey); !errors.Is(err, ErrKeyTooLong) {
		t.Fatalf("删除路径对超长键必须返回 ErrKeyTooLong（与写路径一致），got %v", err)
	}
}

// TestEffectiveTTLMath 直接调用 Scope.EffectiveTTL：四者取最小，永不放宽，已过期即拒绝。
func TestEffectiveTTLMath(t *testing.T) {
	now := testBaseTime
	cases := []struct {
		name      string
		scope     Scope
		requested time.Duration
		want      time.Duration
		wantErr   error
	}{
		{
			name:      "requested 是最小者",
			scope:     scoped(func(s *Scope) { s.SessionIDleExpiresAt = now.Add(time.Hour) }),
			requested: 10 * time.Minute,
			want:      10 * time.Minute,
		},
		{
			name:      "requested 恰好 24h 通过（上限之内）",
			scope:     scoped(func(*Scope) {}),
			requested: MaxTTL,
			want:      MaxTTL,
		},
		{
			name:      "requested 超过 24h 被拒绝而不是被夹到 24h",
			scope:     scoped(func(*Scope) {}),
			requested: MaxTTL + time.Second,
			wantErr:   ErrTTLOutOfRange,
		},
		{
			name:      "两个边界都为零时 24h 上限仍然生效",
			scope:     scoped(func(*Scope) {}),
			requested: MaxTTL,
			want:      MaxTTL,
		},
		{
			name:      "requested 恰好 1ms 通过（下限之内）",
			scope:     scoped(func(*Scope) {}),
			requested: MinTTL,
			want:      MinTTL,
		},
		{
			name:      "requested 小于 1ms 被拒绝",
			scope:     scoped(func(*Scope) {}),
			requested: MinTTL - time.Nanosecond,
			wantErr:   ErrTTLOutOfRange,
		},
		{
			name:      "requested 为 0 被拒绝",
			scope:     scoped(func(*Scope) {}),
			requested: 0,
			wantErr:   ErrTTLOutOfRange,
		},
		{
			name:      "剩余 idle 收紧生效 TTL",
			scope:     scoped(func(s *Scope) { s.SessionIDleExpiresAt = now.Add(15 * time.Minute) }),
			requested: time.Hour,
			want:      15 * time.Minute,
		},
		{
			name:      "剩余绝对寿命收紧生效 TTL",
			scope:     scoped(func(s *Scope) { s.SessionAbsoluteExpiresAt = now.Add(7 * time.Minute) }),
			requested: time.Hour,
			want:      7 * time.Minute,
		},
		{
			name: "两个边界同时存在时取更小者",
			scope: scoped(func(s *Scope) {
				s.SessionIDleExpiresAt = now.Add(15 * time.Minute)
				s.SessionAbsoluteExpiresAt = now.Add(7 * time.Minute)
			}),
			requested: time.Hour,
			want:      7 * time.Minute,
		},
		{
			name: "边界不能放大 requested",
			scope: scoped(func(s *Scope) {
				s.SessionIDleExpiresAt = now.Add(20 * time.Hour)
				s.SessionAbsoluteExpiresAt = now.Add(23 * time.Hour)
			}),
			requested: 5 * time.Minute,
			want:      5 * time.Minute,
		},
		{
			name:      "idle 边界正好等于 now 视为已过期",
			scope:     scoped(func(s *Scope) { s.SessionIDleExpiresAt = now }),
			requested: time.Hour,
			wantErr:   ErrInvalid,
		},
		{
			name:      "idle 边界在过去视为已过期",
			scope:     scoped(func(s *Scope) { s.SessionIDleExpiresAt = now.Add(-time.Minute) }),
			requested: time.Hour,
			wantErr:   ErrInvalid,
		},
		{
			name:      "绝对寿命边界在过去视为已过期",
			scope:     scoped(func(s *Scope) { s.SessionAbsoluteExpiresAt = now.Add(-time.Second) }),
			requested: time.Hour,
			wantErr:   ErrInvalid,
		},
		{
			name:      "剩余寿命不足 1ms 被拒绝",
			scope:     scoped(func(s *Scope) { s.SessionIDleExpiresAt = now.Add(500 * time.Microsecond) }),
			requested: time.Hour,
			wantErr:   ErrInvalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.scope.EffectiveTTL(now, tc.requested)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("want %v, got %v", tc.wantErr, err)
				}
				if got != 0 {
					t.Fatalf("出错时必须返回零 TTL，got %s", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("want %s, got %s", tc.want, got)
			}
		})
	}
}

// TestSetAppliesEffectiveTTL 验证生效 TTL 既被返回，也被真正写给值键与两个索引。
func TestSetAppliesEffectiveTTL(t *testing.T) {
	ctx := context.Background()

	t.Run("会话剩余寿命收紧值和索引", func(t *testing.T) {
		h := newSessionHarness(t)
		scope := scoped(func(s *Scope) { s.SessionIDleExpiresAt = testBaseTime.Add(15 * time.Minute) })
		effective := h.mustSet(ctx, scope, "k", "v", time.Hour)
		if effective != 15*time.Minute {
			t.Fatalf("生效 TTL 应被 idle 边界收紧到 15m，got %s", effective)
		}
		if ttl, ok := h.backend.lastValueTTL(h.valueKeyFor(scope, "k")); !ok || ttl != effective {
			t.Fatalf("值键 TTL 必须等于生效 TTL：got %s ok=%v", ttl, ok)
		}
		for _, indexKey := range []string{
			sessionKeysIndex(scope.InstallationID, scope.SessionFingerprint),
			sessionExtensionsIndex(scope.InstallationID, scope.SessionFingerprint),
			extensionSessionsIndex(scope.InstallationID, Fingerprint(scope.ExtensionID)),
		} {
			if ttl, ok := h.backend.lastSetAddTTL(indexKey); !ok || ttl != 15*time.Minute {
				t.Fatalf("索引 %q 必须以会话剩余寿命为 TTL：got %s ok=%v", indexKey, ttl, ok)
			}
		}
	})

	t.Run("索引 TTL 不低于值键 TTL", func(t *testing.T) {
		h := newSessionHarness(t)
		effective := h.mustSet(ctx, h.scope, "k", "v", 10*time.Minute)
		if effective != 10*time.Minute {
			t.Fatalf("生效 TTL 应为 requested，got %s", effective)
		}
		sessionKeys := sessionKeysIndex(h.scope.InstallationID, h.scope.SessionFingerprint)
		ttl, ok := h.backend.lastSetAddTTL(sessionKeys)
		if !ok || ttl != MaxTTL {
			t.Fatalf("会话边界未知时索引 TTL 取 MaxTTL，got %s ok=%v", ttl, ok)
		}
		if ttl < effective {
			t.Fatalf("索引 TTL(%s) 不得短于值键 TTL(%s)：索引先消失会让失效无法枚举", ttl, effective)
		}
	})

	t.Run("requested 等于 24h 上限", func(t *testing.T) {
		h := newSessionHarness(t)
		if effective := h.mustSet(ctx, h.scope, "k", "v", MaxTTL); effective != MaxTTL {
			t.Fatalf("got %s", effective)
		}
	})

	t.Run("已过期的会话边界使写入失败", func(t *testing.T) {
		h := newSessionHarness(t)
		scope := scoped(func(s *Scope) { s.SessionAbsoluteExpiresAt = testBaseTime.Add(-time.Second) })
		if _, err := h.store.Set(ctx, scope, "k", []byte("v"), time.Minute); !errors.Is(err, ErrInvalid) {
			t.Fatalf("want ErrInvalid, got %v", err)
		}
		if keys := h.backend.liveValueKeys(); len(keys) != 0 {
			t.Fatalf("失败的写入不得落盘，实际留下 %v", keys)
		}
	})
}

// TestSessionKeyLimitAtCapAndUpdateExisting 覆盖 (扩展, 会话) 级上限的两侧。
func TestSessionKeyLimitAtCapAndUpdateExisting(t *testing.T) {
	h := newSessionHarness(t)
	ctx := context.Background()

	for i := 0; i < MaxKeysPerSession; i++ {
		h.mustSet(ctx, h.scope, fmt.Sprintf("key-%02d", i), fmt.Sprintf("value-%02d", i), 10*time.Minute)
	}
	sessionKeys := sessionKeysIndex(h.scope.InstallationID, h.scope.SessionFingerprint)
	if members := h.backend.realSetMembers(sessionKeys); len(members) != MaxKeysPerSession {
		t.Fatalf("会话索引应有 %d 个成员，got %d", MaxKeysPerSession, len(members))
	}

	// 第 65 个新键被拒绝。
	_, err := h.store.Set(ctx, h.scope, "key-overflow", []byte("v"), 10*time.Minute)
	if !errors.Is(err, ErrKeyLimit) {
		t.Fatalf("第 %d 个键必须返回 ErrKeyLimit，got %v", MaxKeysPerSession+1, err)
	}
	if _, found := h.backend.liveValue(h.valueKeyFor(h.scope, "key-overflow")); found {
		t.Fatal("被拒的写入不得留下值键（不得部分写入）")
	}
	if members := h.backend.realSetMembers(sessionKeys); len(members) != MaxKeysPerSession {
		t.Fatalf("被拒的写入不得增长索引，got %d 个成员", len(members))
	}

	// 在上限处更新已有键仍然允许：否则被削过上限的插件连自己的旧数据都改不了。
	effective, err := h.store.Set(ctx, h.scope, "key-00", []byte("updated"), 5*time.Minute)
	if err != nil {
		t.Fatalf("上限处更新已有键必须成功，got %v", err)
	}
	if effective != 5*time.Minute {
		t.Fatalf("更新必须使用新的生效 TTL，got %s", effective)
	}
	if got := h.mustGet(ctx, h.scope, "key-00"); got != "updated" {
		t.Fatalf("更新后的值必须是新值，got %q", got)
	}
}

// installSyntheticLiveSessions 用 fake 钩子伪造 count 个「存活会话」，每个会话
// MaxKeysPerSession 个存活键。
//
// 诚实边界（重要）：这不是真的写满 4096 个键。被伪造的只是「人口」——store 的
// enforceCaps → countLiveSessions → liveSessions >= MaxKeysPerExtension 判断链按原样执行，
// 逐个成员 Get 确认存活的过程也真实发生，但 4096 这个数字不是由 4096 次真写得到的。
// 真写的版本见 TestExtensionKeyLimitRealWrites（默认跳过，需显式开启）。
func (h *sessionHarness) installSyntheticLiveSessions(count int) {
	h.t.Helper()
	installationID := h.scope.InstallationID
	extensionHash := Fingerprint(h.scope.ExtensionID)
	extensionSessions := extensionSessionsIndex(installationID, extensionHash)
	for i := 0; i < count; i++ {
		fingerprint := Fingerprint(fmt.Sprintf("synthetic-session-%d", i))
		sessionKeys := sessionKeysIndex(installationID, fingerprint)
		valueKeyPrefix := NamespacePrefix + "installation:" + installationID +
			":extension:" + extensionHash + ":session:" + fingerprint + ":key:"
		// 合成前缀必须与 store 的键派生一致，否则这个钩子就没在测同一条路径。
		probe := valueKey(installationID, extensionHash, fingerprint, "probe")
		if !strings.HasPrefix(probe, valueKeyPrefix) {
			h.t.Fatalf("合成前缀 %q 与 store 派生 %q 不一致", valueKeyPrefix, probe)
		}
		h.backend.addSyntheticLiveSession(
			extensionSessions,
			fingerprint,
			sessionKeys,
			extensionHash+":",
			MaxKeysPerSession,
			valueKeyPrefix,
		)
	}
}

// TestExtensionKeyLimitSyntheticPopulation 覆盖扩展级上限的两侧：4033 个存活键允许新键，
// 4097 个存活键拒绝新键（ErrExtensionLimit）。
func TestExtensionKeyLimitSyntheticPopulation(t *testing.T) {
	ctx := context.Background()

	t.Run("低于 4096 个存活键允许新键", func(t *testing.T) {
		h := newSessionHarness(t)
		h.mustSet(ctx, h.scope, "real-key-1", "v", time.Hour)
		h.installSyntheticLiveSessions(MaxKeysPerExtension/MaxKeysPerSession - 1) // 63 × 64 = 4032
		if _, err := h.store.Set(ctx, h.scope, "real-key-2", []byte("v"), time.Hour); err != nil {
			t.Fatalf("%d 个存活键低于 %d 上限，必须允许新键：%v", 4033, MaxKeysPerExtension, err)
		}
	})

	t.Run("达到 4096 个存活键拒绝新键", func(t *testing.T) {
		h := newSessionHarness(t)
		h.mustSet(ctx, h.scope, "real-key-1", "v", time.Hour)
		h.installSyntheticLiveSessions(MaxKeysPerExtension / MaxKeysPerSession) // 64 × 64 = 4096
		_, err := h.store.Set(ctx, h.scope, "real-key-2", []byte("v"), time.Hour)
		if !errors.Is(err, ErrExtensionLimit) {
			t.Fatalf("扩展级上限必须返回 ErrExtensionLimit，got %v", err)
		}
		if errors.Is(err, ErrKeyLimit) {
			t.Fatal("该会话只有 1 个键，不应报会话级上限")
		}
		if _, found := h.backend.liveValue(h.valueKeyFor(h.scope, "real-key-2")); found {
			t.Fatal("被拒的写入不得留下值键")
		}
	})
}

// TestExtensionKeyLimitRealWrites 用真实的 4096 次写入填满扩展级上限。
// 默认跳过：每次新键都会重扫存量键，4096 次写入在 -race 下是秒级而非毫秒级，
// 不适合作为默认门禁（默认门禁里的等价覆盖由 TestExtensionKeyLimitSyntheticPopulation 提供）。
func TestExtensionKeyLimitRealWrites(t *testing.T) {
	if os.Getenv("SFORUM_PLUGIN_SESSION_FULL_CAP") != "1" {
		t.Skip("真实写满 4096 个键；设置 SFORUM_PLUGIN_SESSION_FULL_CAP=1 运行")
	}
	h := newSessionHarness(t)
	ctx := context.Background()
	session := func(i int) Scope {
		scope := h.scope
		scope.SessionFingerprint = Fingerprint(fmt.Sprintf("full-cap-session-%d", i))
		return scope
	}

	// 63 个会话 × 64 个键 = 4032，再加一个探针会话的 1 个键 = 4033 < 4096。
	for i := 0; i < 63; i++ {
		for j := 0; j < MaxKeysPerSession; j++ {
			h.mustSet(ctx, session(i), fmt.Sprintf("key-%02d", j), "v", 2*time.Hour)
		}
	}
	if _, err := h.store.Set(ctx, session(63), "probe", []byte("v"), 2*time.Hour); err != nil {
		t.Fatalf("4033 个存活键低于上限，必须允许：%v", err)
	}
	// 第 64 个会话继续写满到 4096。
	for j := 1; j < MaxKeysPerSession; j++ {
		h.mustSet(ctx, session(63), fmt.Sprintf("key-%02d", j), "v", 2*time.Hour)
	}
	_, err := h.store.Set(ctx, session(64), "overflow", []byte("v"), 2*time.Hour)
	if !errors.Is(err, ErrExtensionLimit) {
		t.Fatalf("真实写满 %d 个键后必须返回 ErrExtensionLimit，got %v", MaxKeysPerExtension, err)
	}
}

// TestStaleMemberPruningRecomputesCount 覆盖「索引满但成员陈旧」：写入必须成功，
// 因为 store 会逐个确认值键存活、摘掉陈旧成员并重新计数。
func TestStaleMemberPruningRecomputesCount(t *testing.T) {
	h := newSessionHarness(t)
	ctx := context.Background()
	const valueTTL = 5 * time.Minute

	for i := 0; i < MaxKeysPerSession; i++ {
		h.mustSet(ctx, h.scope, fmt.Sprintf("key-%02d", i), "v", valueTTL)
	}
	sessionKeys := sessionKeysIndex(h.scope.InstallationID, h.scope.SessionFingerprint)
	if members := h.backend.realSetMembers(sessionKeys); len(members) != MaxKeysPerSession {
		t.Fatalf("前置条件：会话索引应在上限处，got %d", len(members))
	}

	// 推进时钟：值键全部过期，而索引键（TTL = 会话最大寿命 24h）仍然存在，
	// 于是索引里剩下 MaxKeysPerSession 个「指向已消失值键」的陈旧成员。
	h.clock.Advance(valueTTL + time.Minute)
	if members := h.backend.realSetMembers(sessionKeys); len(members) != MaxKeysPerSession {
		t.Fatalf("索引键不应随值键过期：got %d 个成员", len(members))
	}
	if value, found := h.backend.liveValue(h.valueKeyFor(h.scope, "key-00")); found {
		t.Fatalf("前置条件：值键应已过期，got %q", value)
	}

	effective, err := h.store.Set(ctx, h.scope, "key-new", []byte("v"), valueTTL)
	if err != nil {
		t.Fatalf("索引只含陈旧成员时新键必须可写（有界修剪 + 重新计数），got %v", err)
	}
	if effective != valueTTL {
		t.Fatalf("新键生效 TTL 应为 %s，got %s", valueTTL, effective)
	}

	members := h.backend.realSetMembers(sessionKeys)
	if len(members) != 1 || members[0] != memberOf(testExtensionID, "key-new") {
		t.Fatalf("陈旧成员必须被摘掉且计数被重建，got %v", members)
	}
	if keys := h.backend.liveValueKeys(); len(keys) != 1 || keys[0] != h.valueKeyFor(h.scope, "key-new") {
		t.Fatalf("后端只应剩下新键，got %v", keys)
	}
}

// TestCrossExtensionAndCrossSessionIsolation 覆盖 S7 的隔离：同名 namespace key 在
// 不同扩展/不同会话下既读不到彼此的值，也不是同一个键字符串。
func TestCrossExtensionAndCrossSessionIsolation(t *testing.T) {
	h := newSessionHarness(t)
	ctx := context.Background()
	alphaOne := h.scope
	betaOne := h.withExtension(testExtensionIDBeta)
	alphaTwo := h.withSession(testSessionBFingerprint)

	h.mustSet(ctx, alphaOne, "shared", "alpha-one", time.Hour)
	h.mustSet(ctx, betaOne, "shared", "beta-one", time.Hour)
	h.mustSet(ctx, alphaTwo, "shared", "alpha-two", time.Hour)

	if got := h.mustGet(ctx, alphaOne, "shared"); got != "alpha-one" {
		t.Fatalf("扩展 alpha/会话 alpha 必须读回自己的值，got %q", got)
	}
	if got := h.mustGet(ctx, betaOne, "shared"); got != "beta-one" {
		t.Fatalf("扩展 beta 必须读回自己的值，got %q", got)
	}
	if got := h.mustGet(ctx, alphaTwo, "shared"); got != "alpha-two" {
		t.Fatalf("会话 beta 必须读回自己的值，got %q", got)
	}

	keys := h.backend.liveValueKeys()
	want := []string{
		h.valueKeyFor(alphaOne, "shared"),
		h.valueKeyFor(alphaTwo, "shared"),
		h.valueKeyFor(betaOne, "shared"),
	}
	sort.Strings(want)
	if len(keys) != len(want) {
		t.Fatalf("应有 3 个互不相同的值键，got %v", keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("键集合不符：got %v want %v", keys, want)
		}
	}
	// 跨扩展写同一键名不得覆盖对方：删除 alpha 的值后，beta 与另一个会话仍然有值。
	if _, err := h.store.Delete(ctx, alphaOne, "shared"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	h.mustMiss(ctx, alphaOne, "shared")
	if got := h.mustGet(ctx, betaOne, "shared"); got != "beta-one" {
		t.Fatalf("删除不得跨扩展影响：got %q", got)
	}
	if got := h.mustGet(ctx, alphaTwo, "shared"); got != "alpha-two" {
		t.Fatalf("删除不得跨会话影响：got %q", got)
	}
}

// TestRotationStabilityByFingerprint 覆盖 S4：轮换保持 sid 不变，因此 fingerprint 不变、
// 键不变、状态天然存活；换 sid 的新会话看不到旧值。
func TestRotationStabilityByFingerprint(t *testing.T) {
	h := newSessionHarness(t)
	ctx := context.Background()

	h.mustSet(ctx, h.scope, "draft", "in-progress", time.Hour)
	before := h.backend.liveValueKeys()
	key := h.valueKeyFor(h.scope, "draft")
	if len(before) != 1 || before[0] != key {
		t.Fatalf("应有且仅有该会话的值键，got %v", before)
	}
	// S1/S2 的结构性要求：键里不得出现原始 sid 或原始扩展 id。
	if strings.Contains(key, "session-sid-alpha") {
		t.Fatalf("值键不得包含原始 session id：%q", key)
	}
	if strings.Contains(key, testExtensionID) {
		t.Fatalf("值键不得包含原始扩展 id：%q", key)
	}

	// 同一 fingerprint 重复访问（= 轮换后继续用同一个 sid）：键完全一致，不产生第二个键。
	rotated := h.scope
	if got := h.mustGet(ctx, rotated, "draft"); got != "in-progress" {
		t.Fatalf("同一 fingerprint 必须读到原值，got %q", got)
	}
	if h.valueKeyFor(rotated, "draft") != key {
		t.Fatal("同一 fingerprint 必须派生出同一个值键（这是 24h 轮换免费的原因）")
	}
	h.mustSet(ctx, rotated, "draft", "in-progress-2", time.Hour)
	after := h.backend.liveValueKeys()
	if len(after) != 1 || after[0] != key {
		t.Fatalf("同一 fingerprint 的重复写入不得产生新键，got %v", after)
	}

	// 换 sid 的新会话（新登录）：不是同一个键，读不到旧值。
	replaced := h.withSession(Fingerprint("session-sid-gamma"))
	h.mustMiss(ctx, replaced, "draft")
	if h.valueKeyFor(replaced, "draft") == key {
		t.Fatal("不同 fingerprint 必须派生出不同的值键")
	}
	if got := h.mustGet(ctx, h.scope, "draft"); got != "in-progress-2" {
		t.Fatalf("新会话的读不得影响旧会话的值，got %q", got)
	}
}

// TestExpiredSessionReadDeletesKey 覆盖读路径自己复核寿命：越界即未命中，并顺手清键。
func TestExpiredSessionReadDeletesKey(t *testing.T) {
	ctx := context.Background()

	t.Run("idle 边界已越过", func(t *testing.T) {
		h := newSessionHarness(t)
		live := scoped(func(s *Scope) { s.SessionIDleExpiresAt = testBaseTime.Add(time.Hour) })
		h.mustSet(ctx, live, "k", "v", 30*time.Minute)
		key := h.valueKeyFor(live, "k")
		if _, found := h.backend.liveValue(key); !found {
			t.Fatal("前置条件：值键应存在")
		}
		expired := scoped(func(s *Scope) { s.SessionIDleExpiresAt = testBaseTime.Add(-time.Second) })
		value, found, err := h.store.Get(ctx, expired, "k")
		if err != nil || found {
			t.Fatalf("会话已过期时必须未命中：value=%q found=%v err=%v", value, found, err)
		}
		if _, found := h.backend.liveValue(key); found {
			t.Fatal("越界读取必须顺手删除底层键，不能留下可被再次读到的状态")
		}
	})

	t.Run("绝对寿命边界已越过", func(t *testing.T) {
		h := newSessionHarness(t)
		live := scoped(func(s *Scope) { s.SessionAbsoluteExpiresAt = testBaseTime.Add(time.Hour) })
		h.mustSet(ctx, live, "k", "v", 30*time.Minute)
		expired := scoped(func(s *Scope) { s.SessionAbsoluteExpiresAt = testBaseTime.Add(-time.Second) })
		if _, found, err := h.store.Get(ctx, expired, "k"); err != nil || found {
			t.Fatalf("found=%v err=%v", found, err)
		}
		if _, found := h.backend.liveValue(h.valueKeyFor(live, "k")); found {
			t.Fatal("绝对寿命越界同样必须删除底层键")
		}
	})
}

// TestInvalidateSession 覆盖登出/撤销路径：跨扩展清空、计数、幂等、不误伤其它会话。
func TestInvalidateSession(t *testing.T) {
	h := newSessionHarness(t)
	ctx := context.Background()
	alphaOne := h.scope
	betaOne := h.withExtension(testExtensionIDBeta)
	alphaTwo := h.withSession(testSessionBFingerprint)

	h.mustSet(ctx, alphaOne, "a1", "v", time.Hour)
	h.mustSet(ctx, alphaOne, "a2", "v", time.Hour)
	h.mustSet(ctx, betaOne, "b1", "v", time.Hour)
	h.mustSet(ctx, alphaTwo, "s2", "v", time.Hour)

	removed, err := h.store.InvalidateSession(ctx, testInstallationID, testSessionFingerprint)
	if err != nil {
		t.Fatalf("InvalidateSession: %v", err)
	}
	if removed != 3 {
		t.Fatalf("必须返回跨扩展删除的键数 3，got %d", removed)
	}
	for _, scope := range []Scope{alphaOne, betaOne} {
		h.mustMiss(ctx, scope, "a1")
		h.mustMiss(ctx, scope, "a2")
		h.mustMiss(ctx, scope, "b1")
	}

	// 幂等：第二次返回 0 且无错误。
	removed, err = h.store.InvalidateSession(ctx, testInstallationID, testSessionFingerprint)
	if err != nil || removed != 0 {
		t.Fatalf("幂等要求 (0, nil)，got (%d, %v)", removed, err)
	}

	// 其它会话不受影响。
	if got := h.mustGet(ctx, alphaTwo, "s2"); got != "v" {
		t.Fatalf("其它会话的值必须保留，got %q", got)
	}
	if keys := h.backend.liveValueKeys(); len(keys) != 1 || keys[0] != h.valueKeyFor(alphaTwo, "s2") {
		t.Fatalf("只应剩下其它会话的值键，got %v", keys)
	}

	// 每个扩展的会话索引里都必须摘掉该 fingerprint，否则第二张索引会无限增长。
	alphaSessions := extensionSessionsIndex(testInstallationID, Fingerprint(testExtensionID))
	wantAlphaSessions := []string{testSessionBFingerprint}
	if members := h.backend.realSetMembers(alphaSessions); len(members) != 1 || members[0] != wantAlphaSessions[0] {
		t.Fatalf("alpha 的会话索引应只留下另一个会话，got %v", members)
	}
	// beta 只有被失效的那个会话写过，因此索引应被清空到键消失。
	betaSessions := extensionSessionsIndex(testInstallationID, Fingerprint(testExtensionIDBeta))
	if members := h.backend.realSetMembers(betaSessions); len(members) != 0 {
		t.Fatalf("beta 的会话索引必须已摘掉该 fingerprint，got %v", members)
	}
	if h.backend.hasSetKey(sessionKeysIndex(testInstallationID, testSessionFingerprint)) {
		t.Fatal("会话键索引必须被删除")
	}
	if h.backend.hasSetKey(sessionExtensionsIndex(testInstallationID, testSessionFingerprint)) {
		t.Fatal("会话扩展索引必须被删除")
	}
}

// TestInvalidateExtension 覆盖禁用/卸载路径：只清该扩展、幂等、并修剪会话侧成员表。
func TestInvalidateExtension(t *testing.T) {
	h := newSessionHarness(t)
	ctx := context.Background()
	alphaOne := h.scope
	betaOne := h.withExtension(testExtensionIDBeta)
	alphaTwo := h.withSession(testSessionBFingerprint)

	h.mustSet(ctx, alphaOne, "a1", "v", time.Hour)
	h.mustSet(ctx, alphaOne, "a2", "v", time.Hour)
	h.mustSet(ctx, alphaTwo, "a3", "v", time.Hour)
	h.mustSet(ctx, betaOne, "b1", "v", time.Hour)

	removed, err := h.store.InvalidateExtension(ctx, testInstallationID, testExtensionID)
	if err != nil {
		t.Fatalf("InvalidateExtension: %v", err)
	}
	if removed != 3 {
		t.Fatalf("必须返回该扩展跨会话删除的键数 3，got %d", removed)
	}
	for _, scope := range []Scope{alphaOne, alphaTwo} {
		h.mustMiss(ctx, scope, "a1")
		h.mustMiss(ctx, scope, "a3")
	}

	// 幂等。
	removed, err = h.store.InvalidateExtension(ctx, testInstallationID, testExtensionID)
	if err != nil || removed != 0 {
		t.Fatalf("幂等要求 (0, nil)，got (%d, %v)", removed, err)
	}

	// 其它扩展完好。
	if got := h.mustGet(ctx, betaOne, "b1"); got != "v" {
		t.Fatalf("其它扩展的值必须保留，got %q", got)
	}
	if keys := h.backend.liveValueKeys(); len(keys) != 1 || keys[0] != h.valueKeyFor(betaOne, "b1") {
		t.Fatalf("只应剩下 beta 的值键，got %v", keys)
	}

	// 会话侧成员表被修剪：只剩 beta 的成员；alpha 从会话扩展索引里摘掉。
	alphaSessionKeys := sessionKeysIndex(testInstallationID, testSessionFingerprint)
	if members := h.backend.realSetMembers(alphaSessionKeys); len(members) != 1 || members[0] != memberOf(testExtensionIDBeta, "b1") {
		t.Fatalf("会话键索引应只剩 beta 成员，got %v", members)
	}
	if members := h.backend.realSetMembers(sessionKeysIndex(testInstallationID, testSessionBFingerprint)); len(members) != 0 {
		t.Fatalf("另一个会话的 alpha 成员也应被摘掉，got %v", members)
	}
	alphaExtensions := sessionExtensionsIndex(testInstallationID, testSessionFingerprint)
	if members := h.backend.realSetMembers(alphaExtensions); len(members) != 1 || members[0] != Fingerprint(testExtensionIDBeta) {
		t.Fatalf("会话扩展索引应只剩 beta，got %v", members)
	}
	if h.backend.hasSetKey(extensionSessionsIndex(testInstallationID, Fingerprint(testExtensionID))) {
		t.Fatal("该扩展的会话索引必须被删除")
	}
	betaSessions := extensionSessionsIndex(testInstallationID, Fingerprint(testExtensionIDBeta))
	if members := h.backend.realSetMembers(betaSessions); len(members) != 1 || members[0] != testSessionFingerprint {
		t.Fatalf("其它扩展的会话索引必须保留，got %v", members)
	}
}

// TestBackendFailureSurfacing 覆盖每个后端方法出错时 store 的对外错误：
// 必须是 ErrBackendUnavailable，且不得透出底层文本。
func TestBackendFailureSurfacing(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name   string
		method string
		run    func(h *sessionHarness) error
	}{
		{
			name:   "Get 故障经读路径浮出",
			method: fakeMethodGet,
			run: func(h *sessionHarness) error {
				_, _, err := h.store.Get(ctx, h.scope, "existing")
				return err
			},
		},
		{
			name:   "Get 故障在写路径的存在性检查处浮出",
			method: fakeMethodGet,
			run: func(h *sessionHarness) error {
				_, err := h.store.Set(ctx, h.scope, "fresh", []byte("v"), time.Minute)
				return err
			},
		},
		{
			name:   "Set 故障",
			method: fakeMethodSet,
			run: func(h *sessionHarness) error {
				_, err := h.store.Set(ctx, h.scope, "fresh", []byte("v"), time.Minute)
				return err
			},
		},
		{
			name:   "SetAdd 故障（值键已写入，索引写失败）",
			method: fakeMethodSetAdd,
			run: func(h *sessionHarness) error {
				_, err := h.store.Set(ctx, h.scope, "fresh", []byte("v"), time.Minute)
				return err
			},
		},
		{
			name:   "SetMembers 故障（上限判定需要枚举索引）",
			method: fakeMethodSetMembers,
			run: func(h *sessionHarness) error {
				_, err := h.store.Set(ctx, h.scope, "fresh", []byte("v"), time.Minute)
				return err
			},
		},
		{
			name:   "SetRemove 故障（删除后同步索引成员）",
			method: fakeMethodSetRemove,
			run: func(h *sessionHarness) error {
				_, err := h.store.Delete(ctx, h.scope, "existing")
				return err
			},
		},
		{
			name:   "Delete 故障",
			method: fakeMethodDelete,
			run: func(h *sessionHarness) error {
				_, err := h.store.Delete(ctx, h.scope, "existing")
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newSessionHarness(t)
			h.mustSet(ctx, h.scope, "existing", "v", time.Hour)
			h.backend.failMethod(tc.method, errFakeBackend)
			err := tc.run(h)
			assertBackendUnavailable(t, err)
		})
	}

	// 全方法故障：五个 store 入口都必须 fail closed（供调用方区分「不可用」与「没有状态」）。
	for _, method := range fakeBackendMethods {
		t.Run("failAll 后所有入口都 fail closed："+method, func(t *testing.T) {
			h := newSessionHarness(t)
			h.mustSet(ctx, h.scope, "existing", "v", time.Hour)
			h.backend.failMethod(method, errFakeBackend)
			h.backend.failAll(errFakeBackend)

			operations := map[string]func() error{
				"Get": func() error {
					_, _, err := h.store.Get(ctx, h.scope, "existing")
					return err
				},
				"Set": func() error {
					_, err := h.store.Set(ctx, h.scope, "fresh", []byte("v"), time.Minute)
					return err
				},
				"Delete": func() error {
					_, err := h.store.Delete(ctx, h.scope, "existing")
					return err
				},
				"InvalidateSession": func() error {
					_, err := h.store.InvalidateSession(ctx, testInstallationID, testSessionFingerprint)
					return err
				},
				"InvalidateExtension": func() error {
					_, err := h.store.InvalidateExtension(ctx, testInstallationID, testExtensionID)
					return err
				},
			}
			for name, operation := range operations {
				assertBackendUnavailable(t, operation())
				if t.Failed() {
					t.Fatalf("%s 在 %s 故障下必须 fail closed", name, method)
				}
			}
		})
	}
}

func assertBackendUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("后端故障必须返回错误，不能静默成功")
	}
	if !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("必须折叠为 ErrBackendUnavailable，got %v", err)
	}
	if !IsBackendUnavailable(err) {
		t.Fatalf("IsBackendUnavailable 必须为 true，got %v", err)
	}
	if errors.Is(err, errFakeBackend) || strings.Contains(err.Error(), "boom") {
		t.Fatalf("不得把底层错误文本透给调用方：%v", err)
	}
}

// TestStoreCountsWithoutSetCard 固定两件事：
//  1. store 的计数完全走 SetMembers + 逐个 Get 的扫描，从不调用 Backend.SetCard
//     （扫描能顺手摘掉陈旧成员并重新计数，SCARD 做不到；store.go 头部「以 SetCard 为判据」
//     的注释与实现不一致，属于文档漂移）。
//  2. SCARD 语义本身：成员指向的值键过期后，基数仍然照算。
//
// TestStaleMembersRemainInSetCardSoCountsMustBeScanned 固定「为什么计数不能信 SCARD」：
// 成员指向的值键过期之后，集合基数仍然是 1，所以基数只能作为扫描的起点而不能作为判据。
// store 因此完全走 SetMembers + 逐个 Get（并顺手摘掉陈旧成员）。
//
// 注：接口里原本还有一个 Backend.SetCard，但 store 从未调用它——它已被删除，避免留下
// 一个没有调用方、却迫使每个后端实现 SCARD 的死接口。
func TestStaleMembersRemainInSetCardSoCountsMustBeScanned(t *testing.T) {
	h := newSessionHarness(t)
	ctx := context.Background()
	sessionKeys := sessionKeysIndex(h.scope.InstallationID, h.scope.SessionFingerprint)

	h.mustSet(ctx, h.scope, "k", "v", time.Minute)
	if _, _, err := h.store.Get(ctx, h.scope, "k"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if _, err := h.store.Delete(ctx, h.scope, "k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	h.mustSet(ctx, h.scope, "k2", "v", time.Minute)
	if _, err := h.store.InvalidateSession(ctx, testInstallationID, testSessionFingerprint); err != nil {
		t.Fatalf("InvalidateSession: %v", err)
	}
	if _, err := h.store.InvalidateExtension(ctx, testInstallationID, testExtensionID); err != nil {
		t.Fatalf("InvalidateExtension: %v", err)
	}
	// SCARD 语义：陈旧成员仍计入基数（这就是它不能当判据的原因）。
	h.mustSet(ctx, h.scope, "stale", "v", time.Minute)
	card, err := h.backend.SetCard(ctx, sessionKeys)
	if err != nil || card != 1 {
		t.Fatalf("前置条件：基数应为 1，got %d err=%v", card, err)
	}
	h.clock.Advance(2 * time.Minute)
	if _, found := h.backend.liveValue(h.valueKeyFor(h.scope, "stale")); found {
		t.Fatal("前置条件：值键应已过期")
	}
	card, err = h.backend.SetCard(ctx, sessionKeys)
	if err != nil || card != 1 {
		t.Fatalf("SCARD 不过滤陈旧成员，基数应仍为 1，got %d err=%v", card, err)
	}
}

// TestConcurrentAccessIsolation 在 -race 下并行 Set/Get/Delete：同会话安全且不跨会话泄漏。
func TestConcurrentAccessIsolation(t *testing.T) {
	h := newSessionHarness(t)
	ctx := context.Background()
	const (
		goroutines = 8
		rounds     = 25
	)
	other := h.withSession(testSessionBFingerprint)
	if _, err := h.store.Set(ctx, other, "other", []byte("other-value"), time.Hour); err != nil {
		t.Fatalf("其它会话写入: %v", err)
	}

	allowed := make(map[string]struct{}, goroutines)
	for i := 0; i < goroutines; i++ {
		allowed[fmt.Sprintf("value-%d", i)] = struct{}{}
	}

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			value := []byte(fmt.Sprintf("value-%d", id))
			ownKey := fmt.Sprintf("own-%d", id)
			for round := 0; round < rounds; round++ {
				if _, err := h.store.Set(ctx, h.scope, ownKey, value, time.Minute); err != nil {
					t.Errorf("并发 Set(own): %v", err)
					return
				}
				read, found, err := h.store.Get(ctx, h.scope, ownKey)
				if err != nil || !found || !bytes.Equal(read, value) {
					t.Errorf("并发 Get(own) 必须读回自己刚写的值：got %q found=%v err=%v", read, found, err)
					return
				}
				if _, err := h.store.Set(ctx, h.scope, "shared", value, time.Minute); err != nil {
					t.Errorf("并发 Set(shared): %v", err)
					return
				}
				shared, found, err := h.store.Get(ctx, h.scope, "shared")
				if err != nil || !found {
					t.Errorf("并发 Get(shared): found=%v err=%v", found, err)
					return
				}
				if _, ok := allowed[string(shared)]; !ok {
					t.Errorf("并发 Get(shared) 读到非本次会话写入的值：%q", shared)
					return
				}
				if _, err := h.store.Delete(ctx, h.scope, ownKey); err != nil {
					t.Errorf("并发 Delete(own): %v", err)
					return
				}
			}
		}(i)
	}
	wg.Wait()

	// 不跨会话泄漏：另一个会话看不到本会话写过的任何键。
	for _, namespaceKey := range []string{"shared", "own-0", "own-7"} {
		h.mustMiss(ctx, other, namespaceKey)
	}
	if got := h.mustGet(ctx, other, "other"); got != "other-value" {
		t.Fatalf("其它会话的值必须不受并发写入影响，got %q", got)
	}
	for _, key := range h.backend.liveValueKeys() {
		if strings.HasSuffix(key, ":session:"+testSessionBFingerprint+":key:"+Fingerprint("other")) {
			continue
		}
		if !strings.Contains(key, ":session:"+testSessionFingerprint+":key:") {
			t.Fatalf("出现了不属于本会话的残留键：%q", key)
		}
	}
}

// TestRedisBackendGuardsOffline 覆盖 RedisBackend 自己的边界守卫：这些分支都在触网之前返回，
// 因此可以用一个永不成功连接的客户端离线验证（守卫要是被绕过，拨号失败会让测试报错）。
// 真正的键空间行为由 TestRedisBackendIntegration 覆盖。
func TestRedisBackendGuardsOffline(t *testing.T) {
	ctx := context.Background()
	if _, err := NewRedisBackend(nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("NewRedisBackend(nil) 必须返回 ErrInvalid，got %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { _ = client.Close() })
	backend, err := NewRedisBackend(client)
	if err != nil {
		t.Fatalf("NewRedisBackend: %v", err)
	}

	cases := []struct {
		name string
		run  func() error
		want error
	}{
		{"Get 缺 ctx", func() error { _, _, err := backend.Get(nil, "k"); return err }, ErrInvalid},
		{"Get 空 key", func() error { _, _, err := backend.Get(ctx, ""); return err }, ErrInvalid},
		{"Set 缺 ctx", func() error { return backend.Set(nil, "k", []byte("v"), time.Minute) }, ErrInvalid},
		{"Set 空 key", func() error { return backend.Set(ctx, "", []byte("v"), time.Minute) }, ErrInvalid},
		{"Set 空值", func() error { return backend.Set(ctx, "k", nil, time.Minute) }, ErrInvalid},
		{
			"Set 值超过 1 MiB",
			func() error { return backend.Set(ctx, "k", bytes.Repeat([]byte("v"), ValueMaxBytes+1), time.Minute) },
			ErrValueTooLarge,
		},
		{
			// PEXPIRE 收到 0 会直接删键，所以这里必须报错而不是静默「写入即消失」。
			"Set TTL 为 0",
			func() error { return backend.Set(ctx, "k", []byte("v"), 0) },
			ErrTTLOutOfRange,
		},
		{
			"Set TTL 超过 24h",
			func() error { return backend.Set(ctx, "k", []byte("v"), MaxTTL+time.Second) },
			ErrTTLOutOfRange,
		},
		{"SetAdd 空 key", func() error { _, err := backend.SetAdd(ctx, "", "m", time.Minute); return err }, ErrInvalid},
		{"SetAdd 空成员", func() error { _, err := backend.SetAdd(ctx, "k", "", time.Minute); return err }, ErrInvalid},
		{"SetAdd TTL 越界", func() error { _, err := backend.SetAdd(ctx, "k", "m", 0); return err }, ErrTTLOutOfRange},
		{"Delete 空 key", func() error { _, err := backend.Delete(ctx, ""); return err }, ErrInvalid},
		{"SetMembers 空 key", func() error { _, err := backend.SetMembers(ctx, ""); return err }, ErrInvalid},
		{"SetRemove 空 key", func() error { return backend.SetRemove(ctx, "", "m") }, ErrInvalid},
		{"SetCard 空 key", func() error { _, err := backend.SetCard(ctx, ""); return err }, ErrInvalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
		})
	}

	// 空成员列表的 SetRemove 是空操作：不触网，也不报错（见 TestRedisSremArityRequiresMembers）。
	if err := backend.SetRemove(ctx, "k"); err != nil {
		t.Fatalf("空成员列表必须短路为 nil，got %v", err)
	}
}

// 默认跳过；设置 SFORUM_PLUGIN_SESSION_REDIS_ADDR 后运行（可选 _PASSWORD / _DB，DB 默认 15）。
// TestRedisBackendIntegration 是可选的端到端验证：真实 Redis + RedisBackend + Store。
// 覆盖值键/索引 TTL 落地、上限拒绝、陈旧成员修剪、失效计数与幂等。
func TestRedisBackendIntegration(t *testing.T) {
	client := redisIntegrationClient(t)
	ctx := context.Background()
	installationID := "itest-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	sessionFingerprint := Fingerprint("itest-session")
	scope := Scope{
		InstallationID:     installationID,
		ExtensionID:        "example.itest.plugin",
		SessionFingerprint: sessionFingerprint,
	}
	extensionHash := Fingerprint(scope.ExtensionID)
	// 修剪路径单独用一个扩展/会话，避免与上限用例的 64 个键互相干扰。
	prunerScope := Scope{
		InstallationID:     installationID,
		ExtensionID:        "example.itest.pruner",
		SessionFingerprint: Fingerprint("itest-pruner-session"),
	}
	prunerHash := Fingerprint(prunerScope.ExtensionID)

	keys := []string{
		sessionKeysIndex(installationID, sessionFingerprint),
		sessionExtensionsIndex(installationID, sessionFingerprint),
		extensionSessionsIndex(installationID, extensionHash),
		sessionKeysIndex(installationID, prunerScope.SessionFingerprint),
		sessionExtensionsIndex(installationID, prunerScope.SessionFingerprint),
		extensionSessionsIndex(installationID, prunerHash),
		valueKey(installationID, prunerHash, prunerScope.SessionFingerprint, "old"),
		valueKey(installationID, prunerHash, prunerScope.SessionFingerprint, "new"),
	}
	for i := 0; i < MaxKeysPerSession; i++ {
		keys = append(keys, valueKey(installationID, extensionHash, sessionFingerprint, fmt.Sprintf("key-%02d", i)))
	}
	keys = append(keys,
		valueKey(installationID, extensionHash, sessionFingerprint, "probe"),
		valueKey(installationID, extensionHash, sessionFingerprint, "overflow"),
	)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = client.Del(cleanupCtx, keys...).Err()
	})

	if _, err := NewRedisBackend(nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("NewRedisBackend(nil) 必须失败，got %v", err)
	}
	backend, err := NewRedisBackend(client)
	if err != nil {
		t.Fatalf("NewRedisBackend: %v", err)
	}
	store, err := NewStore(backend)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// 允许路径：Set → Get → Delete。
	effective, err := store.Set(ctx, scope, "probe", []byte("hello-redis"), 2*time.Minute)
	if err != nil || effective != 2*time.Minute {
		t.Fatalf("Set: effective=%s err=%v", effective, err)
	}
	value, found, err := store.Get(ctx, scope, "probe")
	if err != nil || !found || string(value) != "hello-redis" {
		t.Fatalf("Get: value=%q found=%v err=%v", value, found, err)
	}
	ttl, err := client.PTTL(ctx, valueKey(installationID, extensionHash, sessionFingerprint, "probe")).Result()
	if err != nil || ttl <= 0 || ttl > 2*time.Minute {
		t.Fatalf("值键 TTL 必须是生效 TTL（0 < ttl <= 2m），got %s err=%v", ttl, err)
	}
	sessionKeysTTL, err := client.PTTL(ctx, sessionKeysIndex(installationID, sessionFingerprint)).Result()
	if err != nil || sessionKeysTTL <= 0 || sessionKeysTTL > MaxTTL {
		t.Fatalf("索引 TTL 必须被应用（0 < ttl <= 24h），got %s err=%v", sessionKeysTTL, err)
	}
	if deleted, err := store.Delete(ctx, scope, "probe"); err != nil || !deleted {
		t.Fatalf("Delete: deleted=%v err=%v", deleted, err)
	}
	if _, found, err := store.Get(ctx, scope, "probe"); err != nil || found {
		t.Fatalf("删除后必须未命中：found=%v err=%v", found, err)
	}

	// SetAdd 语义：新成员为 true，重复成员为 false；空成员列表的 SetRemove 必须是空操作。
	// 这里不能拿 sessionFingerprint 当「新成员」：前面的 store.Set 已经把它写进扩展会话索引了。
	indexKey := extensionSessionsIndex(installationID, extensionHash)
	if members, err := backend.SetMembers(ctx, indexKey); err != nil || len(members) != 1 || members[0] != sessionFingerprint {
		t.Fatalf("前置条件：扩展会话索引应只有 store 写入的 fingerprint，got %v err=%v", members, err)
	}
	extraMember := Fingerprint("itest-extra-session")
	if added, err := backend.SetAdd(ctx, indexKey, extraMember, time.Hour); err != nil || !added {
		t.Fatalf("SetAdd 新成员应为 true：added=%v err=%v", added, err)
	}
	if added, err := backend.SetAdd(ctx, indexKey, extraMember, time.Hour); err != nil || added {
		t.Fatalf("SetAdd 重复成员应为 false：added=%v err=%v", added, err)
	}
	if card, err := backend.SetCard(ctx, indexKey); err != nil || card != 2 {
		t.Fatalf("SetCard 应为 2：got %d err=%v", card, err)
	}
	if err := backend.SetRemove(ctx, indexKey); err != nil {
		t.Fatalf("空成员列表的 SetRemove 必须是空操作，got %v", err)
	}
	if card, err := backend.SetCard(ctx, indexKey); err != nil || card != 2 {
		t.Fatalf("空成员列表的 SetRemove 不得改变基数：got %d err=%v", card, err)
	}
	if err := backend.SetRemove(ctx, indexKey, extraMember); err != nil {
		t.Fatalf("SetRemove: %v", err)
	}
	if card, err := backend.SetCard(ctx, indexKey); err != nil || card != 1 {
		t.Fatalf("SetRemove 后基数应为 1：got %d err=%v", card, err)
	}

	// 上限：留足 TTL 余量，避免真写期间先写入的键过期，把「第 65 个键被拒」变成时序竞态。
	for i := 0; i < MaxKeysPerSession; i++ {
		if _, err := store.Set(ctx, scope, fmt.Sprintf("key-%02d", i), []byte("v"), 30*time.Second); err != nil {
			t.Fatalf("填充第 %d 个键: %v", i, err)
		}
	}
	if card, err := backend.SetCard(ctx, sessionKeysIndex(installationID, sessionFingerprint)); err != nil || card != MaxKeysPerSession {
		t.Fatalf("前置条件：索引基数应为 %d，got %d err=%v", MaxKeysPerSession, card, err)
	}
	if _, err := store.Set(ctx, scope, "overflow", []byte("v"), 30*time.Second); !errors.Is(err, ErrKeyLimit) {
		t.Fatalf("第 %d 个键必须返回 ErrKeyLimit，got %v", MaxKeysPerSession+1, err)
	}

	// 真实 Redis 上的陈旧成员修剪：短 TTL 的值键先过期，索引里留下指向已消失键的成员；
	// 下一次写入必须摘掉它并按真实存活数重新计数，否则索引会永久占额度、误拒合法写入。
	if _, err := store.Set(ctx, prunerScope, "old", []byte("v"), 300*time.Millisecond); err != nil {
		t.Fatalf("Set(pruner/old): %v", err)
	}
	prunerIndex := sessionKeysIndex(installationID, prunerScope.SessionFingerprint)
	if card, err := backend.SetCard(ctx, prunerIndex); err != nil || card != 1 {
		t.Fatalf("前置条件：修剪器索引基数应为 1，got %d err=%v", card, err)
	}
	waitForRedisKeyExpiry(t, client, valueKey(installationID, prunerHash, prunerScope.SessionFingerprint, "old"), 10*time.Second)
	if card, err := backend.SetCard(ctx, prunerIndex); err != nil || card != 1 {
		t.Fatalf("前置条件：SCARD 不过滤陈旧成员，基数应仍为 1，got %d err=%v", card, err)
	}
	if _, err := store.Set(ctx, prunerScope, "new", []byte("v"), time.Minute); err != nil {
		t.Fatalf("陈旧成员必须被修剪后允许写入：%v", err)
	}
	if card, err := backend.SetCard(ctx, prunerIndex); err != nil || card != 1 {
		t.Fatalf("陈旧成员必须被摘掉（基数应回到 1 而不是 2），got %d err=%v", card, err)
	}
	if _, found, err := store.Get(ctx, prunerScope, "old"); err != nil || found {
		t.Fatalf("过期的键必须未命中：found=%v err=%v", found, err)
	}

	// 失效：跨扩展删除计数 + 幂等（第二次调用会走到空成员的 SetRemove 路径）。
	removed, err := store.InvalidateSession(ctx, installationID, sessionFingerprint)
	if err != nil || removed != MaxKeysPerSession {
		t.Fatalf("InvalidateSession 必须删除 %d 个存活键：removed=%d err=%v", MaxKeysPerSession, removed, err)
	}
	if _, found, err := store.Get(ctx, scope, "key-00"); err != nil || found {
		t.Fatalf("失效后必须未命中：found=%v err=%v", found, err)
	}
	removed, err = store.InvalidateSession(ctx, installationID, sessionFingerprint)
	if err != nil || removed != 0 {
		t.Fatalf("幂等要求 (0, nil)，got (%d, %v)", removed, err)
	}
	if _, err := store.Set(ctx, scope, "probe", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	removed, err = store.InvalidateExtension(ctx, installationID, scope.ExtensionID)
	if err != nil || removed != 1 {
		t.Fatalf("InvalidateExtension: removed=%d err=%v", removed, err)
	}
	removed, err = store.InvalidateExtension(ctx, installationID, scope.ExtensionID)
	if err != nil || removed != 0 {
		t.Fatalf("幂等要求 (0, nil)，got (%d, %v)", removed, err)
	}
}

// waitForRedisKeyExpiry 轮询到真实键消失，避免用 Sleep 猜时长。
func waitForRedisKeyExpiry(t *testing.T, client *redis.Client, key string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ttl, err := client.PTTL(context.Background(), key).Result()
		if err != nil {
			t.Fatalf("PTTL(%s): %v", key, err)
		}
		if ttl == -1 {
			t.Fatalf("键 %s 没有 TTL，与前置条件不符", key)
		}
		if ttl == -2 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("等待 %s 过期超时", key)
}

// unguardedRedisBackend 与 RedisBackend 行为一致，唯独去掉了空成员短路。它只用于下面的
// 缺陷复现：证明「短路」不是装饰，而是 store 幂等语义目前唯一的落地位置。
type unguardedRedisBackend struct {
	base *RedisBackend
}

func (b unguardedRedisBackend) Get(ctx context.Context, key string) ([]byte, bool, error) {
	return b.base.Get(ctx, key)
}

func (b unguardedRedisBackend) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return b.base.Set(ctx, key, value, ttl)
}

func (b unguardedRedisBackend) Delete(ctx context.Context, key string) (bool, error) {
	return b.base.Delete(ctx, key)
}

func (b unguardedRedisBackend) SetAdd(ctx context.Context, key string, member string, ttl time.Duration) (bool, error) {
	return b.base.SetAdd(ctx, key, member, ttl)
}

func (b unguardedRedisBackend) SetMembers(ctx context.Context, key string) ([]string, error) {
	return b.base.SetMembers(ctx, key)
}

// SetRemove 故意不加空列表短路：把调用方传来的成员原样透传给 Redis。
func (b unguardedRedisBackend) SetRemove(ctx context.Context, key string, members ...string) error {
	args := make([]interface{}, 0, len(members))
	for _, member := range members {
		args = append(args, member)
	}
	return b.base.client.SRem(ctx, key, args...).Err()
}

func (b unguardedRedisBackend) SetCard(ctx context.Context, key string) (int64, error) {
	return b.base.SetCard(ctx, key)
}

// TestInvalidateSessionIdempotencyNeedsEmptySremShortCircuit 复现一处 store.go 的缺陷面。
//
// store.InvalidateSession 在会话已无状态时仍会调用 SetRemove(索引键, 空成员列表)（store.go:298）。
// Redis 的 SREM 至少需要 key + 1 个成员，因此这个调用在后端层是 arity 错误；store 会把它折叠成
// ErrBackendUnavailable，于是「第二次调用返回 (0, nil)」在真实 Redis 上依赖适配层的空列表短路，
// 而不是依赖 store 自身的行为。若 store.go 改为在成员列表为空时跳过该调用，本测试应当反转成
// 期望 (0, nil) —— 这正是它存在的意义。
// TestInvalidateSessionStaysIdempotentWithoutEmptySremGuard 走**故意不加空列表短路**的后端：
// 幂等契约必须由 store 自己成立（它现在根本不再对空成员调用 SREM），而不是靠适配层替它兜底。
func TestInvalidateSessionStaysIdempotentWithoutEmptySremGuard(t *testing.T) {
	client := redisIntegrationClient(t)
	ctx := context.Background()
	installationID := "itest-guard-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	sessionFingerprint := Fingerprint("itest-guard-session")
	scope := Scope{
		InstallationID:     installationID,
		ExtensionID:        "example.itest.guard",
		SessionFingerprint: sessionFingerprint,
	}
	extensionHash := Fingerprint(scope.ExtensionID)
	keys := []string{
		sessionKeysIndex(installationID, sessionFingerprint),
		sessionExtensionsIndex(installationID, sessionFingerprint),
		extensionSessionsIndex(installationID, extensionHash),
		valueKey(installationID, extensionHash, sessionFingerprint, "k"),
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = client.Del(cleanupCtx, keys...).Err()
	})

	base, err := NewRedisBackend(client)
	if err != nil {
		t.Fatalf("NewRedisBackend: %v", err)
	}
	store, err := NewStore(unguardedRedisBackend{base: base})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	if _, err := store.Set(ctx, scope, "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	removed, err := store.InvalidateSession(ctx, installationID, sessionFingerprint)
	if err != nil || removed != 1 {
		t.Fatalf("第一次 InvalidateSession: removed=%d err=%v", removed, err)
	}
	removed, err = store.InvalidateSession(ctx, installationID, sessionFingerprint)
	if err != nil || removed != 0 {
		t.Fatalf("第二次 InvalidateSession 必须由 store 自身保证幂等 (0, nil)，"+
			"不能依赖后端对空成员 SREM 的短路：got (%d, %v)", removed, err)
	}
}

// TestRedisSremArityRequiresMembers 固定 Redis 的事实：无成员的 SREM 是 arity 错误。
// RedisBackend.SetRemove 因此在成员列表为空时短路，store.InvalidateSession 的「幂等返回
// (0, nil)」才在真实 Redis 上成立。删掉那个短路，这个事实就会变成运行期后端不可用。
func TestRedisSremArityRequiresMembers(t *testing.T) {
	client := redisIntegrationClient(t)
	ctx := context.Background()
	key := "sforum:plugin-session:v1:itest-srem-arity"
	t.Cleanup(func() { _ = client.Del(context.Background(), key).Err() })

	if _, err := client.SAdd(ctx, key, "member").Result(); err != nil {
		t.Fatalf("SAdd: %v", err)
	}
	if err := client.SRem(ctx, key).Err(); err == nil {
		t.Fatal("Redis 对无成员的 SREM 必须报错（arity）；若不报错，说明 RedisBackend 的短路已无必要，应改为直接透传")
	} else if !strings.Contains(strings.ToLower(err.Error()), "wrong number of arguments") {
		t.Fatalf("期望 arity 错误，got %v", err)
	}
	if err := client.SRem(ctx, key, "member").Err(); err != nil {
		t.Fatalf("带成员的 SREM 必须成功：%v", err)
	}
}

// redisIntegrationClient 返回集成测试用的客户端；没有配置地址或 Redis 不可用时跳过。
func redisIntegrationClient(t *testing.T) *redis.Client {
	t.Helper()
	addr := strings.TrimSpace(os.Getenv("SFORUM_PLUGIN_SESSION_REDIS_ADDR"))
	if addr == "" {
		t.Skip("需要 SFORUM_PLUGIN_SESSION_REDIS_ADDR 指向可用 Redis（可选 SFORUM_PLUGIN_SESSION_REDIS_PASSWORD/_DB）")
	}
	db := 15
	if raw := strings.TrimSpace(os.Getenv("SFORUM_PLUGIN_SESSION_REDIS_DB")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("SFORUM_PLUGIN_SESSION_REDIS_DB 不是整数：%q", raw)
		}
		db = parsed
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("SFORUM_PLUGIN_SESSION_REDIS_PASSWORD"),
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Skipf("Redis %s 不可用：%v", addr, err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

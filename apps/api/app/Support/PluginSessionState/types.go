// Package pluginsessionstate owns Host-managed, plugin-namespaced session state.
//
// Contract: Revision 2 (S1-S8) of
// knowledge/decisions/2026-10-16-extension-authoring-contracts-freeze.md.
//
// 三条不变量决定了这个包的全部形状，改代码前先读它们：
//
//  1. 插件状态永远不进 Core 会话载荷（AuthSession 的 payload 不能新增 key）。
//     每个请求都会重写整键 TTL，而 clearSessionData 会在不相关的签发/续期失败时
//     清空载荷——放进去等于让插件状态延长 Core 会话寿命，并且会被莫名清掉。
//  2. 状态绝不按原始 session id 建键。session 分量用 sha256(sid)：它跨 24h 轮换稳定
//     （sid 本身稳定），且不是凭据（原始 sid 只存在于 Core 的会话键里）。
//  3. 插件状态不得延长 Core 会话。有效 TTL 取 requested/24h/剩余 idle/剩余绝对寿命
//     四者的最小值；读时也要复核绝对寿命，因为 Redis TTL 只是机制、不是权威。
//
// 本包只负责「存储 + 边界 + 隔离」。Safe Mode、运行时准入与信任判定属于调用方
// （Host 侧适配器持有运行时上下文的地方），这样存储层可以被独立测试。
package pluginsessionstate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const (
	// NamespacePrefix 是唯一的键前缀，与 sforum:host-cache:v1:… 同族；应用里
	// 唯一不带前缀的键是 Core 会话键本身。
	NamespacePrefix = "sforum:plugin-session:v1:"

	// KeyMaxBytes / ValueMaxBytes / MinTTL / MaxTTL 沿用已验证的 host-cache 信封
	// （Support/HostAPI/cache_registry_service.go:27-31），不另发明数字。
	KeyMaxBytes   = 512
	ValueMaxBytes = 1 << 20
	MinTTL        = time.Millisecond
	MaxTTL        = 24 * time.Hour

	// MaxKeysPerSession / MaxKeysPerExtension 是每会话与每扩展的键数上限。
	MaxKeysPerSession   = 64
	MaxKeysPerExtension = 4096
)

var (
	// ErrInvalid 表示调用方给了不可能的输入（nil ctx、空 scope、空键等）。
	ErrInvalid = errors.New("pluginsessionstate: invalid input")
	// ErrKeyTooLong 表示 namespace key 超过 KeyMaxBytes。
	ErrKeyTooLong = errors.New("pluginsessionstate: namespace key too long")
	// ErrValueTooLarge 表示 value 超过 ValueMaxBytes。
	ErrValueTooLarge = errors.New("pluginsessionstate: value too large")
	// ErrTTLOutOfRange 表示请求的 TTL 不在 [MinTTL, MaxTTL]。
	ErrTTLOutOfRange = errors.New("pluginsessionstate: ttl out of range")
	// ErrKeyLimit 表示该 (扩展, 会话) 已达 MaxKeysPerSession，且这是一个新键。
	ErrKeyLimit = errors.New("pluginsessionstate: session key limit reached")
	// ErrExtensionLimit 表示该扩展安装已达 MaxKeysPerExtension，且这是一个新键。
	ErrExtensionLimit = errors.New("pluginsessionstate: extension key limit reached")
	// ErrBackendUnavailable 表示底层存储不可用；调用方必须 fail closed，不得降级为允许。
	ErrBackendUnavailable = errors.New("pluginsessionstate: backend unavailable")
)

// Scope 是 Host 侧从运行时上下文派生出来的一次调用身份。
//
// 关键点：这些字段**全部**由 Host 填充。插件侧请求里只有 namespace key / value / ttl，
// 因此「调用方伪造扩展 id 或 session id」在请求形状层面就不存在（S7）。
type Scope struct {
	// InstallationID 是持久安装身份（Support/InstallationIdentity）。
	InstallationID string
	// ExtensionID 是当前运行实例所属扩展；不来自请求。
	ExtensionID string
	// RuntimeInstanceID 是该扩展的精确运行实例；仅用于审计/追踪，不参与键派生
	// （键派生只依赖 installation + extension + session，否则重启换实例就会丢状态）。
	RuntimeInstanceID string
	// SessionFingerprint 是 sha256(sid) 的十六进制表示。原始 sid 永不进入本包。
	SessionFingerprint string
	// SessionIDleExpiresAt / SessionAbsoluteExpiresAt 是 Core 会话的两个寿命边界。
	// 零值表示该边界未知；未知时不放宽任何边界（见 EffectiveTTL 的注释）。
	SessionIDleExpiresAt     time.Time
	SessionAbsoluteExpiresAt time.Time
}

// Validate 检查 Host 侧派生的身份是否完整。任何字段缺失都是编程错误而非用户错误，
// 因此一律 fail closed，不允许「缺字段就当成匿名或无隔离」。
func (s Scope) Validate() error {
	if !validIdentityPart(s.InstallationID) {
		return ErrInvalid
	}
	if !validIdentityPart(s.ExtensionID) {
		return ErrInvalid
	}
	if !validFingerprint(s.SessionFingerprint) {
		return ErrInvalid
	}
	return nil
}

// EffectiveTTL 按 S3 计算实际生效的 TTL：四者取最小，永不放宽。
//
//	effective = min(requested, MaxTTL, 剩余 idle, 剩余绝对)
//
// 边界未知（零值）时**不**当作无限：请求值仍被 MaxTTL 夹住，只是无法进一步收紧。
// 已过期（边界早于 now）直接判为过期错误，避免写入一个「活着但已失效」的状态。
func (s Scope) EffectiveTTL(now time.Time, requested time.Duration) (time.Duration, error) {
	if requested < MinTTL || requested > MaxTTL {
		return 0, ErrTTLOutOfRange
	}
	// 注意：这里不需要再夹一次 MaxTTL——上面的范围门已经拒绝了 > MaxTTL 的请求值，
	// 而两个会话边界只会把它收紧。留着「夹取」只会是一段永远不会执行的死代码。
	effective := requested
	if !s.SessionIDleExpiresAt.IsZero() {
		remaining := s.SessionIDleExpiresAt.Sub(now)
		if remaining <= 0 {
			return 0, ErrInvalid
		}
		if remaining < effective {
			effective = remaining
		}
	}
	if !s.SessionAbsoluteExpiresAt.IsZero() {
		remaining := s.SessionAbsoluteExpiresAt.Sub(now)
		if remaining <= 0 {
			return 0, ErrInvalid
		}
		if remaining < effective {
			effective = remaining
		}
	}
	if effective < MinTTL {
		return 0, ErrInvalid
	}
	return effective, nil
}

// Expired 报告 Core 会话是否已经越过任一寿命边界。读路径必须调用它：Redis TTL 只是
// 机制，会话是否还有效由 Host 说了算。
func (s Scope) Expired(now time.Time) bool {
	if !s.SessionIDleExpiresAt.IsZero() && !now.Before(s.SessionIDleExpiresAt) {
		return true
	}
	if !s.SessionAbsoluteExpiresAt.IsZero() && !now.Before(s.SessionAbsoluteExpiresAt) {
		return true
	}
	return false
}

// ValidateNamespaceKey 检查插件提供的键。键名可以是任何 UTF-8（含中文与分隔符）；
// 它只作为哈希输入，因此不会与我们的键结构产生歧义，也不会有路径穿越问题。
func ValidateNamespaceKey(key string) error {
	if key == "" || strings.TrimSpace(key) != key {
		return ErrInvalid
	}
	if len(key) > KeyMaxBytes {
		return ErrKeyTooLong
	}
	return nil
}

// ValidateValue 检查值。V1 是不透明字节（S2）：不做 schema 校验，但必须有界，
// 且不允许写入空值（删除是单独的语义，空值会让「存在但空」与「不存在」难以区分）。
func ValidateValue(value []byte) error {
	if len(value) == 0 {
		return ErrInvalid
	}
	if len(value) > ValueMaxBytes {
		return ErrValueTooLarge
	}
	return nil
}

// Fingerprint 复用 sha256 十六进制表示。sid 与扩展 id 都走同一条路径，
// 使键长恒定、无原始标识符、无分隔符歧义。
func Fingerprint(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func validFingerprint(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func validIdentityPart(value string) bool {
	if value == "" || strings.TrimSpace(value) != value {
		return false
	}
	if strings.ContainsAny(value, ":[]*? \t\n\r") {
		// 冒号会与键结构冲突；通配符会与 keyspace 扫描冲突。
		return false
	}
	return len(value) <= 255
}

// Backend 是本包需要的全部 Redis 能力，刻意保持窄面以便测试替身与后续换存储。
type Backend interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) (bool, error)
	SetAdd(ctx context.Context, key string, member string, ttl time.Duration) (bool, error)
	SetMembers(ctx context.Context, key string) ([]string, error)
	SetRemove(ctx context.Context, key string, members ...string) error
}

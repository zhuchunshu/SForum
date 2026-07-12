package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// LoginLockout 分层登录失败节流（防定向账号锁定与喷洒）。
//
// 计数维度（Redis key 仅含哈希，不含原始邮箱/用户名）：
//   - account+IP：低阈值 → 仅锁定该 pair（不拖累其它来源）
//   - IP：更高阈值 → 限制单 IP 喷洒
//   - account：更高阈值 → 账号级退避（非与 pair 相同的低阈值硬锁）
//
// Redis 故障时 fail open：不阻断合法登录，仅跳过节流。
type LoginLockout struct {
	client loginRedis
}

type loginRedis interface {
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Incr(ctx context.Context, key string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

func NewLoginLockout(client *redis.Client) *LoginLockout {
	if client == nil {
		return nil
	}
	return &LoginLockout{client: client}
}

// IsLocked 只检查 pair/IP 硬限制；账号累计失败不得锁死所有来源。
func (l *LoginLockout) IsLocked(ctx context.Context, loginKey, clientIP string) (bool, error) {
	if l == nil || l.client == nil {
		return false, nil
	}
	loginHash := hashID(loginKey)
	ipHash := hashID(normalizeIP(clientIP))
	if loginHash == "" && ipHash == "" {
		return false, nil
	}
	keys := make([]string, 0, 2)
	if loginHash != "" && ipHash != "" {
		keys = append(keys, pairLockKey(loginHash, ipHash))
	}
	if ipHash != "" {
		keys = append(keys, ipLockKey(ipHash))
	}
	if len(keys) == 0 {
		return false, nil
	}
	n, err := l.client.Exists(ctx, keys...).Result()
	if err != nil {
		// Redis 非权威：故障时不全局拒绝登录。
		return false, nil
	}
	return n > 0, nil
}

func (l *LoginLockout) RequiresVerification(ctx context.Context, loginKey string) (bool, error) {
	if l == nil || l.client == nil {
		return false, nil
	}
	loginHash := hashID(loginKey)
	if loginHash == "" {
		return false, nil
	}
	n, err := l.client.Exists(ctx, accountVerificationKey(loginHash)).Result()
	if err != nil {
		return false, nil
	}
	return n > 0, nil
}

// RecordFailure 写入分层计数；阈值相对 policy.MaxFailures：
// pair = max，IP = max*5，account = max*3（账号仅要求验证，不硬锁）。
func (l *LoginLockout) RecordFailure(ctx context.Context, loginKey, clientIP string, maxFailures int, lockout time.Duration) error {
	if l == nil || l.client == nil || maxFailures <= 0 || lockout <= 0 {
		return nil
	}
	loginHash := hashID(loginKey)
	ipHash := hashID(normalizeIP(clientIP))
	if loginHash == "" && ipHash == "" {
		return nil
	}

	pairMax := maxFailures
	ipMax := maxFailures * 5
	if ipMax < 20 {
		ipMax = 20
	}
	accountMax := maxFailures * 3
	if accountMax < pairMax+2 {
		accountMax = pairMax + 2
	}

	// pair 锁定：同一账号+IP 低阈值
	if loginHash != "" && ipHash != "" {
		if err := l.bump(ctx, pairFailKey(loginHash, ipHash), pairLockKey(loginHash, ipHash), pairMax, lockout); err != nil {
			return nil // fail open
		}
	}
	// IP 喷洒
	if ipHash != "" {
		if err := l.bump(ctx, ipFailKey(ipHash), ipLockKey(ipHash), ipMax, lockout); err != nil {
			return nil
		}
	}
	// 账号级只设置验证标记，不参与 IsLocked；分布式来源不能锁死正确密码。
	if loginHash != "" {
		// 账号锁 TTL 可略短，鼓励其它可信来源恢复
		acctTTL := lockout
		if acctTTL > 5*time.Minute {
			acctTTL = lockout / 2
			if acctTTL < 5*time.Minute {
				acctTTL = 5 * time.Minute
			}
		}
		if err := l.bump(ctx, accountFailKey(loginHash), accountVerificationKey(loginHash), accountMax, acctTTL); err != nil {
			return nil
		}
	}
	return nil
}

func (l *LoginLockout) ClearFailures(ctx context.Context, loginKey, clientIP string) error {
	if l == nil || l.client == nil {
		return nil
	}
	loginHash := hashID(loginKey)
	ipHash := hashID(normalizeIP(clientIP))
	keys := []string{}
	if loginHash != "" && ipHash != "" {
		keys = append(keys, pairFailKey(loginHash, ipHash), pairLockKey(loginHash, ipHash))
	}
	if loginHash != "" {
		// 成功登录清除账号风险（IP 计数保留，防喷洒）。
		keys = append(keys, accountFailKey(loginHash), accountVerificationKey(loginHash))
	}
	if len(keys) == 0 {
		return nil
	}
	if err := l.client.Del(ctx, keys...).Err(); err != nil {
		return nil // fail open
	}
	return nil
}

func (l *LoginLockout) bump(ctx context.Context, failKey, lockKey string, max int, ttl time.Duration) error {
	count, err := l.client.Incr(ctx, failKey).Result()
	if err != nil {
		return fmt.Errorf("login lockout incr: %w", err)
	}
	if count == 1 {
		_ = l.client.Expire(ctx, failKey, ttl).Err()
	}
	if int(count) >= max {
		if err := l.client.Set(ctx, lockKey, "1", ttl).Err(); err != nil {
			return fmt.Errorf("login lockout set: %w", err)
		}
		_ = l.client.Del(ctx, failKey).Err()
	}
	return nil
}

func hashID(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(raw))
	// 截断 16 字节 hex 足够区分，缩短 key。
	return hex.EncodeToString(sum[:16])
}

func normalizeIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return "unknown"
	}
	return ip
}

func pairFailKey(loginHash, ipHash string) string {
	return "sforum:login_fail:pair:" + loginHash + ":" + ipHash
}
func pairLockKey(loginHash, ipHash string) string {
	return "sforum:login_lock:pair:" + loginHash + ":" + ipHash
}
func ipFailKey(ipHash string) string         { return "sforum:login_fail:ip:" + ipHash }
func ipLockKey(ipHash string) string         { return "sforum:login_lock:ip:" + ipHash }
func accountFailKey(h string) string         { return "sforum:login_fail:acct:" + h }
func accountVerificationKey(h string) string { return "sforum:login_verify:acct:" + h }

// 兼容旧接口测试：仅账号维度时仍可用空 IP（记为 unknown）。
func failKey(login string) string {
	return accountFailKey(hashID(login))
}
func lockKey(login string) string {
	return accountVerificationKey(hashID(login))
}

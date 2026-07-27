package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// 外部主体摘要：Core 拥有的 keyed digest。
//
// 安全约束（见 plans/2026-07-27-github-social-login-builtin-plugin.md 与
// decisions/2026-07-27-github-social-login-builtin-v1.md）：
//   - 插件只在精确 typed 响应里返回 raw 外部 subject；
//   - Core 校验后计算 HMAC-SHA256(key, providerId || 0x00 || subject)，
//     只持久化 digest；
//   - raw subject 与 digest 都不得出现在浏览器/API 响应、回调 URL、
//     持久化 Core 存储之外的位置、日志或审计里；
//   - 生产环境经 config.APP_ENV=production 拒绝弱/默认密钥；
//   - 开发使用稳定可配置默认，禁止进程随机材料；
//   - 密钥属于 identity 备份/恢复；轮换需要未来的 versioned dual-read 迁移。

const (
	// IdentitySubjectHMACSecretEnv 是生产 secret 的环境变量名（与 config 一致）。
	IdentitySubjectHMACSecretEnv = "IDENTITY_SUBJECT_HMAC_SECRET"
	// identitySubjectHMACMinKeyBytes 是最小密钥字节数（256-bit）。
	identitySubjectHMACMinKeyBytes = 32
	// identitySubjectHMACStableDevDefault 是测试/未注入路径的稳定开发默认。
	// 必须与 config.IdentitySubjectHMACDevDefault 保持一致；禁止进程随机。
	identitySubjectHMACStableDevDefault = "sforum-dev-identity-subject-hmac-v1-not-for-prod!!"
)

var (
	// ErrIdentitySubjectHMACWeak 在生产环境密钥弱/缺失时致命失败。
	ErrIdentitySubjectHMACWeak = errors.New("identity subject hmac secret is weak or unset in production")
	// ErrIdentitySubjectHMACNotConfigured 表示 digest 服务尚未注入稳定密钥。
	ErrIdentitySubjectHMACNotConfigured = errors.New("identity subject hmac key is not configured")
)

// subjectHMACState 缓存进程级 HMAC 密钥；由 bootstrap 注入，测试可重置。
type subjectHMACState struct {
	mu         sync.RWMutex
	key        []byte
	configured bool
}

var subjectHMACCache subjectHMACState

// ConfigureIdentitySubjectHMAC 由 bootstrap 注入稳定密钥材料（config 已按 APP_ENV 校验）。
// 禁止进程随机材料；同一进程可幂等覆盖（主要用于测试）。
func ConfigureIdentitySubjectHMAC(secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		secret = identitySubjectHMACStableDevDefault
	}
	key := parseSubjectHMACKeyBytes(secret)
	if len(key) == 0 {
		return fmt.Errorf("%w: empty key material", ErrIdentitySubjectHMACNotConfigured)
	}
	// 复制密钥，避免调用方持有可变切片。
	cloned := make([]byte, len(key))
	copy(cloned, key)

	subjectHMACCache.mu.Lock()
	subjectHMACCache.key = cloned
	subjectHMACCache.configured = true
	subjectHMACCache.mu.Unlock()
	return nil
}

// IdentitySubjectHMACKey 返回进程级缓存的主体 HMAC 密钥。
// 优先使用 bootstrap/Configure 注入的密钥；测试未注入时回退到稳定开发默认。
// 从不使用进程随机材料。
func IdentitySubjectHMACKey() ([]byte, error) {
	subjectHMACCache.mu.RLock()
	if subjectHMACCache.configured && len(subjectHMACCache.key) > 0 {
		out := make([]byte, len(subjectHMACCache.key))
		copy(out, subjectHMACCache.key)
		subjectHMACCache.mu.RUnlock()
		return out, nil
	}
	subjectHMACCache.mu.RUnlock()

	// 懒加载稳定开发默认（仅测试/未走 bootstrap 的路径）。
	// 生产 API 启动路径必须先 Configure；config.Load 已在 APP_ENV=production 拒绝弱密钥。
	if err := ConfigureIdentitySubjectHMAC(identitySubjectHMACStableDevDefault); err != nil {
		return nil, err
	}
	return IdentitySubjectHMACKey()
}

// ResetIdentitySubjectHMACKeyForTest 仅供测试重置缓存。
func ResetIdentitySubjectHMACKeyForTest() {
	subjectHMACCache.mu.Lock()
	subjectHMACCache.key = nil
	subjectHMACCache.configured = false
	subjectHMACCache.mu.Unlock()
}

func parseSubjectHMACKeyBytes(secret string) []byte {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil
	}
	if decoded, err := hex.DecodeString(secret); err == nil && len(decoded) >= identitySubjectHMACMinKeyBytes {
		return decoded
	}
	return []byte(secret)
}

// ComputeSubjectDigest 计算 HMAC-SHA256(key, providerId || 0x00 || subject)，
// 返回小写 hex。Provider 与 subject 由 Core 校验，密钥由 Core 拥有。
// 失败时返回错误；调用方必须 fail closed。
func ComputeSubjectDigest(providerID, rawSubject string) (string, error) {
	if providerID == "" || rawSubject == "" {
		return "", errors.New("subject digest requires providerId and subject")
	}
	key, err := IdentitySubjectHMACKey()
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(providerID))
	mac.Write([]byte{0x00})
	mac.Write([]byte(rawSubject))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// VerifySubjectDigest 用 constant-time 比较校验 digest 是否匹配。
func VerifySubjectDigest(providerID, rawSubject, expectedHex string) (bool, error) {
	actual, err := ComputeSubjectDigest(providerID, rawSubject)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expectedHex)) == 1, nil
}

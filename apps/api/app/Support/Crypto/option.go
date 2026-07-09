// Package crypto 提供 option 敏感值的 AES-256-GCM 静态加密（H2a）。
// 密钥来自 APP_OPTION_ENC_KEY（hex 编码的 32 字节），用于加解密 web_options 中的凭证类值，
// 避免数据库泄漏时云存储/SSH/FTP 凭证明文暴露。
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// cipherPrefix 标识已加密的值，便于解密时区分明文与密文（幂等迁移）。
const cipherPrefix = "enc::"

// ErrInvalidKey 表示加密密钥格式或长度不合法。
var ErrInvalidKey = errors.New("crypto: invalid encryption key (expected 32-byte hex)")

// ErrAlreadyEncrypted 表示值已是密文（避免重复加密）。
var ErrAlreadyEncrypted = errors.New("crypto: value is already encrypted")

// OptionCipher 对 option 敏感值做 AES-256-GCM 加解密。
// key 为 32 字节原始密钥（AES-256）。空 cipher（key 为 nil）时为透明模式，
// 加解密直接返回原文，用于未配置密钥的开发环境。
type OptionCipher struct {
	aead cipher.AEAD
}

// NewOptionCipher 用 hex 编码的 32 字节密钥构造 cipher。
// hexKey 为空时返回透明 cipher（不加密），便于开发环境。
func NewOptionCipher(hexKey string) (*OptionCipher, error) {
	if hexKey == "" {
		return &OptionCipher{}, nil
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: decoded %d bytes, want 32", ErrInvalidKey, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: create aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: create gcm: %w", err)
	}
	return &OptionCipher{aead: aead}, nil
}

// Enabled 表示是否实际启用加密（密钥已配置）。
func (c *OptionCipher) Enabled() bool {
	return c != nil && c.aead != nil
}

// Encrypt 加密明文。已带密文前缀的值视为已加密原样返回（幂等）。
// 未启用加密时（透明模式）直接返回原文。
func (c *OptionCipher) Encrypt(plaintext string) (string, error) {
	if !c.Enabled() {
		return plaintext, nil
	}
	if IsEncrypted(plaintext) {
		return plaintext, nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("crypto: read nonce: %w", err)
	}
	// nonce || ciphertext（GCM 把 tag 附在密文末尾）
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return cipherPrefix + hex.EncodeToString(sealed), nil
}

// Decrypt 解密密文。非密文（无前缀）的值视为明文原样返回（兼容历史明文数据）。
// 未启用加密时直接返回原文。
func (c *OptionCipher) Decrypt(stored string) (string, error) {
	if !c.Enabled() || !IsEncrypted(stored) {
		return stored, nil
	}
	raw, err := hex.DecodeString(stored[len(cipherPrefix):])
	if err != nil {
		return "", fmt.Errorf("crypto: decode ciphertext hex: %w", err)
	}
	if len(raw) < c.aead.NonceSize() {
		return "", errors.New("crypto: ciphertext too short")
	}
	nonce, ciphertext := raw[:c.aead.NonceSize()], raw[c.aead.NonceSize():]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: decrypt: %w", err)
	}
	return string(plaintext), nil
}

// IsEncrypted 判断存储值是否为密文（带前缀）。
func IsEncrypted(stored string) bool {
	return len(stored) > len(cipherPrefix) && stored[:len(cipherPrefix)] == cipherPrefix
}

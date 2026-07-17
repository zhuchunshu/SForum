package idempotency

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

const requiredReplayCipherDomain = "sforum.required-route-replay.context@1"

var ErrRequiredReplayCipherInvalid = errors.New("idempotency: required replay cipher is invalid")

// RequiredReplayCipher deliberately has no plaintext compatibility mode. An
// empty key keeps ordinary response-only replay available, while any mutable
// authorization transcript fails closed until production key material exists.
type RequiredReplayCipher struct {
	aead cipher.AEAD
}

func NewRequiredReplayCipher(hexKey string) (*RequiredReplayCipher, error) {
	if hexKey == "" {
		return &RequiredReplayCipher{}, nil
	}
	master, err := hex.DecodeString(hexKey)
	if err != nil || len(master) != 32 {
		return nil, ErrRequiredReplayCipherInvalid
	}
	key, err := hkdf.Key(sha256.New, master, nil, requiredReplayCipherDomain, 32)
	if err != nil {
		return nil, fmt.Errorf("%w: derive key", ErrRequiredReplayCipherInvalid)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: create AES", ErrRequiredReplayCipherInvalid)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: create GCM", ErrRequiredReplayCipherInvalid)
	}
	return &RequiredReplayCipher{aead: aead}, nil
}

func (c *RequiredReplayCipher) Enabled() bool { return c != nil && c.aead != nil }

func (c *RequiredReplayCipher) Encrypt(
	storageKey string,
	fingerprint string,
	planDigest string,
	plaintext []byte,
) (string, error) {
	if !c.Enabled() || storageKey == "" || !validRequiredReplayFingerprint(fingerprint) ||
		!validRequiredReplayFingerprint(planDigest) || len(plaintext) == 0 || len(plaintext) > MaxRequiredReplayEvidence {
		return "", ErrRequiredReplayCipherInvalid
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("%w: read nonce", ErrRequiredReplayCipherInvalid)
	}
	sealed := c.aead.Seal(nonce, nonce, plaintext, requiredReplayCipherAAD(storageKey, fingerprint, planDigest))
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (c *RequiredReplayCipher) Decrypt(
	storageKey string,
	fingerprint string,
	planDigest string,
	encoded string,
) ([]byte, error) {
	if !c.Enabled() || storageKey == "" || !validRequiredReplayFingerprint(fingerprint) ||
		!validRequiredReplayFingerprint(planDigest) || encoded == "" {
		return nil, ErrRequiredReplayCipherInvalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(raw) < c.aead.NonceSize()+c.aead.Overhead() {
		return nil, ErrRequiredReplayCipherInvalid
	}
	nonce, ciphertext := raw[:c.aead.NonceSize()], raw[c.aead.NonceSize():]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, requiredReplayCipherAAD(storageKey, fingerprint, planDigest))
	if err != nil || len(plaintext) == 0 || len(plaintext) > MaxRequiredReplayEvidence {
		return nil, ErrRequiredReplayCipherInvalid
	}
	return plaintext, nil
}

func requiredReplayCipherAAD(storageKey, fingerprint, planDigest string) []byte {
	return []byte(requiredReplaySchemaV2 + "\x00" + storageKey + "\x00" + fingerprint + "\x00" + planDigest)
}

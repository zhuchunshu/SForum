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

const (
	requiredReplayCipherDomain        = "sforum.required-route-replay.context@1"
	requiredReplayPayloadCipherDomain = "sforum.required-route-replay.payload@1"
)

var ErrRequiredReplayCipherInvalid = errors.New("idempotency: required replay cipher is invalid")

// RequiredReplayCipher deliberately has no plaintext compatibility mode for
// encrypted records. An empty key keeps legacy response-only replay available
// in development; production always configures the root key.
type RequiredReplayCipher struct {
	aead        cipher.AEAD
	payloadAEAD cipher.AEAD
}

func NewRequiredReplayCipher(hexKey string) (*RequiredReplayCipher, error) {
	if hexKey == "" {
		return &RequiredReplayCipher{}, nil
	}
	master, err := hex.DecodeString(hexKey)
	if err != nil || len(master) != 32 {
		return nil, ErrRequiredReplayCipherInvalid
	}
	aead, err := newRequiredReplayAEAD(master, requiredReplayCipherDomain)
	if err != nil {
		return nil, err
	}
	payloadAEAD, err := newRequiredReplayAEAD(master, requiredReplayPayloadCipherDomain)
	if err != nil {
		return nil, err
	}
	return &RequiredReplayCipher{aead: aead, payloadAEAD: payloadAEAD}, nil
}

func newRequiredReplayAEAD(master []byte, domain string) (cipher.AEAD, error) {
	key, err := hkdf.Key(sha256.New, master, nil, domain, 32)
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
	return aead, nil
}

func (c *RequiredReplayCipher) Enabled() bool {
	return c != nil && c.aead != nil && c.payloadAEAD != nil
}

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

func (c *RequiredReplayCipher) EncryptReplay(
	storageKey string,
	fingerprint string,
	planDigest string,
	plaintext []byte,
) (string, error) {
	if !c.Enabled() || storageKey == "" || !validRequiredReplayFingerprint(fingerprint) ||
		!validRequiredReplayFingerprint(planDigest) ||
		len(plaintext) == 0 || len(plaintext) > MaxRequiredReplayPayload {
		return "", ErrRequiredReplayCipherInvalid
	}
	nonce := make([]byte, c.payloadAEAD.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("%w: read nonce", ErrRequiredReplayCipherInvalid)
	}
	sealed := c.payloadAEAD.Seal(
		nonce, nonce, plaintext, requiredReplayPayloadCipherAAD(
			requiredReplayPayloadSchemaV2, storageKey, fingerprint, planDigest,
		),
	)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (c *RequiredReplayCipher) DecryptReplay(
	storageKey string,
	fingerprint string,
	planDigest string,
	encoded string,
) ([]byte, error) {
	if !c.Enabled() || storageKey == "" || !validRequiredReplayFingerprint(fingerprint) ||
		!validRequiredReplayFingerprint(planDigest) || encoded == "" ||
		len(encoded) > base64.RawURLEncoding.EncodedLen(MaxRequiredReplayPayload+c.payloadAEAD.NonceSize()+c.payloadAEAD.Overhead()) {
		return nil, ErrRequiredReplayCipherInvalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(raw) < c.payloadAEAD.NonceSize()+c.payloadAEAD.Overhead() {
		return nil, ErrRequiredReplayCipherInvalid
	}
	nonce, ciphertext := raw[:c.payloadAEAD.NonceSize()], raw[c.payloadAEAD.NonceSize():]
	for _, schema := range []string{requiredReplayPayloadSchemaV2, requiredReplayPayloadSchemaV1} {
		plaintext, openErr := c.payloadAEAD.Open(
			nil, nonce, ciphertext,
			requiredReplayPayloadCipherAAD(schema, storageKey, fingerprint, planDigest),
		)
		if openErr == nil && len(plaintext) > 0 && len(plaintext) <= MaxRequiredReplayPayload {
			return plaintext, nil
		}
	}
	return nil, ErrRequiredReplayCipherInvalid
}

func requiredReplayPayloadCipherAAD(schema, storageKey, fingerprint, planDigest string) []byte {
	return []byte(
		requiredReplaySchemaV3 + "\x00" + schema + "\x00" + storageKey + "\x00" +
			fingerprint + "\x00" + planDigest,
	)
}

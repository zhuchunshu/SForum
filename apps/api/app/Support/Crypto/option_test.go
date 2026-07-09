package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"testing"
)

func testKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("read key: %v", err)
	}
	return hex.EncodeToString(key)
}

// TestOptionCipherRoundTrip 验证加解密往返一致。
func TestOptionCipherRoundTrip(t *testing.T) {
	cipher, err := NewOptionCipher(testKey(t))
	if err != nil {
		t.Fatalf("NewOptionCipher: %v", err)
	}
	for _, plaintext := range []string{"secret-key", "AKIAIOSFODNN7EXAMPLE", "复杂的中文密钥 🔑", ""} {
		encrypted, err := cipher.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", plaintext, err)
		}
		if plaintext != "" && encrypted == plaintext {
			t.Fatalf("expected ciphertext to differ from plaintext for %q", plaintext)
		}
		decrypted, err := cipher.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Decrypt(%q): %v", encrypted, err)
		}
		if decrypted != plaintext {
			t.Fatalf("round-trip mismatch: got %q, want %q", decrypted, plaintext)
		}
	}
}

// TestOptionCipherEncryptIsIdempotent 验证已加密值不重复加密（幂等）。
func TestOptionCipherEncryptIsIdempotent(t *testing.T) {
	cipher, _ := NewOptionCipher(testKey(t))
	encrypted, _ := cipher.Encrypt("secret")
	again, err := cipher.Encrypt(encrypted)
	if err != nil {
		t.Fatalf("re-encrypt: %v", err)
	}
	if again != encrypted {
		t.Fatal("expected re-encrypting ciphertext to be idempotent")
	}
}

// TestOptionCipherDecryptAcceptsPlaintext 验证解密器兼容历史明文（无前缀）。
func TestOptionCipherDecryptAcceptsPlaintext(t *testing.T) {
	cipher, _ := NewOptionCipher(testKey(t))
	got, err := cipher.Decrypt("historical-plaintext-secret")
	if err != nil {
		t.Fatalf("Decrypt plaintext: %v", err)
	}
	if got != "historical-plaintext-secret" {
		t.Fatalf("expected plaintext passthrough, got %q", got)
	}
}

// TestOptionCipherTransparentWhenNoKey 验证未配置密钥时为透明模式（开发环境）。
func TestOptionCipherTransparentWhenNoKey(t *testing.T) {
	cipher, err := NewOptionCipher("")
	if err != nil {
		t.Fatalf("NewOptionCipher empty: %v", err)
	}
	if cipher.Enabled() {
		t.Fatal("expected disabled cipher when key empty")
	}
	got, err := cipher.Encrypt("plain")
	if err != nil || got != "plain" {
		t.Fatalf("transparent encrypt expected 'plain', got %q err %v", got, err)
	}
	got, err = cipher.Decrypt("plain")
	if err != nil || got != "plain" {
		t.Fatalf("transparent decrypt expected 'plain', got %q err %v", got, err)
	}
}

// TestOptionCipherRejectsInvalidKey 验证非法密钥格式被拒。
func TestOptionCipherRejectsInvalidKey(t *testing.T) {
	for _, bad := range []string{"not-hex!", "abcd", "0123456789abcdef"} {
		if _, err := NewOptionCipher(bad); err == nil {
			t.Fatalf("expected error for invalid key %q", bad)
		}
	}
}

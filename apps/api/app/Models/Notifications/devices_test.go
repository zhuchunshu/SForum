package notifications

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizePushDeviceInputBounds(t *testing.T) {
	valid := RegisterPushDeviceInput{
		UserID:     7,
		DeviceID:   "0f0a1b2c-3d4e-5f60-7a8b-9c0d1e2f3a4b",
		Platform:   "ios",
		Token:      strings.Repeat("a", pushDeviceTokenMinRunes),
		AppVersion: "1.4.0",
		Locale:     "zh-CN",
		DeviceName: "iPhone 15",
	}
	if _, err := NormalizePushDeviceInput(valid); err != nil {
		t.Fatalf("minimal valid device rejected: %v", err)
	}

	upper := valid
	upper.Platform = "ANDROID"
	normalized, err := NormalizePushDeviceInput(upper)
	if err != nil {
		t.Fatalf("platform should normalize case-insensitively: %v", err)
	}
	if normalized.Platform != "android" {
		t.Fatalf("platform normalized to %q", normalized.Platform)
	}

	cases := []struct {
		name   string
		mutate func(*RegisterPushDeviceInput)
		want   error
	}{
		{"missing user", func(in *RegisterPushDeviceInput) { in.UserID = 0 }, ErrPushDeviceInvalid},
		{"short device id", func(in *RegisterPushDeviceInput) { in.DeviceID = "abc" }, ErrPushDeviceInvalid},
		{"device id charset", func(in *RegisterPushDeviceInput) { in.DeviceID = "has space/../../etc" }, ErrPushDeviceInvalid},
		{"unknown platform", func(in *RegisterPushDeviceInput) { in.Platform = "web" }, ErrPushDevicePlatformKnown},
		{"short token", func(in *RegisterPushDeviceInput) { in.Token = "short" }, ErrPushDeviceInvalid},
		{"token too long", func(in *RegisterPushDeviceInput) { in.Token = strings.Repeat("a", pushDeviceTokenMaxRunes+1) }, ErrPushDeviceTokenTooLong},
		{"token non ascii", func(in *RegisterPushDeviceInput) { in.Token = strings.Repeat("推", 32) }, ErrPushDeviceInvalid},
		{"app version too long", func(in *RegisterPushDeviceInput) { in.AppVersion = strings.Repeat("9", pushDeviceAppVersionMax+1) }, ErrPushDeviceInvalid},
		{"device name too long", func(in *RegisterPushDeviceInput) { in.DeviceName = strings.Repeat("名", pushDeviceNameMax+1) }, ErrPushDeviceInvalid},
		{"locale too long", func(in *RegisterPushDeviceInput) { in.Locale = strings.Repeat("x", pushDeviceLocaleMax+1) }, ErrPushDeviceInvalid},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			input := valid
			testCase.mutate(&input)
			if _, err := NormalizePushDeviceInput(input); !errors.Is(err, testCase.want) {
				t.Fatalf("expected %v, got %v", testCase.want, err)
			}
		})
	}
}

func TestPushDeviceTokenHashingIsStableAndOpaque(t *testing.T) {
	token := "fcm-registration-token-example-0001"
	first := hashPushDeviceToken(token)
	if first != hashPushDeviceToken(token) {
		t.Fatal("token hash must be stable for dedupe")
	}
	if len(first) != 64 {
		t.Fatalf("token hash must be a sha256 hex digest, got %d chars", len(first))
	}
	if strings.Contains(first, token) {
		t.Fatal("token hash must not embed the plaintext token")
	}
	if first == hashPushDeviceToken(token+"2") {
		t.Fatal("different tokens must hash differently")
	}
}

// 未注入 Core 密钥时（开发环境）保持明文；注入后必须存密文。
func TestPushDeviceTokenStorageDependsOnInjectedCipher(t *testing.T) {
	store := newPostgresStore(nil)
	stored, err := store.encryptPushDeviceToken("plain-token-value")
	if err != nil {
		t.Fatal(err)
	}
	if stored != "plain-token-value" {
		t.Fatalf("transparent cipher must keep the plaintext, got %q", stored)
	}

	store.WithPushDeviceCipher(upperCipher{})
	encrypted, err := store.encryptPushDeviceToken("plain-token-value")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "plain-token-value" || !strings.HasPrefix(encrypted, "cipher:") {
		t.Fatalf("expected encrypted storage, got %q", encrypted)
	}
}

type upperCipher struct{}

func (upperCipher) Encrypt(value string) (string, error) { return "cipher:" + value, nil }

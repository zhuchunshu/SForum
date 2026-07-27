package identity

import (
	"strings"
	"testing"
)

// 验证 Core-HMAC 模式下 parseAuthCompleteOutput 的双契约支持：
//  1. 插件返回 raw providerSubject → parsed.rawSubject 非空、subjectDigest 空；
//  2. 插件返回 providerSubjectDigest（兼容旧 fixture）→ parsed.subjectDigest 非空、rawSubject 空；
//  3. 二者皆空 → fail closed；
//  4. 两者同时存在 → 优先 rawSubject（Core-HMAC），丢弃插件 digest。

func TestParseAuthCompleteOutput_RawSubjectCoreHMACMode(t *testing.T) {
	ResetIdentitySubjectHMACKeyForTest()
	t.Setenv(IdentitySubjectHMACSecretEnv, "")
	t.Setenv("SFORUM_IN_PRODUCTION", "")

	parsed, err := parseAuthCompleteOutput(map[string]any{
		"providerSubject": "12345",
		"displayName":     "octocat",
		"emailHint":       "octo@example.com",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.rawSubject != "12345" {
		t.Fatalf("rawSubject = %q want 12345", parsed.rawSubject)
	}
	if parsed.subjectDigest != "" {
		t.Fatalf("subjectDigest should be empty in Core-HMAC mode, got %q", parsed.subjectDigest)
	}
}

func TestParseAuthCompleteOutput_LegacyDigestFixtureMode(t *testing.T) {
	// 兼容旧 fixture：插件返回 64-hex digest。
	legacyDigest := strings.Repeat("a", 64)
	parsed, err := parseAuthCompleteOutput(map[string]any{
		"providerSubjectDigest": legacyDigest,
		"displayName":           "member",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.rawSubject != "" {
		t.Fatalf("rawSubject should be empty in legacy mode, got %q", parsed.rawSubject)
	}
	if parsed.subjectDigest != legacyDigest {
		t.Fatalf("subjectDigest = %q want %q", parsed.subjectDigest, legacyDigest)
	}
}

func TestParseAuthCompleteOutput_BothPresentPrefersRawSubject(t *testing.T) {
	legacyDigest := strings.Repeat("b", 64)
	parsed, err := parseAuthCompleteOutput(map[string]any{
		"providerSubject":       "999",
		"providerSubjectDigest": legacyDigest,
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.rawSubject != "999" {
		t.Fatalf("rawSubject = %q want 999", parsed.rawSubject)
	}
	if parsed.subjectDigest != "" {
		t.Fatalf("subjectDigest must be cleared when rawSubject present, got %q", parsed.subjectDigest)
	}
}

func TestParseAuthCompleteOutput_BothEmptyFailsClosed(t *testing.T) {
	if _, err := parseAuthCompleteOutput(map[string]any{
		"displayName": "x",
	}); err == nil {
		t.Fatalf("expected fail-closed when neither subject nor digest present")
	}
	if _, err := parseAuthCompleteOutput(nil); err == nil {
		t.Fatalf("expected fail-closed on nil output")
	}
}

func TestParseAuthCompleteOutput_MalformedLegacyDigestFails(t *testing.T) {
	if _, err := parseAuthCompleteOutput(map[string]any{
		"providerSubjectDigest": "not-a-hex-digest",
	}); err == nil {
		t.Fatalf("expected fail-closed on malformed digest")
	}
}

func TestParseAuthCompleteOutput_OverlongSubjectFails(t *testing.T) {
	long := strings.Repeat("x", 321)
	if _, err := parseAuthCompleteOutput(map[string]any{
		"providerSubject": long,
	}); err == nil {
		t.Fatalf("expected fail-closed on overlong raw subject")
	}
}

package identity

import (
	"fmt"
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("expected password hash")
	}

	ok, err := VerifyPassword("correct horse battery staple", hash)
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected password to verify")
	}
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	ok, err := VerifyPassword("wrong password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to fail")
	}
}

func TestVerifyPasswordRejectsUnsafeArgon2Parameters(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	tests := []struct {
		name        string
		oldParam    string
		unsafeParam string
	}{
		{name: "memory zero", oldParam: "m=65536", unsafeParam: "m=0"},
		{name: "memory below Argon2 minimum", oldParam: "m=65536", unsafeParam: "m=31"},
		{name: "memory above cost limit", oldParam: "m=65536", unsafeParam: "m=65537"},
		{name: "memory uint32 wrap", oldParam: "m=65536", unsafeParam: "m=4295032832"},
		{name: "time zero", oldParam: "t=1", unsafeParam: "t=0"},
		{name: "time above cost limit", oldParam: "t=1", unsafeParam: "t=2"},
		{name: "time uint32 wrap", oldParam: "t=1", unsafeParam: "t=4294967297"},
		{name: "threads zero", oldParam: "p=4", unsafeParam: "p=0"},
		{name: "threads above cost limit", oldParam: "p=4", unsafeParam: "p=5"},
		{name: "threads uint8 wrap", oldParam: "p=4", unsafeParam: "p=260"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsafeHash := strings.Replace(hash, tt.oldParam, tt.unsafeParam, 1)
			if ok, err := VerifyPassword("correct horse battery staple", unsafeHash); err == nil || ok {
				t.Fatalf("expected unsafe params to fail before Argon2, ok=%v err=%v", ok, err)
			}
		})
	}
}

func TestVerifyPasswordRejectsUnexpectedSaltAndKeyLengths(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Fatalf("unexpected generated hash format: %q", hash)
	}

	for _, index := range []int{4, 5} {
		t.Run(fmt.Sprintf("part %d", index), func(t *testing.T) {
			changed := append([]string(nil), parts...)
			changed[index] += "AA"
			if ok, err := VerifyPassword("correct horse battery staple", strings.Join(changed, "$")); err == nil || ok {
				t.Fatalf("expected unexpected hash part length to fail, ok=%v err=%v", ok, err)
			}
		})
	}
}

func TestPasswordPolicyValidatesLengthAndRequiredClasses(t *testing.T) {
	policy := PasswordPolicy{
		MinLength:        10,
		MaxLength:        20,
		RequireLowercase: true,
		RequireUppercase: true,
		RequireNumber:    true,
		RequireSymbol:    true,
	}

	fields := policy.Validate("short")
	if len(fields[FieldPassword]) == 0 {
		t.Fatal("expected password policy errors")
	}
	if !fieldMessagesContain(fields, FieldPassword, MessagePasswordMin) {
		t.Fatalf("expected min length error, got %#v", fields)
	}
	if !fieldMessagesContain(fields, FieldPassword, MessagePasswordUppercase) {
		t.Fatalf("expected uppercase error, got %#v", fields)
	}

	fields = policy.Validate("Correct-1234")
	if len(fields) != 0 {
		t.Fatalf("expected valid password, got %#v", fields)
	}
}

func TestPasswordPolicyCountsUnicodeRunes(t *testing.T) {
	policy := PasswordPolicy{MinLength: 4, MaxLength: 8}

	if fields := policy.Validate("密码短"); !fieldMessagesContain(fields, FieldPassword, MessagePasswordMin) {
		t.Fatalf("expected unicode rune min length error, got %#v", fields)
	}
	if fields := policy.Validate("密码短句"); len(fields) != 0 {
		t.Fatalf("expected unicode password to pass, got %#v", fields)
	}
}

func TestHashPasswordDoesNotOwnRuntimePolicy(t *testing.T) {
	hash, err := HashPassword("short")
	if err != nil {
		t.Fatalf("HashPassword should only hash, got %v", err)
	}
	ok, err := VerifyPassword("short", hash)
	if err != nil || !ok {
		t.Fatalf("expected short password hash to verify, ok=%v err=%v", ok, err)
	}
}

func fieldMessagesContain(fields FieldMessages, field string, message string) bool {
	for _, item := range fields[field] {
		if item == message {
			return true
		}
	}
	return false
}

// TestDummyPasswordHashForTimingAlignment 验证 L1：dummy hash 用于登录时序对齐。
// 它必须是合法的 argon2id 格式（VerifyPassword 不报错），且对任意密码验证结果为 false。
func TestDummyPasswordHashForTimingAlignment(t *testing.T) {
	hash, err := dummyPasswordHash()
	if err != nil {
		t.Fatalf("dummyPasswordHash returned error: %v", err)
	}
	// 重复调用应返回缓存的同一 hash（sync.Once 复用）。
	hash2, _ := dummyPasswordHash()
	if hash != hash2 {
		t.Fatal("expected dummyPasswordHash to return cached hash on second call")
	}
	// 对任意密码验证结果为 false，但不报错（格式合法，会跑完 argon2）。
	ok, err := VerifyPassword("any-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword on dummy hash returned error: %v", err)
	}
	if ok {
		t.Fatal("expected dummy hash to never verify any password")
	}
}

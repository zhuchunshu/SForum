package identity

import "testing"

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

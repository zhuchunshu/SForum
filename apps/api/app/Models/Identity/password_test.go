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

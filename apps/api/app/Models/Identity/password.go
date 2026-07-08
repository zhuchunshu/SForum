package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

const (
	passwordSaltBytes = 16
	passwordTime      = uint32(1)
	passwordMemory    = uint32(64 * 1024)
	passwordThreads   = uint8(4)
	passwordKeyBytes  = uint32(32)
)

const (
	RecommendedPasswordMinLength = 12
	RecommendedPasswordMaxLength = 128
)

type PasswordPolicy struct {
	MinLength        int
	MaxLength        int
	RequireLowercase bool
	RequireUppercase bool
	RequireNumber    bool
	RequireSymbol    bool
}

func RecommendedPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{MinLength: RecommendedPasswordMinLength, MaxLength: RecommendedPasswordMaxLength}
}

func (p PasswordPolicy) Normalized() PasswordPolicy {
	if p.MinLength <= 0 {
		p.MinLength = RecommendedPasswordMinLength
	}
	if p.MaxLength <= 0 {
		p.MaxLength = RecommendedPasswordMaxLength
	}
	if p.MaxLength < p.MinLength {
		p.MaxLength = p.MinLength
	}
	return p
}

func (p PasswordPolicy) Validate(password string) FieldMessages {
	p = p.Normalized()
	fields := FieldMessages{}
	length := len([]rune(password))
	if length < p.MinLength {
		addFieldMessage(fields, FieldPassword, MessagePasswordMin)
	}
	if length > p.MaxLength {
		addFieldMessage(fields, FieldPassword, MessagePasswordMax)
	}

	var hasLower, hasUpper, hasNumber, hasSymbol bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsNumber(r):
			hasNumber = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}
	if p.RequireLowercase && !hasLower {
		addFieldMessage(fields, FieldPassword, MessagePasswordLowercase)
	}
	if p.RequireUppercase && !hasUpper {
		addFieldMessage(fields, FieldPassword, MessagePasswordUppercase)
	}
	if p.RequireNumber && !hasNumber {
		addFieldMessage(fields, FieldPassword, MessagePasswordNumber)
	}
	if p.RequireSymbol && !hasSymbol {
		addFieldMessage(fields, FieldPassword, MessagePasswordSymbol)
	}
	return fields
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read password salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, passwordTime, passwordMemory, passwordThreads, passwordKeyBytes)
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedKey := base64.RawStdEncoding.EncodeToString(key)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", passwordMemory, passwordTime, passwordThreads, encodedSalt, encodedKey), nil
}

func VerifyPassword(password string, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false, fmt.Errorf("invalid password hash format")
	}

	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return false, fmt.Errorf("invalid password hash params")
	}

	memory, err := parseParam(params[0], "m")
	if err != nil {
		return false, err
	}
	time, err := parseParam(params[1], "t")
	if err != nil {
		return false, err
	}
	threads, err := parseParam(params[2], "p")
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("decode password salt: %w", err)
	}
	expectedKey, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("decode password key: %w", err)
	}

	actualKey := argon2.IDKey([]byte(password), salt, uint32(time), uint32(memory), uint8(threads), uint32(len(expectedKey)))
	return subtle.ConstantTimeCompare(actualKey, expectedKey) == 1, nil
}

func parseParam(value string, key string) (int, error) {
	prefix := key + "="
	if !strings.HasPrefix(value, prefix) {
		return 0, fmt.Errorf("missing password hash param %s", key)
	}
	parsed, err := strconv.Atoi(strings.TrimPrefix(value, prefix))
	if err != nil {
		return 0, fmt.Errorf("parse password hash param %s: %w", key, err)
	}
	return parsed, nil
}

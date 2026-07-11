package identity

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Validate 按运营配置的用户名策略校验。策略零值时仅要求非空（由调用方处理）。
func (p UsernamePolicy) Validate(username string) (ok bool, reason string) {
	username = strings.TrimSpace(username)
	if username == "" {
		return false, MessageUsernameRequired
	}
	count := utf8.RuneCountInString(username)
	if p.MinLength > 0 && count < p.MinLength {
		return false, MessageUsernameTooShort
	}
	if p.MaxLength > 0 && count > p.MaxLength {
		return false, MessageUsernameTooLong
	}
	lower := strings.ToLower(username)
	for _, reserved := range p.Reserved {
		if lower == strings.ToLower(strings.TrimSpace(reserved)) {
			return false, MessageUsernameReserved
		}
	}
	if p.Charset == "ascii" {
		for _, r := range username {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
				return false, MessageUsernameCharset
			}
		}
		return true, ""
	}
	if p.Charset == "unicode_letters_numbers" || p.Charset != "" {
		for _, r := range username {
			if unicode.IsSpace(r) || unicode.IsControl(r) {
				return false, MessageUsernameCharset
			}
			if !(unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' || r == '-') {
				return false, MessageUsernameCharset
			}
		}
	}
	return true, ""
}

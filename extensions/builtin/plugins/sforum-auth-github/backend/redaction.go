package main

import (
	"regexp"
	"strings"
)

// 敏感材料匹配：code、token、secret、verifier、state 等不得进入错误消息。
var (
	redactBearer   = regexp.MustCompile(`(?i)bearer\s+[a-z0-9._\-+/=]{8,}`)
	redactKVSecret = regexp.MustCompile(`(?i)(client_secret|access_token|refresh_token|code_verifier|code=)[=:]\s*[^\s&,"']+`)
	redactLongHex  = regexp.MustCompile(`\b[a-f0-9]{32,}\b`)
)

// RedactSensitive 擦除错误/诊断字符串中的密钥、令牌与类凭证材料。
// 用于 fail-closed 返回给 Host 的 reason 文本；Host 也不应记录 raw 主体。
func RedactSensitive(message string) string {
	if message == "" {
		return message
	}
	out := message
	out = redactBearer.ReplaceAllString(out, "bearer [redacted]")
	out = redactKVSecret.ReplaceAllString(out, "$1=[redacted]")
	out = redactLongHex.ReplaceAllString(out, "[redacted]")
	// 额外擦除常见密钥字段字面量泄漏。
	for _, needle := range []string{
		"client_secret", "access_token", "code_verifier", "clientSecret",
	} {
		if strings.Contains(strings.ToLower(out), strings.ToLower(needle)) {
			// 已由正则处理大部分；保留字面量时整段降级为通用错误。
			if strings.Contains(out, "=") || strings.Contains(out, ":") {
				return "github oauth request failed"
			}
		}
	}
	return out
}

// ErrPublic 是可安全返回给 Host 的协议错误（已脱敏）。
type ErrPublic struct {
	Reason  string
	Message string
}

func (e *ErrPublic) Error() string {
	if e == nil {
		return "github oauth error"
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Reason
}

func publicErr(reason, message string) error {
	return &ErrPublic{Reason: reason, Message: RedactSensitive(message)}
}

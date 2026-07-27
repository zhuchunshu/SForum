package identity

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"
)

// AuthProviderProbeResult 是 Host 解析后的有界探测结果。
// Reason 为稳定短码；Message 已脱敏，可进入 audit/API。
type AuthProviderProbeResult struct {
	ProviderID            string
	OK                    bool
	Reason                string
	Message               string
	OwnerExtensionID      string
	OwnerExtensionVersion string
	OwnerPackageDigest    string
}

// Probe 调用选定 auth 提供方的 provider.probe 操作。
//
// 约束：
//   - 仅当 live Registry 有 RuntimeInstanceID 且声明了 provider.probe 时执行；
//   - invoker 使用 operation TimeoutMS 作为 deadline；
//   - 返回的 reason/message 经 Host 再脱敏，禁止密钥/令牌/raw subject；
//   - 不写 activation、不签发会话、不创建用户。
func (f *AuthProviderFlow) Probe(ctx context.Context, providerID string) (AuthProviderProbeResult, error) {
	if f == nil || f.source == nil || f.invoker == nil {
		return AuthProviderProbeResult{}, ErrAuthProviderFlowUnavailable
	}
	providerID = strings.ToLower(strings.TrimSpace(providerID))
	if providerID == "" {
		return AuthProviderProbeResult{}, ErrAuthProviderFlowInvalid
	}
	provider, err := f.resolveAuthProvider(ctx, providerID, AuthOperationProviderProbe)
	if err != nil {
		return AuthProviderProbeResult{}, err
	}

	var result AuthProviderProbeResult
	err = f.invoker.InvokeExact(
		ctx, provider, AuthOperationProviderProbe, 0, map[string]any{},
		func(_ context.Context, output map[string]any, fence func() error) error {
			parsed, parseErr := parseAuthProbeOutput(output)
			if parseErr != nil {
				return parseErr
			}
			if fence != nil {
				if fenceErr := fence(); fenceErr != nil {
					return fenceErr
				}
			}
			result = AuthProviderProbeResult{
				ProviderID:            provider.ID,
				OK:                    parsed.ok,
				Reason:                parsed.reason,
				Message:               parsed.message,
				OwnerExtensionID:      provider.Artifact.ExtensionID,
				OwnerExtensionVersion: provider.Artifact.ExtensionVersion,
				OwnerPackageDigest:    provider.Artifact.PackageDigest,
			}
			return nil
		},
	)
	if err != nil {
		return AuthProviderProbeResult{}, mapAuthProviderInvokeError(err)
	}
	if result.ProviderID == "" || result.Reason == "" {
		return AuthProviderProbeResult{}, ErrAuthProviderFlowUnavailable
	}
	return result, nil
}

type authProbeParsed struct {
	ok      bool
	reason  string
	message string
}

func parseAuthProbeOutput(output map[string]any) (authProbeParsed, error) {
	if output == nil {
		return authProbeParsed{}, ErrAuthProviderFlowUnavailable
	}
	ok, okPresent := boolFromAuthOutput(output, "ok")
	if !okPresent {
		return authProbeParsed{}, ErrAuthProviderFlowUnavailable
	}
	reason := redactProbeText(stringFromAuthOutput(output, "reason"), 120)
	if reason == "" {
		return authProbeParsed{}, ErrAuthProviderFlowUnavailable
	}
	// 拒绝把历史占位 reason 当作真实探测成功路径的实现信号。
	if reason == ProbeReasonPending {
		return authProbeParsed{}, ErrAuthProviderFlowUnavailable
	}
	message := redactProbeText(stringFromAuthOutput(output, "message"), 500)
	return authProbeParsed{ok: ok, reason: reason, message: message}, nil
}

func boolFromAuthOutput(output map[string]any, key string) (bool, bool) {
	raw, found := output[key]
	if !found || raw == nil {
		return false, false
	}
	switch v := raw.(type) {
	case bool:
		return v, true
	default:
		return false, false
	}
}

var (
	probeRedactBearer   = regexp.MustCompile(`(?i)bearer\s+[a-z0-9._\-+/=]{8,}`)
	probeRedactKVSecret = regexp.MustCompile(`(?i)(client_secret|access_token|refresh_token|code_verifier|code=)[=:]\s*[^\s&,"']+`)
	probeRedactLongHex  = regexp.MustCompile(`\b[a-f0-9]{32,}\b`)
)

// redactProbeText 对 probe reason/message 做 Host 侧脱敏与长度裁剪。
func redactProbeText(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	out := probeRedactBearer.ReplaceAllString(value, "bearer [redacted]")
	out = probeRedactKVSecret.ReplaceAllString(out, "$1=[redacted]")
	out = probeRedactLongHex.ReplaceAllString(out, "[redacted]")
	if maxRunes > 0 && utf8.RuneCountInString(out) > maxRunes {
		runes := []rune(out)
		out = string(runes[:maxRunes])
	}
	return out
}

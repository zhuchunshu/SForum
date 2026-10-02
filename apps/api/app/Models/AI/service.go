// Package ai 是权限感知的 AI 网关服务层。Support/AI 保持纯基础设施：契约、
// 协议翻译、存储与执行；这里回答「谁能配置、谁能查看」。
package ai

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	supportai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
)

// CredentialStore 是密钥写入与状态查询的最小面。它以接口声明，让服务层不必
// 依赖 Secret Store 的具体实现。
type CredentialStore interface {
	// Put 写入新版本并返回版本号。
	Put(ctx context.Context, reference, plaintext, actor string) (int64, error)
	// Configured 报告该引用是否已有可用密钥。
	Configured(ctx context.Context, reference string) (bool, error)
}

type Service struct {
	settings    supportai.SettingsStore
	gateway     *supportai.Gateway
	usage       supportai.UsageStore
	traces      supportai.ExecutionStore
	credentials CredentialStore
}

// Config 汇总服务层依赖。除 Settings 外均可缺省，缺省时对应视图返回空结果
// 而不是报错，让一个尚未接入存储的部署仍能打开配置页。
type Config struct {
	Settings    supportai.SettingsStore
	Gateway     *supportai.Gateway
	Usage       supportai.UsageStore
	Traces      supportai.ExecutionStore
	Credentials CredentialStore
}

func NewService(cfg Config) *Service {
	return &Service{
		settings:    cfg.Settings,
		gateway:     cfg.Gateway,
		usage:       cfg.Usage,
		traces:      cfg.Traces,
		credentials: cfg.Credentials,
	}
}

// Gateway 暴露底层网关，供 Core 用途（审核建议、摘要等）直接调用。它不经过
// 权限校验：调用方是服务端流程，已经完成了自己的授权。
func (s *Service) Gateway() *supportai.Gateway {
	if s == nil {
		return nil
	}
	return s.gateway
}

func (s *Service) GetSettings(ctx context.Context, actor identity.Actor) (supportai.Settings, error) {
	if !actor.Can(identity.PermissionAIManage) {
		return supportai.Settings{}, identity.ErrPermissionDenied
	}
	if s == nil || s.settings == nil {
		return supportai.RecommendedSettings().Normalized(), nil
	}
	return s.settings.GetSettings(ctx)
}

// UpdateSettings 在写入前做策略校验。越界值直接拒绝，而不是截断到边界。
func (s *Service) UpdateSettings(ctx context.Context, actor identity.Actor, settings supportai.Settings) (supportai.Settings, error) {
	if !actor.Can(identity.PermissionAIManage) {
		return supportai.Settings{}, identity.ErrPermissionDenied
	}
	if err := settings.Validate(); err != nil {
		return supportai.Settings{}, err
	}
	if s == nil || s.settings == nil {
		return supportai.Settings{}, supportai.ErrSettingsInvalid
	}
	return s.settings.SaveSettings(ctx, settings, actor.ID)
}

// ResetSettings 恢复推荐默认值。它不会清空 Secret Store 中的密钥：密钥值不在
// 配置文档里，重置只影响引用与策略。
func (s *Service) ResetSettings(ctx context.Context, actor identity.Actor) (supportai.Settings, error) {
	if !actor.Can(identity.PermissionAIManage) {
		return supportai.Settings{}, identity.ErrPermissionDenied
	}
	if s == nil || s.settings == nil {
		return supportai.RecommendedSettings().Normalized(), nil
	}
	return s.settings.ResetSettings(ctx, supportai.RecommendedSettings(), actor.ID)
}

// Usage 返回站点级用量，供成本看板展示。它不带发起方维度：那是 trace 的职责。
func (s *Service) Usage(ctx context.Context, actor identity.Actor, now time.Time) (supportai.UsageByScope, error) {
	if !actor.Can(identity.PermissionAIManage) {
		return supportai.UsageByScope{}, identity.ErrPermissionDenied
	}
	if s == nil || s.usage == nil {
		return supportai.UsageByScope{}, nil
	}
	return s.usage.Snapshot(ctx, now, "", 0)
}

// SaveProfileCredential 为指定 profile 写入新密钥。明文只在这一次请求里存在：
// 它被写进 Secret Store 后立即丢弃，设置文档与后续读取都只看到引用。
func (s *Service) SaveProfileCredential(ctx context.Context, actor identity.Actor, profileID, plaintext string) (int64, error) {
	if !actor.Can(identity.PermissionAIManage) {
		return 0, identity.ErrPermissionDenied
	}
	if s == nil || s.credentials == nil || s.settings == nil {
		return 0, supportai.ErrCredentialMissing
	}
	settings, err := s.settings.GetSettings(ctx)
	if err != nil {
		return 0, err
	}
	profile, ok := settings.Profile(profileID)
	if !ok {
		return 0, supportai.ErrProfileUnavailable
	}
	return s.credentials.Put(ctx, profile.APIKeyRef, plaintext, actorDisplayName(actor))
}

// Diagnose 发一次真实的最小调用，回答「现在这套配置能不能用」。前三类问题
// （开关关闭、没有可用提供商、密钥缺失）在本地就能判定，不必花钱发请求；
// 剩下的一律走完整的网关路径，因此延迟、用量、失败原因都是真的。
//
// 它刻意不绕过闸门：被配额拦住本身就是一条诊断结论，而且绕过会让「测试」
// 成为配额系统的一个后门。
func (s *Service) Diagnose(ctx context.Context, actor identity.Actor, message string) (supportai.DiagnoseResult, error) {
	if !actor.Can(identity.PermissionAIManage) {
		return supportai.DiagnoseResult{}, identity.ErrPermissionDenied
	}
	if s == nil || s.settings == nil || s.gateway == nil {
		return supportai.DiagnoseResult{
			Reason: supportai.DiagnoseReasonDisabled,
			Error:  "AI gateway is not wired in this deployment",
		}, nil
	}
	settings, err := s.settings.GetSettings(ctx)
	if err != nil {
		return supportai.DiagnoseResult{}, err
	}
	if !settings.Enabled {
		return supportai.DiagnoseResult{Reason: supportai.DiagnoseReasonDisabled}, nil
	}
	profile, err := settings.ResolveProfile(supportai.CostClassEconomy, "")
	if err != nil {
		return supportai.DiagnoseResult{Reason: supportai.DiagnoseReasonNoProvider}, nil
	}
	result := supportai.DiagnoseResult{
		ProviderID: profile.ID,
		Model:      profile.Model,
		Protocol:   profile.Protocol,
	}
	configured := false
	if s.credentials != nil {
		configured, err = s.credentials.Configured(ctx, profile.APIKeyRef)
		if err != nil {
			return supportai.DiagnoseResult{}, err
		}
	}
	if !configured {
		result.Reason = supportai.DiagnoseReasonNoCredential
		return result, nil
	}
	probe := strings.TrimSpace(message)
	if probe == "" {
		probe = supportai.DefaultDiagnoseMessage
	}
	completion, err := s.gateway.Execute(ctx, supportai.CompletionRequest{
		Purpose:       supportai.DiagnosePurpose,
		CostClass:     supportai.CostClassEconomy,
		System:        "这是一次连通性探测。严格按用户的要求回复，不要添加任何其它内容。",
		Messages:      []supportai.Message{{Role: supportai.RoleUser, Parts: []supportai.Part{{Type: supportai.PartText, Text: probe}}}},
		MaxTokens:     64,
		PromptVersion: supportai.DiagnosePromptVersion,
	})
	if err != nil {
		var unavailable *supportai.UnavailableError
		if errors.As(err, &unavailable) {
			result.Reason = unavailable.Reason
			result.Error = unavailable.Error()
			return result, nil
		}
		result.Reason = supportai.DiagnoseReasonProviderFailed
		result.Error = err.Error()
		return result, nil
	}
	result.OK = true
	result.Reply = completion.Text
	result.LatencyMS = completion.LatencyMS
	result.InputTokens = completion.Usage.InputTokens
	result.OutputTokens = completion.Usage.OutputTokens
	return result, nil
}

// CredentialStatus 返回每个 profile 的密钥配置状态，供控制台显示「已配置」。
// 它永远不返回密钥值本身。
func (s *Service) CredentialStatus(ctx context.Context, actor identity.Actor) (map[string]bool, error) {
	if !actor.Can(identity.PermissionAIManage) {
		return nil, identity.ErrPermissionDenied
	}
	status := map[string]bool{}
	if s == nil || s.credentials == nil || s.settings == nil {
		return status, nil
	}
	settings, err := s.settings.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	for _, profile := range settings.Profiles {
		configured, err := s.credentials.Configured(ctx, profile.APIKeyRef)
		if err != nil {
			return nil, err
		}
		status[profile.ID] = configured
	}
	return status, nil
}

// actorDisplayName 用于 Secret Store 审计。Actor 只记录谁改过，不含密钥值。
func actorDisplayName(actor identity.Actor) string {
	if actor.ID <= 0 {
		return ""
	}
	return "user:" + strconv.FormatInt(actor.ID, 10)
}

func (s *Service) Executions(ctx context.Context, actor identity.Actor, limit int) ([]supportai.ExecutionRecord, error) {
	if !actor.Can(identity.PermissionAIManage) {
		return nil, identity.ErrPermissionDenied
	}
	if s == nil || s.traces == nil {
		return []supportai.ExecutionRecord{}, nil
	}
	return s.traces.ListExecutions(ctx, limit)
}

package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"
)

var (
	// ErrGatewayDisabled 表示站点总开关关闭，不产生任何出站调用。
	ErrGatewayDisabled = errors.New("ai: gateway is disabled")
)

// CredentialResolver 从 Secret Store 解析 profile 的密钥引用。网关只拿引用，
// 不持有密钥值；解析出来的明文仅用于构造一次请求。
type CredentialResolver interface {
	ResolveAIKey(ctx context.Context, reference string) (string, error)
}

// ProviderInvoker 由扩展运行时实现：把协议请求交给选中的 provider 插件。
// 它负责 SSRF 校验、超时、重试与出站审计；网关只负责契约翻译、闸门与记账。
type ProviderInvoker interface {
	Invoke(ctx context.Context, profile Profile, request WireRequest) (WireResponse, error)
}

// ResultCache 是可选的结果缓存。为 nil 时网关不做缓存。
type ResultCache interface {
	Get(ctx context.Context, key string) (CompletionResult, bool)
	Put(ctx context.Context, key string, result CompletionResult, ttl time.Duration)
}

// UnavailableError 表示 AI 不可用，并附带该用途声明的失败姿态。调用方用
// errors.As 取出 Posture 后决定如何降级，而不是把降级策略写死在网关里。
type UnavailableError struct {
	Posture string
	Reason  string
	Err     error
}

func (e *UnavailableError) Error() string {
	if e == nil {
		return "ai: unavailable"
	}
	if e.Err != nil {
		return "ai: unavailable (" + e.Reason + "): " + e.Err.Error()
	}
	return "ai: unavailable (" + e.Reason + ")"
}

func (e *UnavailableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// GatewayConfig 汇总网关依赖。除 SettingsStore 外全部可缺省，缺省即为停用该能力。
type GatewayConfig struct {
	Settings SettingsStore
	Usage    UsageStore
	Traces   ExecutionStore
	Creds    CredentialResolver
	Invoker  ProviderInvoker
	Cache    ResultCache
	// Clock 可注入，便于测试窗口行为。
	Clock func() time.Time
	// CacheTTL 为 0 时使用 DefaultCacheTTL。
	CacheTTL time.Duration
}

// DefaultCacheTTL 是结果缓存的默认存活时间。
const DefaultCacheTTL = 10 * time.Minute

// MaxErrorSummaryLength 限制写入 trace 的错误摘要长度。
const MaxErrorSummaryLength = 500

type Gateway struct {
	settings SettingsStore
	usage    UsageStore
	traces   ExecutionStore
	creds    CredentialResolver
	invoker  ProviderInvoker
	cache    ResultCache
	clock    func() time.Time
	cacheTTL time.Duration
}

func NewGateway(config GatewayConfig) *Gateway {
	clock := config.Clock
	if clock == nil {
		clock = time.Now
	}
	ttl := config.CacheTTL
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}
	return &Gateway{
		settings: config.Settings,
		usage:    config.Usage,
		traces:   config.Traces,
		creds:    config.Creds,
		invoker:  config.Invoker,
		cache:    config.Cache,
		clock:    clock,
		cacheTTL: ttl,
	}
}

// Settings 返回当前生效配置，供调用方读取失败姿态与开关。
func (g *Gateway) Settings(ctx context.Context) (Settings, error) {
	if g == nil || g.settings == nil {
		return RecommendedSettings().Normalized(), nil
	}
	return g.settings.GetSettings(ctx)
}

// Execute 是唯一的调用入口。它先判闸门再解析 profile，任何一步失败都会留下
// 一条 trace，使「为什么没有答案」永远可解释。
func (g *Gateway) Execute(ctx context.Context, request CompletionRequest) (CompletionResult, error) {
	request = request.Normalized()
	if err := request.Validate(); err != nil {
		return CompletionResult{}, err
	}
	settings, err := g.Settings(ctx)
	if err != nil {
		return CompletionResult{}, err
	}
	posture := settings.FailurePosture(request.Purpose)
	if !settings.Enabled {
		g.record(ctx, ExecutionRecord{
			Purpose: request.Purpose, CostClass: request.CostClass,
			CallerExtensionID: request.Metadata.CallerExtensionID, Status: ExecutionStatusDenied,
			GateReason: GateReasonDisabled, GateScope: GateScopeSite,
			PromptVersion: request.PromptVersion, ConfigRevision: settings.Revision,
			ErrorSummary: ErrGatewayDisabled.Error(),
		})
		return CompletionResult{}, &UnavailableError{Posture: posture, Reason: GateReasonDisabled, Err: ErrGatewayDisabled}
	}

	now := g.clock()
	decision, err := g.gate(ctx, settings, request, now)
	if err != nil {
		return CompletionResult{}, err
	}
	if !decision.Allowed {
		g.record(ctx, ExecutionRecord{
			Purpose: request.Purpose, CostClass: request.CostClass,
			CallerExtensionID: request.Metadata.CallerExtensionID, Status: ExecutionStatusDenied,
			GateReason: decision.Reason, GateScope: decision.Scope,
			PromptVersion: request.PromptVersion, ConfigRevision: settings.Revision,
		})
		return CompletionResult{}, &UnavailableError{
			Posture: posture, Reason: decision.Reason,
			Err: errors.New("ai: gate " + decision.Reason + " at " + decision.Scope),
		}
	}

	profile, err := settings.ResolveProfile(request.CostClass, request.ProfileRef)
	if err != nil {
		g.record(ctx, ExecutionRecord{
			Purpose: request.Purpose, CostClass: request.CostClass,
			CallerExtensionID: request.Metadata.CallerExtensionID, Status: ExecutionStatusFailed,
			PromptVersion: request.PromptVersion, ConfigRevision: settings.Revision,
			ErrorSummary: ErrProfileUnavailable.Error(),
		})
		return CompletionResult{}, &UnavailableError{Posture: posture, Reason: "profile_unavailable", Err: ErrProfileUnavailable}
	}

	if cached, ok := g.cached(ctx, request, profile, settings.Revision); ok {
		g.record(ctx, ExecutionRecord{
			Purpose: request.Purpose, CostClass: request.CostClass,
			CallerExtensionID: request.Metadata.CallerExtensionID, ProfileID: profile.ID,
			Protocol: profile.Protocol, Model: profile.Model, Status: ExecutionStatusSucceeded,
			PromptVersion: request.PromptVersion, ConfigRevision: settings.Revision,
			CacheHit: true,
		})
		return cached, nil
	}

	// 延迟必须用单调时钟测量，不能用可注入时钟：注入时钟的语义是「窗口对齐」，
	// 它被固定住时若同时承担耗时测量，真实调用的延迟会恒为 0。
	started := time.Now()
	result, err := g.invoke(ctx, settings, request, profile)
	latency := time.Since(started).Milliseconds()
	if err != nil {
		// 失败的调用同样消耗一次配额：否则一个持续失败的插件可以无限重试而
		// 永不耗尽配额，配额反而奖励了最不可靠的一方。token 记 0，因为本次
		// 没有产生可计费的产出。
		g.recordUsage(ctx, now, request, 0, 0)
		g.record(ctx, ExecutionRecord{
			Purpose: request.Purpose, CostClass: request.CostClass,
			CallerExtensionID: request.Metadata.CallerExtensionID, ProfileID: profile.ID,
			Protocol: profile.Protocol, Model: profile.Model, Status: ExecutionStatusFailed,
			PromptVersion: request.PromptVersion, ConfigRevision: settings.Revision,
			LatencyMS: int(latency), ErrorSummary: err.Error(),
		})
		return CompletionResult{}, &UnavailableError{Posture: posture, Reason: "provider_failed", Err: err}
	}
	result.LatencyMS = latency
	result.PromptVersion = request.PromptVersion
	result.ConfigRevision = settings.Revision
	result.Provider = ProviderArtifact{
		ExtensionID: request.Metadata.CallerExtensionID, Protocol: profile.Protocol, Model: profile.Model,
	}
	g.account(ctx, settings, request, profile, result, now)
	g.store(ctx, request, profile, settings.Revision, result)
	return result, nil
}

func (g *Gateway) gate(ctx context.Context, settings Settings, request CompletionRequest, now time.Time) (GateDecision, error) {
	if g.usage == nil {
		return GateDecision{Allowed: true}, nil
	}
	scopes, err := g.usage.Snapshot(ctx, now, request.Metadata.CallerExtensionID, request.SubjectUserID)
	if err != nil {
		return GateDecision{}, err
	}
	return EvaluateGates(GateInput{
		Enabled:        settings.Enabled,
		Gates:          settings.Gates,
		Site:           scopes.Site,
		Extension:      scopes.Extension,
		ExtensionBound: request.Metadata.CallerExtensionID != "",
		User:           scopes.User,
		UserBound:      request.SubjectUserID > 0,
	}), nil
}

func (g *Gateway) invoke(ctx context.Context, settings Settings, request CompletionRequest, profile Profile) (CompletionResult, error) {
	if g.invoker == nil || g.creds == nil {
		return CompletionResult{}, errors.New("ai: provider invoker is not wired")
	}
	apiKey, err := g.creds.ResolveAIKey(ctx, profile.APIKeyRef)
	if err != nil {
		return CompletionResult{}, fmt.Errorf("%w: %v", ErrCredentialMissing, err)
	}
	wire, err := BuildWireRequest(profile, apiKey, request)
	if err != nil {
		return CompletionResult{}, err
	}
	response, err := g.invoker.Invoke(ctx, profile, wire)
	if err != nil {
		return CompletionResult{}, err
	}
	return CompletionResult{
		Text:       response.Text,
		StopReason: response.StopReason,
		Usage:      response.Usage,
	}, nil
}

// account 同时写用量计数与执行 trace。两者都写，因为限额判定与成本看板需要
// 不同粒度：计数是可累加的窗口，trace 是逐次的归因。
func (g *Gateway) recordUsage(ctx context.Context, now time.Time, request CompletionRequest, inputTokens, outputTokens int) {
	if g.usage == nil {
		return
	}
	// 计次与计费分开：失败的尝试占用一次配额，但不产生金额（没有可计费产出）。
	_ = g.usage.Record(ctx, now, UsageEntry{
		CallerExtensionID: request.Metadata.CallerExtensionID,
		CallerUserID:      request.SubjectUserID,
		InputTokens:       inputTokens,
		OutputTokens:      outputTokens,
	})
}

func (g *Gateway) account(ctx context.Context, settings Settings, request CompletionRequest, profile Profile, result CompletionResult, now time.Time) {
	spend := EstimateSpendMicros(profile.Price, result.Usage)
	if g.usage != nil {
		_ = g.usage.Record(ctx, now, UsageEntry{
			CallerExtensionID: request.Metadata.CallerExtensionID,
			CallerUserID:      request.SubjectUserID,
			InputTokens:       result.Usage.InputTokens,
			OutputTokens:      result.Usage.OutputTokens,
			SpendMicros:       spend,
		})
	}
	g.record(ctx, ExecutionRecord{
		Purpose: request.Purpose, CostClass: request.CostClass,
		CallerExtensionID: request.Metadata.CallerExtensionID, ProfileID: profile.ID,
		Protocol: profile.Protocol, Model: profile.Model, Status: ExecutionStatusSucceeded,
		PromptVersion: request.PromptVersion, ConfigRevision: settings.Revision,
		LatencyMS: int(result.LatencyMS), InputTokens: result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens, CachedTokens: result.Usage.CachedTokens,
		SpendMicros: spend,
	})
}

func (g *Gateway) store(ctx context.Context, request CompletionRequest, profile Profile, revision int64, result CompletionResult) {
	if g.cache == nil {
		return
	}
	g.cache.Put(ctx, CacheKey(request, profile, revision), result, g.cacheTTL)
}

func (g *Gateway) cached(ctx context.Context, request CompletionRequest, profile Profile, revision int64) (CompletionResult, bool) {
	if g.cache == nil {
		return CompletionResult{}, false
	}
	return g.cache.Get(ctx, CacheKey(request, profile, revision))
}

func (g *Gateway) record(ctx context.Context, entry ExecutionRecord) {
	if g.traces == nil {
		return
	}
	entry.ErrorSummary = TruncateBytes(entry.ErrorSummary, MaxErrorSummaryLength)
	_ = g.traces.RecordExecution(ctx, entry)
}

// CacheKey 把 purpose、调用方插件、profile、模型、提示词版本与请求正文一起
// 摘要。任一维度不同都产生不同键，两个插件因此不可能读到对方的结果。
func CacheKey(request CompletionRequest, profile Profile, revision int64) string {
	hash := sha256.New()
	write := func(parts ...string) {
		for _, part := range parts {
			hash.Write([]byte(part))
			hash.Write([]byte{0})
		}
	}
	write("ai.cache@1", request.Purpose, request.Metadata.CallerExtensionID, profile.ID,
		profile.Model, request.PromptVersion, request.ResponseFormat,
		strconv.FormatInt(revision, 10), request.System)
	for _, message := range request.Messages {
		write(string(message.Role))
		for _, part := range message.Parts {
			write(part.Text)
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}

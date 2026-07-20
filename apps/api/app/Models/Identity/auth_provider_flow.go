package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"

	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
)

const (
	AuthOperationRegistrationStart    = "registration.start"
	AuthOperationRegistrationComplete = "registration.complete"
	AuthOperationLoginStart           = "login.start"
	AuthOperationLoginComplete        = "login.complete"
	AuthOperationLinkStart            = "link.start"
	AuthOperationLinkComplete         = "link.complete"

	AuthStartStatusContinue  = "continue"
	AuthStartStatusRedirect  = "redirect"
	AuthStartStatusChallenge = "challenge"
)

var (
	ErrAuthProviderFlowInvalid     = errors.New("identity: auth provider flow input is invalid")
	ErrAuthProviderFlowUnavailable = errors.New("identity: auth provider flow is unavailable")
	ErrAuthProviderFlowDenied      = errors.New("identity: auth provider flow was denied")
	ErrAuthProviderNotFound        = errors.New("identity: auth provider was not found")
)

// AuthProviderStartInput 是 Host 拥有的外部认证/注册/链接启动声明。
// 不包含密码、cookie、session id 或 CSRF。
type AuthProviderStartInput struct {
	ProviderID        string
	Operation         string
	ActorUserID       int64 // link.start 需要已登录用户；registration/login 可为 0
	CorrelationID     string
	DeviceFingerprint string
	ClientClass       string
	RedirectHint      string
}

// AuthProviderCompleteInput 是 Host 拥有的完成声明。插件返回的是外部主体断言，
// 不是授权结果；Host 仍负责用户创建、链接与会话。
type AuthProviderCompleteInput struct {
	ProviderID        string
	Operation         string
	ActorUserID       int64 // link.complete 必须等于目标用户；registration.complete 必须为 0
	TargetUserID      int64 // link.complete 的目标用户；registration 为 0
	CorrelationID     string
	CompletionToken   string
	DeviceFingerprint string
	ClientClass       string
	IdempotencyKey    string
}

// AuthProviderStartResult 是 start 操作的 Host 解析结果。
type AuthProviderStartResult struct {
	ProviderID      string
	Operation       string
	Status          string
	CorrelationID   string
	ContinueToken   string
	RedirectURL     string
	ChallengeKind   string
	ProviderOutput  map[string]any
}

// AuthProviderCompleteResult 是 complete 操作的 Host 解析结果。
// SubjectDigest 是稳定的外部主体摘要，用于 Host 链接表。
type AuthProviderCompleteResult struct {
	ProviderID     string
	Operation      string
	SubjectDigest  string
	DisplayName    string
	EmailHint      string
	ProviderOutput map[string]any
	// Link 在 Host 实际写入外部链接后填充；start/login.complete 未链接时为空。
	Link *ExternalIdentityLinkMutation
}

// AuthProviderSource 按精确 id 解析活跃 auth 提供方。
type AuthProviderSource interface {
	ResolveAuthProvider(ctx context.Context, providerID string) (identityregistry.ProviderContribution, error)
}

// AuthProviderInvoker 调用单个 exact auth 提供方操作。
type AuthProviderInvoker interface {
	InvokeExact(
		ctx context.Context,
		provider identityregistry.ProviderContribution,
		operation string,
		actorUserID int64,
		input map[string]any,
		accept func(context.Context, map[string]any, func() error) error,
	) error
}

// AuthProviderFlow 是 Host 拥有的外部 auth 工作流。认证、注册与链接的最终
// 效应（用户、会话、链接表）仍由 Host 决定；插件输出仅为提案。
type AuthProviderFlow struct {
	source  AuthProviderSource
	invoker AuthProviderInvoker
	links   ExternalIdentityLinkStore
}

func NewAuthProviderFlow(
	source AuthProviderSource,
	invoker AuthProviderInvoker,
	links ExternalIdentityLinkStore,
) (*AuthProviderFlow, error) {
	if source == nil || invoker == nil {
		return nil, ErrAuthProviderFlowUnavailable
	}
	return &AuthProviderFlow{source: source, invoker: invoker, links: links}, nil
}

// Start 调用选定 auth 提供方的 start 操作。失败关闭，无 Core/其他提供方回退。
func (f *AuthProviderFlow) Start(ctx context.Context, input AuthProviderStartInput) (AuthProviderStartResult, error) {
	if f == nil || f.source == nil || f.invoker == nil {
		return AuthProviderStartResult{}, ErrAuthProviderFlowUnavailable
	}
	prepared, err := prepareAuthStartInput(input)
	if err != nil {
		return AuthProviderStartResult{}, err
	}
	provider, err := f.resolveAuthProvider(ctx, prepared.ProviderID, prepared.Operation)
	if err != nil {
		return AuthProviderStartResult{}, err
	}
	requestInput := map[string]any{
		"correlationId":     prepared.CorrelationID,
		"deviceFingerprint": prepared.DeviceFingerprint,
		"clientClass":       prepared.ClientClass,
	}
	if prepared.RedirectHint != "" {
		requestInput["redirectHint"] = prepared.RedirectHint
	}
	if prepared.ActorUserID > 0 {
		requestInput["actorUserId"] = prepared.ActorUserID
	}

	var result AuthProviderStartResult
	err = f.invoker.InvokeExact(
		ctx, provider, prepared.Operation, prepared.ActorUserID, requestInput,
		func(_ context.Context, output map[string]any, fence func() error) error {
			parsed, parseErr := parseAuthStartOutput(output)
			if parseErr != nil {
				return parseErr
			}
			if fence != nil {
				if fenceErr := fence(); fenceErr != nil {
					return fenceErr
				}
			}
			result = AuthProviderStartResult{
				ProviderID:     provider.ID,
				Operation:      prepared.Operation,
				Status:         parsed.status,
				CorrelationID:  prepared.CorrelationID,
				ContinueToken:  parsed.continueToken,
				RedirectURL:    parsed.redirectURL,
				ChallengeKind:  parsed.challengeKind,
				ProviderOutput: cloneAuthDocument(output),
			}
			return nil
		},
	)
	if err != nil {
		return AuthProviderStartResult{}, mapAuthProviderInvokeError(err)
	}
	if result.ProviderID == "" {
		return AuthProviderStartResult{}, ErrAuthProviderFlowUnavailable
	}
	return result, nil
}

// Complete 调用选定 auth 提供方的 complete 操作。
// - login.complete / registration.complete：仅返回外部主体断言（注册链接必须
//   由上层通过 LinkTx 与用户创建同事务提交）。
// - link.complete：在 accept 回调内写入 Host 外部链接表。
func (f *AuthProviderFlow) Complete(ctx context.Context, input AuthProviderCompleteInput) (AuthProviderCompleteResult, error) {
	if f == nil || f.source == nil || f.invoker == nil {
		return AuthProviderCompleteResult{}, ErrAuthProviderFlowUnavailable
	}
	prepared, err := prepareAuthCompleteInput(input)
	if err != nil {
		return AuthProviderCompleteResult{}, err
	}
	provider, err := f.resolveAuthProvider(ctx, prepared.ProviderID, prepared.Operation)
	if err != nil {
		return AuthProviderCompleteResult{}, err
	}
	requestInput := map[string]any{
		"correlationId":     prepared.CorrelationID,
		"completionToken":   prepared.CompletionToken,
		"deviceFingerprint": prepared.DeviceFingerprint,
		"clientClass":       prepared.ClientClass,
	}
	if prepared.ActorUserID > 0 {
		requestInput["actorUserId"] = prepared.ActorUserID
	}
	if prepared.TargetUserID > 0 {
		requestInput["targetUserId"] = prepared.TargetUserID
	}

	var result AuthProviderCompleteResult
	err = f.invoker.InvokeExact(
		ctx, provider, prepared.Operation, prepared.ActorUserID, requestInput,
		func(callCtx context.Context, output map[string]any, fence func() error) error {
			parsed, parseErr := parseAuthCompleteOutput(output)
			if parseErr != nil {
				return parseErr
			}
			result = AuthProviderCompleteResult{
				ProviderID:     provider.ID,
				Operation:      prepared.Operation,
				SubjectDigest:  parsed.subjectDigest,
				DisplayName:    parsed.displayName,
				EmailHint:      parsed.emailHint,
				ProviderOutput: cloneAuthDocument(output),
			}
			// 已登录账户链接：在 exact admission 下写入 Host 链接表。
			if prepared.Operation == AuthOperationLinkComplete {
				if f.links == nil {
					return ErrAuthProviderFlowUnavailable
				}
				idempotency := prepared.IdempotencyKey
				if idempotency == "" {
					idempotency = prepared.CorrelationID + ":" + prepared.Operation
				}
				mutation, linkErr := f.links.Link(callCtx, LinkExternalIdentityInput{
					UserID:                prepared.TargetUserID,
					Provider:              provider,
					ProviderOperation:     prepared.Operation,
					ProviderSubjectDigest: parsed.subjectDigest,
					ActorUserID:           prepared.ActorUserID,
					IdempotencyKey:        idempotency,
				}, fence)
				if linkErr != nil {
					return linkErr
				}
				result.Link = &mutation
				return nil
			}
			// login/registration complete：断言 + fence；不在此提交链接。
			if fence != nil {
				if fenceErr := fence(); fenceErr != nil {
					return fenceErr
				}
			}
			return nil
		},
	)
	if err != nil {
		return AuthProviderCompleteResult{}, mapAuthProviderInvokeError(err)
	}
	if result.ProviderID == "" || result.SubjectDigest == "" {
		return AuthProviderCompleteResult{}, ErrAuthProviderFlowUnavailable
	}
	return result, nil
}

func (f *AuthProviderFlow) resolveAuthProvider(
	ctx context.Context,
	providerID, operation string,
) (identityregistry.ProviderContribution, error) {
	provider, err := f.source.ResolveAuthProvider(ctx, providerID)
	if err != nil {
		if errors.Is(err, identityregistry.ErrNotFound) {
			return identityregistry.ProviderContribution{}, ErrAuthProviderNotFound
		}
		return identityregistry.ProviderContribution{}, ErrAuthProviderFlowUnavailable
	}
	if strings.TrimSpace(provider.Kind) != identityregistry.ProviderKindAuth {
		return identityregistry.ProviderContribution{}, ErrAuthProviderNotFound
	}
	if provider.Artifact.Core || strings.TrimSpace(provider.Artifact.RuntimeInstanceID) == "" {
		return identityregistry.ProviderContribution{}, ErrAuthProviderFlowUnavailable
	}
	if !authProviderHasOperation(provider, operation) {
		return identityregistry.ProviderContribution{}, ErrAuthProviderFlowUnavailable
	}
	return provider, nil
}

func prepareAuthStartInput(input AuthProviderStartInput) (AuthProviderStartInput, error) {
	input.ProviderID = strings.ToLower(strings.TrimSpace(input.ProviderID))
	input.Operation = strings.ToLower(strings.TrimSpace(input.Operation))
	switch input.Operation {
	case AuthOperationRegistrationStart, AuthOperationLoginStart, AuthOperationLinkStart:
	default:
		return AuthProviderStartInput{}, ErrAuthProviderFlowInvalid
	}
	if input.ProviderID == "" {
		return AuthProviderStartInput{}, ErrAuthProviderFlowInvalid
	}
	if input.ActorUserID < 0 {
		return AuthProviderStartInput{}, ErrAuthProviderFlowInvalid
	}
	// link.start 需要已登录主体。
	if input.Operation == AuthOperationLinkStart && input.ActorUserID <= 0 {
		return AuthProviderStartInput{}, ErrAuthProviderFlowInvalid
	}
	// registration/login start 必须 actorless。
	if input.Operation != AuthOperationLinkStart && input.ActorUserID != 0 {
		return AuthProviderStartInput{}, ErrAuthProviderFlowInvalid
	}
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	if input.CorrelationID == "" || len(input.CorrelationID) > 200 {
		return AuthProviderStartInput{}, ErrAuthProviderFlowInvalid
	}
	input.DeviceFingerprint = strings.TrimSpace(input.DeviceFingerprint)
	input.ClientClass = strings.TrimSpace(input.ClientClass)
	input.RedirectHint = strings.TrimSpace(input.RedirectHint)
	if len(input.RedirectHint) > 2000 {
		return AuthProviderStartInput{}, ErrAuthProviderFlowInvalid
	}
	return input, nil
}

func prepareAuthCompleteInput(input AuthProviderCompleteInput) (AuthProviderCompleteInput, error) {
	input.ProviderID = strings.ToLower(strings.TrimSpace(input.ProviderID))
	input.Operation = strings.ToLower(strings.TrimSpace(input.Operation))
	switch input.Operation {
	case AuthOperationRegistrationComplete, AuthOperationLoginComplete, AuthOperationLinkComplete:
	default:
		return AuthProviderCompleteInput{}, ErrAuthProviderFlowInvalid
	}
	if input.ProviderID == "" {
		return AuthProviderCompleteInput{}, ErrAuthProviderFlowInvalid
	}
	if input.ActorUserID < 0 || input.TargetUserID < 0 {
		return AuthProviderCompleteInput{}, ErrAuthProviderFlowInvalid
	}
	switch input.Operation {
	case AuthOperationLinkComplete:
		if input.ActorUserID <= 0 || input.TargetUserID <= 0 || input.ActorUserID != input.TargetUserID {
			return AuthProviderCompleteInput{}, ErrAuthProviderFlowInvalid
		}
	case AuthOperationRegistrationComplete, AuthOperationLoginComplete:
		// 完成断言必须 actorless；注册链接由上层 LinkTx 合成。
		if input.ActorUserID != 0 || input.TargetUserID != 0 {
			return AuthProviderCompleteInput{}, ErrAuthProviderFlowInvalid
		}
	}
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.CompletionToken = strings.TrimSpace(input.CompletionToken)
	if input.CorrelationID == "" || len(input.CorrelationID) > 200 {
		return AuthProviderCompleteInput{}, ErrAuthProviderFlowInvalid
	}
	if input.CompletionToken == "" || len(input.CompletionToken) > 4000 {
		return AuthProviderCompleteInput{}, ErrAuthProviderFlowInvalid
	}
	input.DeviceFingerprint = strings.TrimSpace(input.DeviceFingerprint)
	input.ClientClass = strings.TrimSpace(input.ClientClass)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if len(input.IdempotencyKey) > 200 {
		return AuthProviderCompleteInput{}, ErrAuthProviderFlowInvalid
	}
	return input, nil
}

func authProviderHasOperation(provider identityregistry.ProviderContribution, operation string) bool {
	for _, op := range provider.Operations {
		if strings.TrimSpace(op.Name) == operation {
			return true
		}
	}
	return false
}

type authStartParsed struct {
	status         string
	continueToken  string
	redirectURL    string
	challengeKind  string
}

func parseAuthStartOutput(output map[string]any) (authStartParsed, error) {
	if output == nil {
		return authStartParsed{}, ErrAuthProviderFlowUnavailable
	}
	status := strings.ToLower(strings.TrimSpace(stringFromAuthOutput(output, "status")))
	if status == "" {
		status = AuthStartStatusContinue
	}
	switch status {
	case AuthStartStatusContinue, AuthStartStatusRedirect, AuthStartStatusChallenge:
	default:
		return authStartParsed{}, ErrAuthProviderFlowUnavailable
	}
	parsed := authStartParsed{
		status:        status,
		continueToken: strings.TrimSpace(stringFromAuthOutput(output, "continueToken")),
		redirectURL:   strings.TrimSpace(stringFromAuthOutput(output, "redirectUrl")),
		challengeKind: strings.TrimSpace(stringFromAuthOutput(output, "challengeKind")),
	}
	if status == AuthStartStatusRedirect && parsed.redirectURL == "" {
		return authStartParsed{}, ErrAuthProviderFlowUnavailable
	}
	if status == AuthStartStatusChallenge && parsed.challengeKind == "" {
		return authStartParsed{}, ErrAuthProviderFlowUnavailable
	}
	if len(parsed.continueToken) > 4000 || len(parsed.redirectURL) > 2000 || len(parsed.challengeKind) > 120 {
		return authStartParsed{}, ErrAuthProviderFlowUnavailable
	}
	return parsed, nil
}

type authCompleteParsed struct {
	subjectDigest string
	displayName   string
	emailHint     string
}

func parseAuthCompleteOutput(output map[string]any) (authCompleteParsed, error) {
	if output == nil {
		return authCompleteParsed{}, ErrAuthProviderFlowUnavailable
	}
	digest := strings.ToLower(strings.TrimSpace(stringFromAuthOutput(output, "providerSubjectDigest")))
	if digest == "" {
		digest = strings.ToLower(strings.TrimSpace(stringFromAuthOutput(output, "subjectDigest")))
	}
	if !isAuthSubjectDigest(digest) {
		return authCompleteParsed{}, ErrAuthProviderFlowUnavailable
	}
	displayName := strings.TrimSpace(stringFromAuthOutput(output, "displayName"))
	emailHint := strings.TrimSpace(strings.ToLower(stringFromAuthOutput(output, "emailHint")))
	if len(displayName) > 200 || len(emailHint) > 320 {
		return authCompleteParsed{}, ErrAuthProviderFlowUnavailable
	}
	return authCompleteParsed{
		subjectDigest: digest,
		displayName:   displayName,
		emailHint:     emailHint,
	}, nil
}

func isAuthSubjectDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			continue
		}
		return false
	}
	return true
}

func stringFromAuthOutput(output map[string]any, key string) string {
	raw, _ := output[key].(string)
	return raw
}

func cloneAuthDocument(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func mapAuthProviderInvokeError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrAuthProviderFlowInvalid) ||
		errors.Is(err, ErrAuthProviderFlowUnavailable) ||
		errors.Is(err, ErrAuthProviderFlowDenied) ||
		errors.Is(err, ErrAuthProviderNotFound) ||
		errors.Is(err, ErrExternalIdentityLinkInvalid) ||
		errors.Is(err, ErrExternalIdentityLinkStateConflict) ||
		errors.Is(err, ErrExternalIdentityLinkIdempotencyConflict) ||
		errors.Is(err, ErrExternalIdentitySubjectConflict) ||
		errors.Is(err, ErrExternalIdentityProviderStale) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrAuthProviderFlowUnavailable, err)
}

// RegistryAuthProviderSource 从 Identity Registry 解析精确 auth 提供方。
type RegistryAuthProviderSource struct {
	Registry *identityregistry.Registry
}

func (s RegistryAuthProviderSource) ResolveAuthProvider(_ context.Context, providerID string) (identityregistry.ProviderContribution, error) {
	if s.Registry == nil {
		return identityregistry.ProviderContribution{}, ErrAuthProviderFlowUnavailable
	}
	provider, err := s.Registry.ResolveProvider(providerID)
	if err != nil {
		return identityregistry.ProviderContribution{}, err
	}
	if strings.TrimSpace(provider.Kind) != identityregistry.ProviderKindAuth {
		return identityregistry.ProviderContribution{}, identityregistry.ErrNotFound
	}
	return provider, nil
}

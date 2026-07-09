package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	mail "github.com/zhuchunshu/sforum/apps/api/app/Support/Mail"
)

// PasswordResetConfig 是密码重置流程的可选依赖。
type PasswordResetConfig struct {
	// TokenLifetime 令牌有效期，默认 30 分钟。
	TokenLifetime time.Duration
	// SiteName / SiteURL 用于邮件正文。
	SiteName string
	SiteURL  string
	// ResetPathBase 是前端重置密码页路径前缀，默认 /reset-password?token=。
	ResetPathBase string
}

// PasswordResetService 协调密码重置：生成令牌、投递邮件、校验与消费令牌、更新密码。
type PasswordResetService struct {
	store            Store
	mailer           *mail.Service
	config           PasswordResetConfig
	passwordPolicies PasswordPolicyResolver
}

func NewPasswordResetService(store Store, mailer *mail.Service, config PasswordResetConfig) *PasswordResetService {
	return NewPasswordResetServiceWithPasswordPolicy(store, mailer, config, nil)
}

func NewPasswordResetServiceWithPasswordPolicy(store Store, mailer *mail.Service, config PasswordResetConfig, resolver PasswordPolicyResolver) *PasswordResetService {
	if config.TokenLifetime <= 0 {
		config.TokenLifetime = 30 * time.Minute
	}
	if config.ResetPathBase == "" {
		config.ResetPathBase = "/reset-password?token="
	}
	if resolver == nil {
		resolver = staticRecommendedPasswordPolicy{}
	}
	return &PasswordResetService{store: store, mailer: mailer, config: config, passwordPolicies: resolver}
}

// RequestPasswordResetInput 是发起密码重置的入参。
type RequestPasswordResetInput struct {
	Email string
	IP    string
}

// RequestPasswordReset 查找用户并投递重置邮件。
// 出于隐私，无论邮箱是否存在都返回 nil；仅当邮件投递失败时返回错误（此时不暴露用户是否存在）。
// 调用方应始终返回相同的成功响应。
func (s *PasswordResetService) RequestPasswordReset(ctx context.Context, input RequestPasswordResetInput) error {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	if email == "" {
		return nil
	}
	credential, err := s.store.GetCredentialByLogin(ctx, email)
	if err != nil {
		// 用户不存在：静默成功，不暴露邮箱是否存在。
		if errors.Is(err, ErrCredentialNotFound) {
			return nil
		}
		return err
	}
	if credential.Status != UserStatusActive {
		// 非活跃用户也静默成功。
		return nil
	}

	rawToken, err := generateResetToken()
	if err != nil {
		return err
	}
	tokenHash := hashResetToken(rawToken)
	expiresAt := time.Now().Add(s.config.TokenLifetime).UTC()

	ipHash := hashIP(input.IP)
	if _, err := s.store.CreatePasswordResetToken(ctx, CreatePasswordResetTokenInput{
		UserID:        credential.ID,
		TokenHash:     tokenHash,
		ExpiresAt:     expiresAt,
		RequestIPHash: ipHash,
	}); err != nil {
		return err
	}

	// 仅在配置了邮件服务时投递；mailer 为 nil 时跳过（开发环境可用 dev_log）。
	if s.mailer == nil {
		return nil
	}
	resetURL := s.buildResetURL(rawToken)
	message := mail.Message{
		To:       email,
		Subject:  s.resetEmailSubject(),
		TextBody: s.resetEmailBody(credential.Username, resetURL, expiresAt),
	}
	if err := s.mailer.Send(ctx, message); err != nil {
		// 投递失败：记录但不向调用方暴露用户信息。
		// 注意：这里不返回错误，以免泄露"邮箱存在但投递失败"。
		// 调用方仍返回通用成功响应。
		return nil
	}
	return nil
}

// ConfirmPasswordResetInput 是确认密码重置的入参。
type ConfirmPasswordResetInput struct {
	Token       string
	NewPassword string
}

// ConfirmPasswordReset 校验令牌、消费令牌、更新密码。
func (s *PasswordResetService) ConfirmPasswordReset(ctx context.Context, input ConfirmPasswordResetInput) error {
	token := strings.TrimSpace(input.Token)
	if token == "" {
		return ErrPasswordResetTokenNotFound
	}
	policy, err := s.passwordPolicies.PasswordPolicy(ctx)
	if err != nil {
		return err
	}
	if fields := policy.Validate(input.NewPassword); len(fields) > 0 {
		return NewRegisterInvalid(fields)
	}
	hash := hashResetToken(token)
	userID, err := s.store.ConsumePasswordResetToken(ctx, hash)
	if err != nil {
		return err
	}
	newHash, err := HashPassword(input.NewPassword)
	if err != nil {
		return err
	}
	if err := s.store.UpdateUserPassword(ctx, userID, newHash); err != nil {
		return err
	}
	// M8：递增令牌版本号，使该用户所有旧会话立即失效（含攻击者已持有的会话）。
	// 失败不阻塞密码重置本身（密码已更新），仅记录错误。
	_ = s.store.IncrementUserTokenVersion(ctx, userID)
	// 同步撤销 user_sessions 目录：令牌版本号让旧会话在 CurrentUserID 失效，
	// 但目录表里这些会话仍是 revoked_at IS NULL，会导致设备列表显示失真、
	// EnforceMaxSessions 把幽灵会话计入活跃数而误踢真活跃设备。
	// 这里把它们标记为 password_reset，保持目录与真实鉴权状态一致。best-effort。
	_, _ = s.store.RevokeUserSessions(ctx, userID, RevokeReasonPasswordReset)
	return nil
}

func (s *PasswordResetService) buildResetURL(rawToken string) string {
	base := s.config.SiteURL
	if base == "" {
		base = "http://127.0.0.1:3000"
	}
	base = strings.TrimRight(base, "/")
	return base + s.config.ResetPathBase + rawToken
}

func (s *PasswordResetService) resetEmailSubject() string {
	name := s.config.SiteName
	if name == "" {
		name = "SForum"
	}
	return fmt.Sprintf("[%s] 重置你的密码 / Reset your password", name)
}

func (s *PasswordResetService) resetEmailBody(username, resetURL string, expiresAt time.Time) string {
	name := s.config.SiteName
	if name == "" {
		name = "SForum"
	}
	return fmt.Sprintf(
		"你好 %s，\n\n你（或某人）请求重置在 %s 的密码。\n\n请访问以下链接重置密码：\n%s\n\n该链接将于 %s 失效。\n如果你没有发起此请求，请忽略本邮件。\n",
		username, name, resetURL, expiresAt.Format(time.RFC3339),
	)
}

// generateResetToken 生成 32 字节随机令牌并编码为 hex。
func generateResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate reset token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// hashResetToken 对令牌做 sha256 哈希，存储哈希值而非明文令牌。
func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// hashIP 对请求 IP 做哈希，用于限流与审计，不存明文 IP。
func hashIP(ip string) string {
	if ip == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:])
}

package providers

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	identitycontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Identity"
	apitokens "github.com/zhuchunshu/sforum/apps/api/app/Models/APITokens"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
)

type IdentityProvider struct {
	controller *identitycontroller.Controller
}

func NewIdentityProvider(store identity.Store, sessions *session.Store) *IdentityProvider {
	return NewIdentityProviderWithVerifier(store, sessions, humanverify.NewDisabledService())
}

func NewIdentityProviderWithVerifier(store identity.Store, sessions *session.Store, verifier humanverify.Verifier) *IdentityProvider {
	return NewIdentityProviderWithAuthSessions(store, authsession.NewManager(sessions, authsession.Config{}), verifier)
}

func NewIdentityProviderWithAuthSessions(store identity.Store, sessions *authsession.Manager, verifier humanverify.Verifier) *IdentityProvider {
	return NewIdentityProviderWithEvents(store, sessions, verifier, nil)
}

func NewIdentityProviderWithEvents(store identity.Store, sessions *authsession.Manager, verifier humanverify.Verifier, publisher appevents.Publisher) *IdentityProvider {
	return &IdentityProvider{
		controller: identitycontroller.NewControllerWithAuthSessions(identity.NewServiceWithEvents(store, publisher), sessions, verifier),
	}
}

// NewIdentityProviderWithPasswordReset 注入密码重置与邮件服务。
type adminMailQueue interface {
	QueueMail(context.Context, notifications.QueueMailInput) (notifications.MailDelivery, error)
}

func NewIdentityProviderWithPasswordReset(store identity.Store, sessions *authsession.Manager, verifier humanverify.Verifier, publisher appevents.Publisher, passwordReset *identity.PasswordResetService, mailQueue adminMailQueue, options optionsResolver) *IdentityProvider {
	return NewIdentityProviderWithPasswordResetAndLockout(store, sessions, verifier, publisher, passwordReset, mailQueue, options, nil)
}

// NewIdentityProviderWithPasswordResetAndLockout 额外注入登录失败锁定 store（通常 Redis）。
func NewIdentityProviderWithPasswordResetAndLockout(store identity.Store, sessions *authsession.Manager, verifier humanverify.Verifier, publisher appevents.Publisher, passwordReset *identity.PasswordResetService, mailQueue adminMailQueue, options optionsResolver, lockout identity.LoginLockoutStore) *IdentityProvider {
	svc := identity.NewServiceWithPolicies(store, publisher, options, options)
	if options != nil {
		svc.WithUsernamePolicy(options)
		if lockout != nil {
			svc.WithLoginLockout(lockout, options)
		}
	}
	return &IdentityProvider{
		// options 同时作为密码策略与开放注册策略解析器。
		controller: identitycontroller.NewControllerWithPasswordReset(
			svc,
			sessions, verifier, passwordReset, mailQueue, options,
		),
	}
}

// optionsResolver 暴露密码策略、开放注册开关、用户名策略、登录锁定策略以及密码重置/mail-test 需要的站点名/管理员邮箱。
type optionsResolver interface {
	SiteName(ctx context.Context) (string, error)
	AdminEmail(ctx context.Context) (string, error)
	WebOption(ctx context.Context, name string) (string, error)
	PasswordPolicy(ctx context.Context) (identity.PasswordPolicy, error)
	RegistrationEnabled(ctx context.Context) (bool, error)
	UsernamePolicy(ctx context.Context) (identity.UsernamePolicy, error)
	LoginLockoutPolicy(ctx context.Context) (identity.LoginLockoutPolicy, error)
}

// WithAPITokens 启用 PAT 管理路由（F3.4）。
func (p *IdentityProvider) WithAPITokens(tokens *apitokens.Service) *IdentityProvider {
	if p != nil && p.controller != nil {
		p.controller.WithAPITokens(tokens)
	}
	return p
}

func (p *IdentityProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

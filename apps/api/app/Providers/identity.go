package providers

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	identitycontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Identity"
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
	return &IdentityProvider{
		controller: identitycontroller.NewControllerWithPasswordReset(identity.NewServiceWithEventsAndPasswordPolicy(store, publisher, options), sessions, verifier, passwordReset, mailQueue, options),
	}
}

// optionsResolver 仅暴露密码重置/mail-test 需要的站点名/URL。
type optionsResolver interface {
	SiteName(ctx context.Context) (string, error)
	WebOption(ctx context.Context, name string) (string, error)
	PasswordPolicy(ctx context.Context) (identity.PasswordPolicy, error)
}

func (p *IdentityProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

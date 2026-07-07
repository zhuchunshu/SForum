package providers

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	identitycontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Identity"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	mail "github.com/zhuchunshu/sforum/apps/api/app/Support/Mail"
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
func NewIdentityProviderWithPasswordReset(store identity.Store, sessions *authsession.Manager, verifier humanverify.Verifier, publisher appevents.Publisher, passwordReset *identity.PasswordResetService, mailService *mail.Service, options optionsResolver) *IdentityProvider {
	return &IdentityProvider{
		controller: identitycontroller.NewControllerWithPasswordReset(identity.NewServiceWithEvents(store, publisher), sessions, verifier, passwordReset, mailService, options),
	}
}

// optionsResolver 仅暴露密码重置/mail-test 需要的站点名/URL。
type optionsResolver interface {
	SiteName(ctx context.Context) (string, error)
	WebOption(ctx context.Context, name string) (string, error)
}

func (p *IdentityProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

package providers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	identitycontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Identity"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
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

func (p *IdentityProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

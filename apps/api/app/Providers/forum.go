package providers

import (
	"github.com/gofiber/fiber/v3"

	forumcontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Forum"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

type ForumProvider struct {
	controller *forumcontroller.Controller
}

func NewForumProvider(store forum.Store, users identity.ActorStore, sessions *authsession.Manager) *ForumProvider {
	return NewForumProviderWithEvents(store, users, sessions, nil)
}

func NewForumProviderWithEvents(store forum.Store, users identity.ActorStore, sessions *authsession.Manager, publisher appevents.Publisher) *ForumProvider {
	return &ForumProvider{
		controller: forumcontroller.NewController(forum.NewServiceWithEvents(store, publisher), users, sessions),
	}
}

func (p *ForumProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

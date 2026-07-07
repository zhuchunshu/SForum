package providers

import (
	"github.com/gofiber/fiber/v3"

	moderationcontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Moderation"
	moderation "github.com/zhuchunshu/sforum/apps/api/app/Models/Moderation"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
)

type ModerationProvider struct {
	controller *moderationcontroller.Controller
}

func NewModerationProvider(store moderation.Store, forumStore forum.Store, users identity.ActorStore, sessions *authsession.Manager) *ModerationProvider {
	validator := moderation.NewForumTargetValidator(forumStore)
	return &ModerationProvider{
		controller: moderationcontroller.NewController(moderation.NewService(store, validator), users, sessions),
	}
}

func (p *ModerationProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

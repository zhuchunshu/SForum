package identity

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

type Provider struct {
	handler *Handler
}

func NewProvider(store Store, sessions *session.Store) *Provider {
	return &Provider{
		handler: NewHandler(NewService(store), sessions),
	}
}

func (p *Provider) RegisterRoutes(api fiber.Router) {
	p.handler.RegisterRoutes(api)
}

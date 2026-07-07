package profilecontroller

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	profile "github.com/zhuchunshu/sforum/apps/api/app/Models/Profile"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
)

type Controller struct {
	service  *profile.Service
	users    identity.ActorStore
	sessions *authsession.Manager
}

func NewController(service *profile.Service, users identity.ActorStore, sessions *authsession.Manager) *Controller {
	return &Controller{service: service, users: users, sessions: sessions}
}

// updateProfileRequest 所有字段可选（指针）；nil 表示不改。
type updateProfileRequest struct {
	Bio                *string `json:"bio"`
	Signature          *string `json:"signature"`
	Location           *string `json:"location"`
	WebsiteURL         *string `json:"websiteUrl"`
	AvatarAttachmentID *int64  `json:"avatarAttachmentId"`
}

func (h *Controller) publicProfile(c fiber.Ctx) error {
	username := c.Params("username")
	data, err := h.service.GetPublicProfile(c.Context(), username)
	if err != nil {
		return mapProfileError(err)
	}
	return apphttp.OK(c, data)
}

func (h *Controller) myProfile(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	data, err := h.service.GetMyProfile(c.Context(), actor.ID)
	if err != nil {
		return mapProfileError(err)
	}
	return apphttp.OK(c, data)
}

func (h *Controller) updateMyProfile(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req updateProfileRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, profile.CodeProfileInvalid)
	}
	updated, err := h.service.UpdateMyProfile(c.Context(), actor, profile.UpdateProfileInput{
		Bio:                req.Bio,
		Signature:          req.Signature,
		Location:           req.Location,
		WebsiteURL:         req.WebsiteURL,
		AvatarAttachmentID: req.AvatarAttachmentID,
	})
	if err != nil {
		return mapProfileError(err)
	}
	return apphttp.OK(c, updated)
}

func (h *Controller) actor(c fiber.Ctx) (identity.Actor, error) {
	userID, ok, err := h.sessions.CurrentUserID(c)
	if err != nil {
		return identity.Actor{}, err
	}
	if !ok {
		return identity.Actor{}, fiber.NewError(fiber.StatusUnauthorized, "auth.required")
	}
	return h.users.LoadActor(c.Context(), userID)
}

func mapProfileError(err error) error {
	switch {
	case errors.Is(err, identity.ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	case errors.Is(err, profile.ErrProfileNotFound):
		return fiber.NewError(fiber.StatusNotFound, profile.CodeProfileNotFound)
	case errors.Is(err, profile.ErrProfileInvalid):
		return fiber.NewError(fiber.StatusUnprocessableEntity, profile.CodeProfileInvalid)
	default:
		return err
	}
}

// 保留 strconv 引用，供未来分页参数扩展使用。
var _ = strconv.Atoi

package profilecontroller

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	profile "github.com/zhuchunshu/sforum/apps/api/app/Models/Profile"
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

func (h *Controller) uploadAvatar(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, profile.CodeProfileInvalid)
	}
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	updated, err := h.service.UploadAvatar(c.Context(), actor, attachments.UploadInput{
		OriginalName: fileHeader.Filename,
		ContentType:  fileHeader.Header.Get("Content-Type"),
		SizeBytes:    fileHeader.Size,
		File:         file,
	})
	if err != nil {
		return mapProfileError(err)
	}
	return apphttp.OK(c, updated)
}

func (h *Controller) deleteAvatar(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	updated, err := h.service.DeleteAvatar(c.Context(), actor)
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
	case errors.Is(err, profile.ErrAvatarUploadDisabled):
		return fiber.NewError(fiber.StatusUnprocessableEntity, profile.CodeAvatarUploadDisabled)
	case errors.Is(err, attachments.ErrUploadDisabled):
		return fiber.NewError(fiber.StatusUnprocessableEntity, profile.CodeAvatarUploadDisabled)
	case errors.Is(err, attachments.ErrInvalidAttachment):
		return fiber.NewError(fiber.StatusUnprocessableEntity, profile.CodeProfileInvalid)
	case errors.Is(err, attachments.ErrStorageUnavailable):
		return fiber.NewError(fiber.StatusServiceUnavailable, attachments.CodeStorageUnavailable)
	default:
		return err
	}
}

// 保留 strconv 引用，供未来分页参数扩展使用。
var _ = strconv.Atoi

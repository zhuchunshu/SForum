package attachmentscontroller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

type Controller struct {
	service  *attachments.Service
	users    identity.ActorStore
	sessions *authsession.Manager
}

func NewController(service *attachments.Service, users identity.ActorStore, sessions *authsession.Manager) *Controller {
	return &Controller{service: service, users: users, sessions: sessions}
}

type updateAttachmentRequest struct {
	Status string `json:"status"`
}

type cleanupRequest struct {
	Limit int `json:"limit"`
}

type attachmentDTO struct {
	ID             int64                     `json:"id"`
	PublicID       string                    `json:"publicId"`
	Name           string                    `json:"name"`
	ContentType    string                    `json:"contentType"`
	Size           int64                     `json:"size"`
	URL            string                    `json:"url"`
	Status         string                    `json:"status"`
	Provider       string                    `json:"provider"`
	Owner          *attachments.OwnerSummary `json:"owner,omitempty"`
	ReferenceCount int                       `json:"referenceCount"`
	CreatedAt      time.Time                 `json:"createdAt"`
}

type adminAttachmentDTO struct {
	attachmentDTO
	ObjectKey   string     `json:"objectKey"`
	Extension   string     `json:"extension"`
	SHA256      string     `json:"sha256"`
	ImageWidth  *int       `json:"imageWidth,omitempty"`
	ImageHeight *int       `json:"imageHeight,omitempty"`
	Visibility  string     `json:"visibility"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

type attachmentListDTO struct {
	Items   []adminAttachmentDTO `json:"items"`
	Total   int64                `json:"total"`
	Page    int                  `json:"page"`
	PerPage int                  `json:"perPage"`
}

type attachmentDetailDTO struct {
	adminAttachmentDTO
	References []attachments.AttachmentReference `json:"references"`
}

func (h *Controller) upload(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, attachments.CodeInvalidAttachment)
	}
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	item, err := h.service.Upload(c.Context(), actor, attachments.UploadInput{
		OriginalName: fileHeader.Filename,
		ContentType:  fileHeader.Header.Get("Content-Type"),
		SizeBytes:    fileHeader.Size,
		File:         file,
	})
	if err != nil {
		return mapAttachmentError(err)
	}
	return apphttp.Created(c, toAttachmentDTO(item))
}

func (h *Controller) get(c fiber.Ctx) error {
	actor, err := h.optionalActor(c)
	if err != nil {
		return err
	}
	item, err := h.service.Get(c.Context(), actor, c.Params("publicId"))
	if err != nil {
		return mapAttachmentError(err)
	}
	return apphttp.OK(c, toAttachmentDTO(item))
}

func (h *Controller) content(c fiber.Ctx) error {
	actor, err := h.optionalActor(c)
	if err != nil {
		return err
	}
	item, reader, err := h.service.OpenContent(c.Context(), actor, c.Params("publicId"))
	if err != nil {
		return mapAttachmentError(err)
	}
	c.Set(fiber.HeaderContentType, item.ContentType)
	// 安全头：阻止 MIME 嗅探；主动内容类型（HTML/SVG/JS 等）强制下载而非内联渲染，
	// 避免公开附件形成同源存储型 XSS（即便入库时 denylist 被绕过，此处兜底）。
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set(fiber.HeaderContentDisposition, attachmentContentDisposition(item.ContentType, item.OriginalName))
	return c.SendStream(reader)
}

// attachmentContentDisposition 按 MIME 决定内联或强制下载，并安全拼装文件名。
// 主动内容类型强制 attachment，杜绝同源 XSS；文件名中的双引号被剔除以防注入。
func attachmentContentDisposition(contentType, originalName string) string {
	disposition := "inline"
	if options.IsAttachmentActiveContentType(contentType) {
		disposition = "attachment"
	}
	// 剥离 CR/LF 与双引号，防止 Content-Disposition 头注入。
	safeName := strings.NewReplacer("\r", "", "\n", "", `"`, "").Replace(originalName)
	return disposition + `; filename="` + safeName + `"`
}

func (h *Controller) settings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	settings, err := h.service.Settings(c.Context(), actor)
	if err != nil {
		return mapAttachmentError(err)
	}
	return apphttp.OK(c, settings)
}

func (h *Controller) updateSettings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req attachments.AttachmentSettings
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, attachments.CodeInvalidAttachment)
	}
	settings, err := h.service.UpdateSettings(c.Context(), actor, req)
	if err != nil {
		return mapAttachmentError(err)
	}
	return apphttp.OK(c, settings)
}

func (h *Controller) testSettings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	result, err := h.service.Probe(c.Context(), actor)
	if err != nil {
		return mapAttachmentError(err)
	}
	return apphttp.OK(c, result)
}

func (h *Controller) listAdmin(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	list, err := h.service.List(c.Context(), actor, attachments.AttachmentListInput{
		Page:            queryInt(c, "page"),
		PerPage:         queryInt(c, "perPage"),
		Query:           c.Query("query"),
		Provider:        c.Query("provider"),
		Status:          c.Query("status"),
		ContentType:     c.Query("contentType"),
		OwnerUserID:     int64(queryInt(c, "ownerUserId")),
		ReferenceStatus: c.Query("referenceStatus"),
		CreatedFrom:     queryTime(c, "createdFrom"),
		CreatedTo:       queryTime(c, "createdTo"),
	})
	if err != nil {
		return mapAttachmentError(err)
	}
	return apphttp.OK(c, toAttachmentListDTO(list))
}

func (h *Controller) detailAdmin(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	detail, err := h.service.Detail(c.Context(), actor, int64(queryParamInt(c, "id")))
	if err != nil {
		return mapAttachmentError(err)
	}
	return apphttp.OK(c, toAttachmentDetailDTO(detail))
}

func (h *Controller) updateAdmin(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req updateAttachmentRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, attachments.CodeInvalidAttachment)
	}
	item, err := h.service.UpdateStatus(c.Context(), actor, int64(queryParamInt(c, "id")), req.Status)
	if err != nil {
		return mapAttachmentError(err)
	}
	return apphttp.OK(c, toAdminAttachmentDTO(item))
}

func (h *Controller) deleteAdmin(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	item, err := h.service.Delete(c.Context(), actor, int64(queryParamInt(c, "id")))
	if err != nil {
		return mapAttachmentError(err)
	}
	return apphttp.OK(c, toAdminAttachmentDTO(item))
}

func (h *Controller) cleanupAdmin(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req cleanupRequest
	_ = c.Bind().Body(&req)
	result, err := h.service.Cleanup(c.Context(), actor, req.Limit)
	if err != nil {
		return mapAttachmentError(err)
	}
	return apphttp.OK(c, result)
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

func (h *Controller) optionalActor(c fiber.Ctx) (identity.Actor, error) {
	userID, ok, err := h.sessions.CurrentUserID(c)
	if err != nil {
		return identity.Actor{}, err
	}
	if !ok {
		return identity.Actor{}, nil
	}
	return h.users.LoadActor(c.Context(), userID)
}

func mapAttachmentError(err error) error {
	var rejected *appevents.RejectedError
	switch {
	case errors.Is(err, identity.ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	case errors.Is(err, attachments.ErrGuestLoginRequired):
		return fiber.NewError(fiber.StatusUnauthorized, attachments.CodeGuestLoginRequired)
	// 插件 attachment.before_upload 等同步拒绝：422 + 稳定 reason。
	case errors.As(err, &rejected):
		return fiber.NewError(fiber.StatusUnprocessableEntity, rejected.Reason)
	case errors.Is(err, attachments.ErrUploadDisabled):
		return fiber.NewError(fiber.StatusUnprocessableEntity, attachments.CodeUploadDisabled)
	case errors.Is(err, attachments.ErrInvalidAttachment):
		return fiber.NewError(fiber.StatusUnprocessableEntity, attachments.CodeInvalidAttachment)
	case errors.Is(err, attachments.ErrReferenced):
		return fiber.NewError(fiber.StatusConflict, attachments.CodeReferenced)
	case errors.Is(err, attachments.ErrAttachmentNotFound):
		return fiber.NewError(fiber.StatusNotFound, attachments.CodeInvalidAttachment)
	case errors.Is(err, attachments.ErrStorageUnavailable):
		return fiber.NewError(fiber.StatusServiceUnavailable, attachments.CodeStorageUnavailable)
	default:
		return err
	}
}

func toAttachmentDTO(item attachments.Attachment) attachmentDTO {
	return attachmentDTO{
		ID:             item.ID,
		PublicID:       item.PublicID,
		Name:           item.OriginalName,
		ContentType:    item.ContentType,
		Size:           item.SizeBytes,
		URL:            item.URL,
		Status:         item.Status,
		Provider:       item.Provider,
		Owner:          item.Owner,
		ReferenceCount: item.ReferenceCount,
		CreatedAt:      item.CreatedAt,
	}
}

func toAdminAttachmentDTO(item attachments.Attachment) adminAttachmentDTO {
	return adminAttachmentDTO{
		attachmentDTO: toAttachmentDTO(item),
		ObjectKey:     item.ObjectKey,
		Extension:     item.Extension,
		SHA256:        item.SHA256,
		ImageWidth:    item.ImageWidth,
		ImageHeight:   item.ImageHeight,
		Visibility:    item.Visibility,
		DeletedAt:     item.DeletedAt,
	}
}

func toAttachmentListDTO(list attachments.AttachmentList) attachmentListDTO {
	items := make([]adminAttachmentDTO, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, toAdminAttachmentDTO(item))
	}
	return attachmentListDTO{Items: items, Total: list.Total, Page: list.Page, PerPage: list.PerPage}
}

func toAttachmentDetailDTO(detail attachments.AttachmentDetail) attachmentDetailDTO {
	return attachmentDetailDTO{
		adminAttachmentDTO: toAdminAttachmentDTO(detail.Attachment),
		References:         detail.References,
	}
}

func queryInt(c fiber.Ctx, key string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(c.Query(key)))
	return value
}

func queryParamInt(c fiber.Ctx, key string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(c.Params(key)))
	return value
}

func queryTime(c fiber.Ctx, key string) time.Time {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed
	}
	return time.Time{}
}

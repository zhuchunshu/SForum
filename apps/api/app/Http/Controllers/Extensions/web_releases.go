package extensionscontroller

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func (h *Controller) listWebReleases(c fiber.Ctx) error {
	if h.webReleases == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeFrontendRuntimeUnavailable)
	}
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	page, err := h.webReleases.List(c.Context(), actor, extensions.WebReleaseListInput{
		Status:  extensions.WebReleaseStatus(c.Query("status")),
		Page:    queryInt(c, "page", 1),
		PerPage: queryInt(c, "perPage", 20),
	})
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, page)
}

func (h *Controller) webReleaseDetail(c fiber.Ctx) error {
	if h.webReleases == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeFrontendRuntimeUnavailable)
	}
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	releaseID, err := webReleaseID(c)
	if err != nil {
		return err
	}
	detail, err := h.webReleases.Detail(c.Context(), actor, releaseID)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.OK(c, detail)
}

// rebuildWebRelease 手动排队一次 Web Release（主题/可信插件 admin 前端变更后）。
func (h *Controller) rebuildWebRelease(c fiber.Ctx) error {
	if h.webReleases == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeFrontendRuntimeUnavailable)
	}
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	operation, err := h.webReleases.Rebuild(c.Context(), actor)
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.JSON(c, fiber.StatusAccepted, apphttp.MessageOK, operation)
}

func (h *Controller) retryWebRelease(c fiber.Ctx) error {
	return h.runWebReleaseCommand(c, false)
}

func (h *Controller) rollbackWebRelease(c fiber.Ctx) error {
	return h.runWebReleaseCommand(c, true)
}

func (h *Controller) runWebReleaseCommand(c fiber.Ctx, rollback bool) error {
	if h.webReleases == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeFrontendRuntimeUnavailable)
	}
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	releaseID, err := webReleaseID(c)
	if err != nil {
		return err
	}
	var operation extensions.WebReleaseOperation
	if rollback {
		operation, err = h.webReleases.Rollback(c.Context(), actor, releaseID)
	} else {
		operation, err = h.webReleases.Retry(c.Context(), actor, releaseID)
	}
	if err != nil {
		return mapExtensionError(err)
	}
	return apphttp.JSON(c, fiber.StatusAccepted, apphttp.MessageOK, operation)
}

func webReleaseID(c fiber.Ctx) (int64, error) {
	id, err := strconv.ParseInt(c.Params("releaseID"), 10, 64)
	if err != nil || id <= 0 {
		return 0, fiber.NewError(fiber.StatusNotFound, extensions.CodeWebReleaseNotFound)
	}
	return id, nil
}

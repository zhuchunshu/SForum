package extensionscontroller

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

const (
	routeInspectorInvalidReason     = "extensions.route_inspector_invalid"
	routeInspectorUnavailableReason = "extensions.route_inspector_unavailable"
)

func (h *Controller) inspectRoute(c fiber.Ctx) error {
	if _, err := h.routeProviderViewer(c); err != nil {
		return err
	}
	if h.routeInspector == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, routeInspectorUnavailableReason)
	}
	method := strings.TrimSpace(c.Query("method"))
	requestPath := strings.TrimSpace(c.Query("path"))
	if method == "" || requestPath == "" {
		return fiber.NewError(fiber.StatusUnprocessableEntity, routeInspectorInvalidReason)
	}
	snapshot, err := h.routeInspector.Inspect(c.Context(), method, requestPath)
	if err != nil {
		return mapRouteInspectorError(err)
	}
	return apphttp.OK(c, snapshot)
}

func mapRouteInspectorError(err error) error {
	switch {
	case errors.Is(err, routes.ErrInspectorInvalid):
		return fiber.NewError(fiber.StatusUnprocessableEntity, routeInspectorInvalidReason)
	case errors.Is(err, routes.ErrRevisionConflict),
		errors.Is(err, routes.ErrProviderSelectionRevisionConflict):
		return fiber.NewError(fiber.StatusConflict, routeProviderConflictReason)
	case errors.Is(err, routes.ErrProviderSelectionInvalid):
		return fiber.NewError(fiber.StatusUnprocessableEntity, routeInspectorInvalidReason)
	default:
		return fiber.NewError(fiber.StatusServiceUnavailable, routeInspectorUnavailableReason)
	}
}

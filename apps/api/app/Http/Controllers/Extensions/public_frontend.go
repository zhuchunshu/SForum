package extensionscontroller

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func (h *Controller) publicFrontendComponent(c fiber.Ctx) error {
	runtime, ok := h.frontend.(PublicFrontendRuntimeService)
	if h.frontend == nil || !ok {
		return publicFrontendNotFound()
	}
	descriptor, err := runtime.PublicComponent(c.Context(), c.Params("extensionId"), c.Params("componentId"))
	if err != nil {
		return mapPublicFrontendError(err)
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set("X-Content-Type-Options", "nosniff")
	return apphttp.OK(c, descriptor)
}

func (h *Controller) publicFrontendAsset(c fiber.Ctx) error {
	runtime, ok := h.frontend.(PublicFrontendRuntimeService)
	if h.frontend == nil || !ok {
		return publicFrontendNotFound()
	}
	packageDigest := c.Params("packageDigest")
	digest := c.Params("digest")
	if !frontendDigestPattern.MatchString(packageDigest) || !frontendDigestPattern.MatchString(digest) {
		return publicFrontendNotFound()
	}
	asset, err := runtime.PublicAsset(
		c.Context(), c.Params("extensionId"), packageDigest, digest, c.Params("handle"),
	)
	if err != nil {
		return mapPublicFrontendError(err)
	}
	c.Set(fiber.HeaderContentType, asset.ContentType)
	c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
	c.Set(fiber.HeaderETag, asset.ETag)
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Cross-Origin-Resource-Policy", "same-origin")
	c.Set("X-SForum-Asset-Digest", asset.Digest)
	c.Set("X-SForum-Asset-Integrity", asset.Integrity)
	return c.Send(asset.Body)
}

func mapPublicFrontendError(err error) error {
	// Public callers cannot distinguish disabled, revoked, stale, changed, or
	// undeclared artifacts. The exact trust inspector remains admin-only.
	if errors.Is(err, extensions.ErrPublicFrontendUnavailable) ||
		errors.Is(err, extensions.ErrFrontendPackageChanged) ||
		errors.Is(err, extensions.ErrExtensionNotFound) {
		return publicFrontendNotFound()
	}
	return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeFrontendRuntimeUnavailable)
}

func publicFrontendNotFound() error {
	return fiber.NewError(fiber.StatusNotFound, extensions.CodeFrontendTrustNotFound)
}

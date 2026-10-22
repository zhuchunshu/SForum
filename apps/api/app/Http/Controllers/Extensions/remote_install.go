package extensionscontroller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	extensiondownload "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionDownload"
)

const (
	CodeRemoteURLInvalid         = "extension.remote_url_invalid"
	CodeRemoteFetchFailed        = "extension.remote_fetch_failed"
	CodeRemoteArchiveTooLarge    = "extension.remote_archive_too_large"
	CodeRemoteDigestMismatch     = "extension.remote_digest_mismatch"
	CodeRemoteInstallUnavailable = "extension.remote_install_unavailable"
)

// ArchiveFetcher downloads an extension ZIP from a URL. The concrete
// implementation is extensiondownload.Downloader; tests inject fakes.
type ArchiveFetcher interface {
	Fetch(context.Context, string) (extensiondownload.Archive, error)
}

// archiveInstaller is the subset of the extension service used by URL install.
// The controller holds the real service in production and a fake in tests.
type archiveInstaller interface {
	InstallOrUpgradeArchive(context.Context, identity.Actor, extensions.ArchiveInput) (extensions.InstallResult, error)
}

type remoteInstallInput struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

// installFromURL downloads a public HTTPS ZIP and installs it as an inert
// package. Authorization is checked before the outbound request so an actor
// without install authority cannot use this route as a network probe.
func (h *Controller) installFromURL(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	if !extensions.CanManageAnyArchive(actor) {
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	}
	if h.remoteArchives == nil || h.installer == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, CodeRemoteInstallUnavailable)
	}

	var input remoteInstallInput
	if err := c.Bind().Body(&input); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}
	expectedDigest := strings.ToLower(strings.TrimSpace(input.SHA256))
	if expectedDigest != "" && !isSHA256Hex(expectedDigest) {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	archive, err := h.remoteArchives.Fetch(c.Context(), strings.TrimSpace(input.URL))
	if err != nil {
		return mapExtensionError(err)
	}
	if expectedDigest != "" && !strings.EqualFold(expectedDigest, archive.Digest) {
		return fiber.NewError(fiber.StatusUnprocessableEntity, CodeRemoteDigestMismatch)
	}

	result, err := h.installer.InstallOrUpgradeArchive(c.Context(), actor, extensions.ArchiveInput{
		FileName:     archive.FileName,
		Data:         archive.Data,
		SourceURL:    archive.SourceURL,
		SourceDigest: archive.Digest,
	})
	if err != nil {
		return mapExtensionError(err)
	}

	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set("X-Content-Type-Options", "nosniff")
	return apphttp.Created(c, result)
}

// mapRemoteInstallError maps download-boundary errors to stable API reasons.
// It returns nil for unrelated errors so mapExtensionError can continue.
func mapRemoteInstallError(err error) error {
	switch {
	case errors.Is(err, extensiondownload.ErrInvalidURL):
		return fiber.NewError(fiber.StatusUnprocessableEntity, CodeRemoteURLInvalid)
	case errors.Is(err, extensiondownload.ErrTooLarge):
		return fiber.NewError(fiber.StatusRequestEntityTooLarge, CodeRemoteArchiveTooLarge)
	case errors.Is(err, extensiondownload.ErrNotArchive):
		return fiber.NewError(fiber.StatusUnprocessableEntity, extensions.CodeInvalidArchive)
	case errors.Is(err, extensiondownload.ErrFetchFailed):
		return fiber.NewError(fiber.StatusBadGateway, CodeRemoteFetchFailed)
	default:
		return nil
	}
}

func isSHA256Hex(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

// archiveSourceDigest records the exact ZIP bytes for uploads so both install
// paths carry the same audit provenance. An empty body has no digest.
func archiveSourceDigest(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

package http

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
)

func errorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status := fiber.StatusInternalServerError
		reason := "internal_error"

		if fiberErr, ok := err.(*fiber.Error); ok {
			status = fiberErr.Code
			reason = normalizeFiberErrorReason(status, fiberErr.Message)
		} else {
			logger.Error("request failed", "error", err)
		}

		return ErrorResponse(c, status, reason)
	}
}

func normalizeFiberErrorReason(status int, message string) string {
	switch {
	case message == "":
		return "internal_error"
	case status == fiber.StatusNotFound && message == "Not Found":
		return "not_found"
	case status == fiber.StatusMethodNotAllowed && message == "Method Not Allowed":
		return "method_not_allowed"
	default:
		return message
	}
}

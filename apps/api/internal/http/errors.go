package http

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
)

type problemResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func errorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status := fiber.StatusInternalServerError
		code := "internal_error"
		message := "服务器暂时不可用，请稍后再试。"

		if fiberErr, ok := err.(*fiber.Error); ok {
			status = fiberErr.Code
			code = "http_error"
			message = fiberErr.Message
		} else {
			logger.Error("request failed", "error", err)
		}

		return c.Status(status).JSON(problemResponse{
			Code:    code,
			Message: message,
		})
	}
}

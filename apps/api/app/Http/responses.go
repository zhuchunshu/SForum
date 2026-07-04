package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Localization"
)

const MessageOK = "ok"

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type ErrorData struct {
	Reason string              `json:"reason"`
	Fields map[string][]string `json:"fields,omitempty"`
}

func JSON(c fiber.Ctx, status int, messageKey string, data any) error {
	return c.Status(status).JSON(Response{
		Code:    status,
		Message: localization.Message(Locale(c), messageKey),
		Data:    data,
	})
}

func OK(c fiber.Ctx, data any) error {
	return JSON(c, fiber.StatusOK, MessageOK, data)
}

func Created(c fiber.Ctx, data any) error {
	return JSON(c, fiber.StatusCreated, MessageOK, data)
}

func NoData(c fiber.Ctx) error {
	return JSON(c, fiber.StatusOK, MessageOK, nil)
}

func ErrorResponse(c fiber.Ctx, status int, reason string) error {
	return JSON(c, status, reason, ErrorData{Reason: reason})
}

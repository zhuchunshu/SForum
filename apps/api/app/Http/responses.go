package http

import (
	"math"
	"strconv"
	"time"

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
	Reason            string              `json:"reason"`
	Fields            map[string][]string `json:"fields,omitempty"`
	RetryAfterSeconds *int                `json:"retryAfterSeconds,omitempty"`
	RetryAt           *time.Time          `json:"retryAt,omitempty"`
}

type APIError struct {
	Status  int
	Reason  string
	Fields  map[string][]string
	RetryAt *time.Time
}

func (e *APIError) Error() string {
	return e.Reason
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

func NewError(status int, reason string) *APIError {
	return &APIError{Status: status, Reason: reason}
}

func NewErrorWithFields(status int, reason string, fields map[string][]string) *APIError {
	return &APIError{Status: status, Reason: reason, Fields: fields}
}

func NewErrorWithRetryAt(status int, reason string, retryAt time.Time) *APIError {
	retryAt = retryAt.UTC()
	return &APIError{Status: status, Reason: reason, RetryAt: &retryAt}
}

func LocalizeFields(c fiber.Ctx, fields map[string][]string) map[string][]string {
	if len(fields) == 0 {
		return nil
	}

	localized := make(map[string][]string, len(fields))
	for field, keys := range fields {
		messages := make([]string, 0, len(keys))
		for _, key := range keys {
			messages = append(messages, localization.Message(Locale(c), key))
		}
		localized[field] = messages
	}
	return localized
}

func ErrorResponse(c fiber.Ctx, status int, reason string) error {
	return JSON(c, status, reason, ErrorData{Reason: reason})
}

func ErrorResponseWithFields(c fiber.Ctx, status int, reason string, fields map[string][]string) error {
	return JSON(c, status, reason, ErrorData{Reason: reason, Fields: fields})
}

func ErrorResponseWithRetryAt(c fiber.Ctx, status int, reason string, retryAt time.Time) error {
	retryAt = retryAt.UTC()
	retryAfterSeconds := max(1, int(math.Ceil(time.Until(retryAt).Seconds())))
	c.Set(fiber.HeaderRetryAfter, strconv.Itoa(retryAfterSeconds))
	return JSON(c, status, reason, ErrorData{
		Reason:            reason,
		RetryAfterSeconds: &retryAfterSeconds,
		RetryAt:           &retryAt,
	})
}

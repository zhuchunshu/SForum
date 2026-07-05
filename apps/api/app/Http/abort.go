package http

import "github.com/gofiber/fiber/v3"

func Abort(status int, reasons ...string) *APIError {
	reason := defaultAbortReason(status)
	if len(reasons) > 0 && reasons[0] != "" {
		reason = reasons[0]
	}
	return NewError(status, reason)
}

func AbortIf(condition bool, status int, reasons ...string) *APIError {
	if !condition {
		return nil
	}
	return Abort(status, reasons...)
}

func AbortUnless(condition bool, status int, reasons ...string) *APIError {
	if condition {
		return nil
	}
	return Abort(status, reasons...)
}

func defaultAbortReason(status int) string {
	switch status {
	case fiber.StatusBadRequest:
		return "validation.invalid"
	case fiber.StatusUnauthorized:
		return "auth.required"
	case fiber.StatusForbidden:
		return "permission.denied"
	case fiber.StatusNotFound:
		return "not_found"
	case fiber.StatusMethodNotAllowed:
		return "method_not_allowed"
	case fiber.StatusTooManyRequests:
		return "rate_limit.exceeded"
	default:
		return "internal_error"
	}
}

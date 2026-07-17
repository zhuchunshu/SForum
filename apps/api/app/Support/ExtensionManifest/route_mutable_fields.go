package extensionmanifest

import (
	"strconv"
	"strings"

	"golang.org/x/net/http/httpguts"
)

// ValidRouteMutableFields validates action direction, pointer shape, and the
// static Host-owned field policy. Runtime code must additionally reject header
// names nominated by the current Connection header.
func ValidRouteMutableFields(route ManifestRoute, rawRequestAuthority bool) bool {
	if len(route.MutableRequestFields) > 0 {
		switch route.Action {
		case RouteActionGlobalMiddleware, RouteActionBefore, RouteActionFilter, RouteActionWrap:
		default:
			return false
		}
	}
	if len(route.MutableResponseFields) > 0 {
		switch route.Action {
		case RouteActionFilter, RouteActionWrap, RouteActionAfter:
		default:
			return false
		}
	}
	return validRouteMutablePointerList(route.MutableRequestFields, func(value string) bool {
		return ValidRouteMutableRequestPointer(value, rawRequestAuthority)
	}) && validRouteMutablePointerList(route.MutableResponseFields, ValidRouteMutableResponsePointer)
}

func validRouteMutablePointerList(values []string, valid func(string) bool) bool {
	if len(values) > RouteMutableFieldsMaximumCount {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !valid(value) {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

// ValidRouteMutableRequestPointer validates the stable synthetic request
// document. Dynamic Connection tokens are additionally checked per request.
func ValidRouteMutableRequestPointer(value string, rawRequestAuthority bool) bool {
	tokens, ok := routeMutablePointerTokens(value)
	if !ok || len(tokens) == 0 {
		return false
	}
	switch tokens[0] {
	case "query":
		return len(tokens) <= 2 || len(tokens) == 3 && routeMutableArrayIndex(tokens[2])
	case "params":
		return len(tokens) <= 2
	case "body":
		return true
	case "headers":
		return len(tokens) >= 2 && len(tokens) <= 3 &&
			routeMutableRequestHeaderAllowed(tokens[1], rawRequestAuthority) &&
			(len(tokens) == 2 || routeMutableArrayIndex(tokens[2]))
	default:
		return false
	}
}

// ValidRouteMutableResponsePointer validates the stable synthetic response
// document. Status is scalar, so only the exact /status pointer is valid.
func ValidRouteMutableResponsePointer(value string) bool {
	tokens, ok := routeMutablePointerTokens(value)
	if !ok || len(tokens) == 0 {
		return false
	}
	switch tokens[0] {
	case "status":
		return len(tokens) == 1
	case "body":
		return true
	case "headers":
		return len(tokens) >= 2 && len(tokens) <= 3 && routeMutableResponseHeaderAllowed(tokens[1]) &&
			(len(tokens) == 2 || routeMutableArrayIndex(tokens[2]))
	default:
		return false
	}
}

func routeMutableArrayIndex(value string) bool {
	if value == "-" || value == "0" {
		return true
	}
	if value == "" || value[0] == '0' {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	_, err := strconv.ParseUint(value, 10, 31)
	return err == nil
}

func validRFC6901Pointer(value string) bool {
	// RFC 6901 的空字符串表示整个文档。字段级最小权限不允许授予该 root pointer。
	if value == "" || value != strings.TrimSpace(value) || len(value) > RouteMutableFieldMaximumBytes ||
		value[0] != '/' || strings.Count(value, "/") > RouteMutableFieldMaximumTokens {
		return false
	}
	for index := 1; index < len(value); index++ {
		if value[index] != '~' {
			continue
		}
		index++
		if index >= len(value) || value[index] != '0' && value[index] != '1' {
			return false
		}
	}
	return true
}

func routeMutablePointerTokens(pointer string) ([]string, bool) {
	if !validRFC6901Pointer(pointer) {
		return nil, false
	}
	parts := strings.Split(pointer[1:], "/")
	for index, part := range parts {
		parts[index] = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
	}
	return parts, true
}

func routeMutableRequestHeaderAllowed(name string, rawRequestAuthority bool) bool {
	canonical := strings.ToLower(strings.TrimSpace(name))
	if name != canonical || !routeMutableHeaderNameAllowed(canonical) {
		return false
	}
	switch canonical {
	case "host", "content-length", "idempotency-key", "proxy-authorization", "x-csrf-token", "connection",
		"keep-alive", "proxy-authenticate", "proxy-connection", "te", "trailer", "transfer-encoding", "upgrade":
		return false
	case "cookie", "authorization", "x-api-key", "x-auth-token":
		return rawRequestAuthority
	default:
		return true
	}
}

func routeMutableResponseHeaderAllowed(name string) bool {
	canonical := strings.ToLower(strings.TrimSpace(name))
	if name != canonical || !routeMutableHeaderNameAllowed(canonical) {
		return false
	}
	switch canonical {
	case "content-length", "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "proxy-connection",
		"set-cookie", "location", "idempotency-replayed", "te", "trailer", "transfer-encoding", "upgrade":
		return false
	default:
		return true
	}
}

func routeMutableHeaderNameAllowed(canonical string) bool {
	return canonical != "" && !strings.HasPrefix(canonical, "x-sforum-") && httpguts.ValidHeaderFieldName(canonical)
}

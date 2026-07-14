package extensionopenapi

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

var openAPIMethodOrder = []string{"GET", "PUT", "POST", "DELETE", "OPTIONS", "HEAD", "PATCH", "TRACE"}

var pathParameterPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var openAPIVersionPattern = regexp.MustCompile(`^3\.1\.(?:0|[1-9][0-9]*)(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$`)
var responseStatusPattern = regexp.MustCompile(`^(?:[1-5][0-9]{2}|[1-5]XX|default)$`)

func validateOpenAPIRoot(value any, fragment extensionmanifest.ManifestOpenAPIFragment) (map[string]any, error) {
	root, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: fragment %s root must be an object", ErrInvalidDocument, fragment.ID)
	}
	version, versionOK := canonicalStringField(root, "openapi")
	if !versionOK || !openAPIVersionPattern.MatchString(version) {
		return nil, fmt.Errorf("%w: fragment %s must use OpenAPI 3.1", ErrInvalidDocument, fragment.ID)
	}
	info, ok := root["info"].(map[string]any)
	_, titleOK := canonicalStringField(info, "title")
	_, infoVersionOK := canonicalStringField(info, "version")
	if !ok || !titleOK || !infoVersionOK {
		return nil, fmt.Errorf("%w: fragment %s requires info.title and info.version", ErrInvalidDocument, fragment.ID)
	}
	if _, exists := root["servers"]; exists {
		return nil, fmt.Errorf("%w: plugin root servers are forbidden", ErrInvalidDocument)
	}
	if _, exists := root["security"]; exists {
		return nil, fmt.Errorf("%w: plugin root security is forbidden; Host route policy owns security", ErrInvalidDocument)
	}
	if _, ok := root["paths"].(map[string]any); !ok {
		return nil, fmt.Errorf("%w: fragment %s paths must be an object", ErrInvalidDocument, fragment.ID)
	}
	return root, nil
}

func effectiveOperation(pathItem, operation map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(operation)+3)
	for key, value := range operation {
		result[key] = value
	}
	if _, exists := pathItem["servers"]; exists {
		return nil, fmt.Errorf("%w: plugin path servers are forbidden", ErrInvalidDocument)
	}
	if _, exists := operation["servers"]; exists {
		return nil, fmt.Errorf("%w: plugin operation servers are forbidden", ErrInvalidDocument)
	}
	if inherited, exists := pathItem["parameters"]; exists {
		pathParameters, ok := inherited.([]any)
		if !ok {
			return nil, fmt.Errorf("%w: path parameters must be an array", ErrInvalidDocument)
		}
		operationParameters, operationHasParameters := result["parameters"].([]any)
		if _, exists := result["parameters"]; exists && !operationHasParameters {
			return nil, fmt.Errorf("%w: operation parameters must be an array", ErrInvalidDocument)
		}
		merged := make([]any, 0, len(pathParameters)+len(operationParameters))
		merged = append(merged, pathParameters...)
		merged = append(merged, operationParameters...)
		result["parameters"] = merged
	}
	return result, nil
}

func validatePathItemKeys(pathItem map[string]any) error {
	known := map[string]bool{
		"summary": true, "description": true, "servers": true, "parameters": true, "$ref": true,
	}
	for _, method := range openAPIMethodOrder {
		known[strings.ToLower(method)] = true
	}
	for key := range pathItem {
		if known[key] || strings.HasPrefix(strings.ToLower(key), "x-") {
			continue
		}
		return fmt.Errorf("unknown Path Item field %q", key)
	}
	return nil
}

func validateOperation(
	operation map[string]any,
	pathValue, method string,
	fragment extensionmanifest.ManifestOpenAPIFragment,
	identity SourceIdentity,
	routes map[string]routeContract,
	artifact *loadedArtifact,
	sourcePath string,
) (GeneratedOperation, routeContract, error) {
	operationID, operationIDOK := canonicalStringField(operation, "operationId")
	if !operationIDOK || !strings.HasPrefix(operationID, fragment.Namespace+".") {
		return GeneratedOperation{}, routeContract{}, fmt.Errorf("%w: operationId %q is not in namespace %q", ErrContractMismatch, operationID, fragment.Namespace)
	}
	routeID, routeIDOK := canonicalStringField(operation, extRouteID)
	if !routeIDOK {
		return GeneratedOperation{}, routeContract{}, fmt.Errorf("%w: operation %s has a non-canonical route id", ErrContractMismatch, operationID)
	}
	route, exists := routes[routeMethodKey(routeID, method)]
	if !exists {
		return GeneratedOperation{}, routeContract{}, fmt.Errorf("%w: operation %s references unknown route %s %s", ErrContractMismatch, operationID, routeID, method)
	}
	if route.path != pathValue {
		return GeneratedOperation{}, routeContract{}, fmt.Errorf("%w: operation %s path %s does not match route path %s", ErrContractMismatch, operationID, pathValue, route.path)
	}
	checks := []struct {
		key      string
		expected string
		required bool
	}{
		{extContractVersion, route.route.ContractVersion, true},
		{extGuard, route.route.Guard, true},
		{extPermission, route.route.Permission, route.route.Permission != ""},
		{extRequestSchema, route.route.RequestSchema, route.route.RequestSchema != ""},
		{extResponseSchema, route.route.ResponseSchema, route.route.ResponseSchema != ""},
		{extRateLimit, route.policy.RateLimit, true},
		{extIdempotency, route.policy.Idempotency, true},
	}
	for _, check := range checks {
		actual, present := operation[check.key]
		text, stringValue := actual.(string)
		if check.required && (!present || !stringValue || text != check.expected) ||
			!check.required && present && (!stringValue || text != check.expected) {
			return GeneratedOperation{}, routeContract{}, fmt.Errorf("%w: operation %s metadata %s does not match %q", ErrContractMismatch, operationID, check.key, check.expected)
		}
	}
	if err := applyHostSecurity(operation, route.policy.Security); err != nil {
		return GeneratedOperation{}, routeContract{}, fmt.Errorf("%w: operation %s: %w", ErrContractMismatch, operationID, err)
	}
	if err := validatePathParameters(operation, pathValue, artifact, sourcePath); err != nil {
		return GeneratedOperation{}, routeContract{}, fmt.Errorf("%w: operation %s: %w", ErrInvalidDocument, operationID, err)
	}
	responses, ok := operation["responses"].(map[string]any)
	if !ok || len(responses) == 0 {
		return GeneratedOperation{}, routeContract{}, fmt.Errorf("%w: operation %s requires responses", ErrInvalidDocument, operationID)
	}
	responseHasSchema, err := validateResponses(responses, artifact, sourcePath)
	if err != nil {
		return GeneratedOperation{}, routeContract{}, fmt.Errorf("%w: operation %s responses: %w", ErrInvalidDocument, operationID, err)
	}
	requestBodyValue, requestBodyExists := operation["requestBody"]
	requestHasSchema, hasRequestBody, err := validateRequestBody(requestBodyValue, requestBodyExists, artifact, sourcePath)
	if err != nil {
		return GeneratedOperation{}, routeContract{}, fmt.Errorf("%w: operation %s request body: %w", ErrInvalidDocument, operationID, err)
	}
	if route.route.RequestSchema != "" && (!hasRequestBody || !requestHasSchema) || route.route.RequestSchema == "" && hasRequestBody {
		return GeneratedOperation{}, routeContract{}, fmt.Errorf("%w: operation %s request body disagrees with route schema", ErrContractMismatch, operationID)
	}
	if route.route.ResponseSchema != "" && !responseHasSchema {
		return GeneratedOperation{}, routeContract{}, fmt.Errorf("%w: operation %s response body disagrees with route schema", ErrContractMismatch, operationID)
	}
	return GeneratedOperation{
		OperationID: operationID, RouteID: route.route.ID, ContractVersion: route.route.ContractVersion,
		Path: pathValue, Method: method, Guard: route.route.Guard, Permission: route.route.Permission,
		RequestSchema: route.route.RequestSchema, ResponseSchema: route.route.ResponseSchema,
		RateLimit: route.policy.RateLimit, Idempotency: route.policy.Idempotency, Security: route.policy.Security,
		ExtensionID: identity.ExtensionID, ExtensionVersion: identity.ExtensionVersion,
		PackageDigest: identity.PackageDigest, FragmentID: identity.FragmentID, Namespace: identity.Namespace,
	}, route, nil
}

func canonicalOpenAPIPath(value string) (string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") ||
		strings.ContainsAny(value, "\\?#\x00") || strings.Contains(value, "..") || path.Clean(value) != value {
		return "", "", fmt.Errorf("invalid path %q", value)
	}
	if value == "/" {
		return value, value, nil
	}
	segments := strings.Split(strings.TrimPrefix(value, "/"), "/")
	normalized := make([]string, len(segments))
	signature := make([]string, len(segments))
	for index, segment := range segments {
		name := ""
		switch {
		case strings.HasPrefix(segment, ":"):
			name = strings.TrimPrefix(segment, ":")
		case segment == "*":
			name = "path"
		case strings.HasPrefix(segment, "*"):
			name = strings.TrimPrefix(segment, "*")
		case strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}"):
			name = strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
		}
		if name != "" {
			if !pathParameterPattern.MatchString(name) || (strings.HasPrefix(segment, "*") && index != len(segments)-1) {
				return "", "", fmt.Errorf("invalid path parameter %q", segment)
			}
			normalized[index] = "{" + name + "}"
			signature[index] = "{}"
			continue
		}
		if segment == "" || strings.ContainsAny(segment, "{}:*") {
			return "", "", fmt.Errorf("invalid path segment %q", segment)
		}
		normalized[index] = segment
		signature[index] = segment
	}
	return "/" + strings.Join(normalized, "/"), "/" + strings.Join(signature, "/"), nil
}

func validHTTPMethod(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", char) {
			continue
		}
		return false
	}
	return true
}

func validOpenAPIMethod(value string) bool {
	index := sort.SearchStrings(openAPIMethodOrderSorted, value)
	return index < len(openAPIMethodOrderSorted) && openAPIMethodOrderSorted[index] == value
}

var openAPIMethodOrderSorted = []string{"DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT", "TRACE"}

func pathMethodKey(signature, method string) string { return signature + "\x00" + method }

func canonicalStringField(object map[string]any, key string) (string, bool) {
	value, ok := object[key].(string)
	return value, ok && value != "" && value == strings.TrimSpace(value)
}

func applyHostSecurity(operation map[string]any, policy string) error {
	var expected []any
	switch policy {
	case SecurityPublic:
		expected = []any{}
	case SecurityAuthenticated:
		expected = []any{
			map[string]any{"cookieAuth": []any{}},
			map[string]any{"bearerAuth": []any{}},
		}
	default:
		return fmt.Errorf("unknown Host security policy %q", policy)
	}
	if declared, exists := operation["security"]; exists && !equivalentSecurity(declared, policy) {
		return fmt.Errorf("plugin security contradicts Host policy %q", policy)
	}
	operation["security"] = expected
	return nil
}

func equivalentSecurity(value any, policy string) bool {
	requirements, ok := value.([]any)
	if !ok {
		return false
	}
	if policy == SecurityPublic {
		return len(requirements) == 0
	}
	if policy != SecurityAuthenticated || len(requirements) != 2 {
		return false
	}
	seen := map[string]bool{}
	for _, value := range requirements {
		requirement, ok := value.(map[string]any)
		if !ok || len(requirement) != 1 {
			return false
		}
		for scheme, scopesValue := range requirement {
			scopes, ok := scopesValue.([]any)
			if !ok || len(scopes) != 0 || scheme != "cookieAuth" && scheme != "bearerAuth" || seen[scheme] {
				return false
			}
			seen[scheme] = true
		}
	}
	return seen["cookieAuth"] && seen["bearerAuth"]
}

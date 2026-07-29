package routes

import (
	"fmt"
	pathpkg "path"
	"strings"
)

type segmentKind uint8

const (
	segmentCatchAll segmentKind = iota + 1
	segmentParam
	segmentStatic
)

type pathSegment struct {
	kind  segmentKind
	value string
}

type compiledPath struct {
	pattern        string
	signature      string
	segments       []pathSegment
	parameterCount int
}

func compileRoutePath(pattern string) (compiledPath, error) {
	if pattern == "" || pattern != strings.TrimSpace(pattern) || !strings.HasPrefix(pattern, "/") ||
		len(pattern) > 1 && (pattern[1] == '/' || pattern[1] == '\\') ||
		strings.ContainsAny(pattern, "\\?#\x00") || strings.Contains(pattern, "..") {
		return compiledPath{}, fmt.Errorf("%w: invalid path %q", ErrInvalidRoute, pattern)
	}
	if clean := pathpkg.Clean(pattern); clean != pattern {
		return compiledPath{}, fmt.Errorf("%w: path %q is not canonical", ErrInvalidRoute, pattern)
	}
	if pattern == "/" {
		return compiledPath{pattern: pattern, signature: pattern}, nil
	}

	parts := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	segments := make([]pathSegment, 0, len(parts))
	signature := make([]string, 0, len(parts))
	parameterCount := 0
	for index, part := range parts {
		segment, err := compilePathSegment(part, index == len(parts)-1)
		if err != nil {
			return compiledPath{}, err
		}
		segments = append(segments, segment)
		switch segment.kind {
		case segmentStatic:
			signature = append(signature, "s:"+segment.value)
		case segmentParam:
			signature = append(signature, ":")
			parameterCount++
		case segmentCatchAll:
			signature = append(signature, "*")
			parameterCount++
		}
	}
	return compiledPath{
		pattern: pattern, signature: "/" + strings.Join(signature, "/"), segments: segments,
		parameterCount: parameterCount,
	}, nil
}

func compilePathSegment(value string, final bool) (pathSegment, error) {
	if value == "" {
		return pathSegment{}, fmt.Errorf("%w: empty path segment", ErrInvalidRoute)
	}
	if value == "*" || strings.HasPrefix(value, "*") {
		if !final || (value != "*" && !validPathName(strings.TrimPrefix(value, "*"))) {
			return pathSegment{}, fmt.Errorf("%w: invalid catch-all segment %q", ErrInvalidRoute, value)
		}
		name := strings.TrimPrefix(value, "*")
		if name == "" {
			name = "path"
		}
		return pathSegment{kind: segmentCatchAll, value: name}, nil
	}
	if strings.HasPrefix(value, ":") {
		name := strings.TrimPrefix(value, ":")
		if !validPathName(name) {
			return pathSegment{}, fmt.Errorf("%w: invalid parameter segment %q", ErrInvalidRoute, value)
		}
		return pathSegment{kind: segmentParam, value: name}, nil
	}
	if strings.ContainsAny(value, ":*") {
		return pathSegment{}, fmt.Errorf("%w: invalid static segment %q", ErrInvalidRoute, value)
	}
	return pathSegment{kind: segmentStatic, value: value}, nil
}

func validPathName(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9' || char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}

func normalizeRequestPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if index := strings.IndexByte(value, '?'); index >= 0 {
		value = value[:index]
	}
	if value == "" || !strings.HasPrefix(value, "/") ||
		len(value) > 1 && (value[1] == '/' || value[1] == '\\') ||
		strings.ContainsAny(value, "\\#\x00") || strings.Contains(value, "..") {
		return "", fmt.Errorf("%w: invalid request path", ErrInvalidRoute)
	}
	return pathpkg.Clean(value), nil
}

func (path compiledPath) match(requestPath string) (map[string]string, bool) {
	if path.pattern == "/" {
		return nil, requestPath == "/"
	}
	if !path.matchSegments(requestPath, nil) {
		return nil, false
	}
	if path.parameterCount == 0 {
		return nil, true
	}
	params := make(map[string]string, path.parameterCount)
	path.matchSegments(requestPath, params)
	return params, true
}

func (path compiledPath) matchSegments(requestPath string, params map[string]string) bool {
	cursor := 1
	for _, segment := range path.segments {
		if cursor >= len(requestPath) {
			return false
		}
		end := strings.IndexByte(requestPath[cursor:], '/')
		if end < 0 {
			end = len(requestPath)
		} else {
			end += cursor
		}
		value := requestPath[cursor:end]
		switch segment.kind {
		case segmentStatic:
			if value != segment.value {
				return false
			}
		case segmentParam:
			if value == "" {
				return false
			}
			if params != nil {
				params[segment.value] = value
			}
		case segmentCatchAll:
			value = requestPath[cursor:]
			if value == "" {
				return false
			}
			if params != nil {
				params[segment.value] = value
			}
			cursor = len(requestPath)
			continue
		}
		if end == len(requestPath) {
			cursor = end
		} else {
			cursor = end + 1
		}
	}
	return cursor == len(requestPath)
}

func routePathParametersCompatible(source, target compiledPath) bool {
	sourceKinds := dynamicPathSegmentKinds(source)
	targetKinds := dynamicPathSegmentKinds(target)
	if len(sourceKinds) != len(targetKinds) {
		return false
	}
	for index := range sourceKinds {
		if sourceKinds[index] != targetKinds[index] {
			return false
		}
	}
	return true
}

func dynamicPathSegmentKinds(path compiledPath) []segmentKind {
	result := make([]segmentKind, 0, path.parameterCount)
	for _, segment := range path.segments {
		if segment.kind != segmentStatic {
			result = append(result, segment.kind)
		}
	}
	return result
}

func materializeTargetRoutePath(source, target compiledPath, params map[string]string) (string, error) {
	if !routePathParametersCompatible(source, target) {
		return "", fmt.Errorf("%w: route mapping path parameters are incompatible", ErrInvalidExecutionPlan)
	}
	values := make([]string, 0, source.parameterCount)
	for _, segment := range source.segments {
		if segment.kind == segmentStatic {
			continue
		}
		value, ok := params[segment.value]
		if !ok || value == "" || segment.kind == segmentParam && strings.Contains(value, "/") {
			return "", fmt.Errorf("%w: route mapping path parameter is unavailable", ErrInvalidExecutionPlan)
		}
		values = append(values, value)
	}
	parts := make([]string, 0, len(target.segments))
	valueIndex := 0
	for _, segment := range target.segments {
		if segment.kind == segmentStatic {
			parts = append(parts, segment.value)
			continue
		}
		parts = append(parts, values[valueIndex])
		valueIndex++
	}
	if len(parts) == 0 {
		return "/", nil
	}
	return "/" + strings.Join(parts, "/"), nil
}

// comparePathSpecificity returns a positive value when left must match before right.
func comparePathSpecificity(left, right compiledPath) int {
	limit := min(len(left.segments), len(right.segments))
	for index := 0; index < limit; index++ {
		if left.segments[index].kind != right.segments[index].kind {
			return int(left.segments[index].kind) - int(right.segments[index].kind)
		}
	}
	if len(left.segments) != len(right.segments) {
		return len(left.segments) - len(right.segments)
	}
	return 0
}

package extensionopenapi

import (
	"fmt"
	"path"
	"strconv"
	"strings"
)

type foundReference struct {
	value string
}

func collectReferences(value any) ([]string, error) {
	result := make([]string, 0)
	var walk func(any, int) error
	walk = func(current any, depth int) error {
		if depth > maxDocumentDepth {
			return fmt.Errorf("%w: reference walk exceeds depth %d", ErrResourceBudget, maxDocumentDepth)
		}
		switch typed := current.(type) {
		case map[string]any:
			for key, child := range typed {
				if key == "$ref" {
					reference, ok := child.(string)
					if !ok || strings.TrimSpace(reference) != reference || reference == "" {
						return fmt.Errorf("$ref must be a non-empty string")
					}
					result = append(result, reference)
					if len(result) > maxReferences {
						return fmt.Errorf("%w: document reference count exceeds %d", ErrResourceBudget, maxReferences)
					}
				}
				if err := walk(child, depth+1); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range typed {
				if err := walk(child, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(value, 0); err != nil {
		return nil, err
	}
	return result, nil
}

func resolvePackageReference(sourcePath, reference string) (targetPath, pointer string, err error) {
	if strings.ContainsAny(reference, "\\?\x00%") || strings.Contains(reference, "://") ||
		strings.HasPrefix(reference, "/") || strings.HasPrefix(reference, "//") {
		return "", "", fmt.Errorf("%w: absolute, URL, escaped, and query references are forbidden: %q", ErrUnsafeReference, reference)
	}
	filePart, fragment, hasFragment := strings.Cut(reference, "#")
	if strings.Contains(filePart, ":") || strings.Contains(fragment, "#") {
		return "", "", fmt.Errorf("%w: invalid reference %q", ErrUnsafeReference, reference)
	}
	if filePart == "" {
		targetPath = sourcePath
	} else {
		targetPath = path.Join(path.Dir(sourcePath), filePart)
		if targetPath == ".." || strings.HasPrefix(targetPath, "../") || targetPath == "." || targetPath == "" {
			return "", "", fmt.Errorf("%w: reference escapes package: %q", ErrUnsafeReference, reference)
		}
	}
	if hasFragment && fragment != "" && !strings.HasPrefix(fragment, "/") {
		return "", "", fmt.Errorf("%w: only JSON Pointer fragments are allowed: %q", ErrUnsafeReference, reference)
	}
	return targetPath, fragment, nil
}

func (a *loadedArtifact) validateReferences() error {
	for sourcePath, document := range a.documents {
		refs, err := collectReferences(document)
		if err != nil {
			return fmt.Errorf("%w: %s: %w", ErrUnsafeReference, sourcePath, err)
		}
		for _, reference := range refs {
			targetPath, pointer, err := resolvePackageReference(sourcePath, reference)
			if err != nil {
				return err
			}
			target, exists := a.documents[targetPath]
			if !exists {
				return fmt.Errorf("%w: unresolved package file %q from %s", ErrUnsafeReference, targetPath, sourcePath)
			}
			if _, err := resolveJSONPointer(target, pointer); err != nil {
				return fmt.Errorf("%w: unresolved reference %q from %s: %w", ErrUnsafeReference, reference, sourcePath, err)
			}
		}
	}
	return nil
}

func resolveJSONPointer(root any, pointer string) (any, error) {
	if pointer == "" {
		return root, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("invalid JSON Pointer")
	}
	current := root
	segments := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	if len(segments) > maxDocumentDepth {
		return nil, fmt.Errorf("%w: JSON Pointer exceeds depth %d", ErrResourceBudget, maxDocumentDepth)
	}
	for _, raw := range segments {
		segment, err := unescapePointerSegment(raw)
		if err != nil {
			return nil, err
		}
		switch typed := current.(type) {
		case map[string]any:
			child, exists := typed[segment]
			if !exists {
				return nil, fmt.Errorf("object key %q does not exist", segment)
			}
			current = child
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, fmt.Errorf("array index %q does not exist", segment)
			}
			current = typed[index]
		default:
			return nil, fmt.Errorf("pointer traverses a scalar")
		}
	}
	return current, nil
}

func unescapePointerSegment(value string) (string, error) {
	var result strings.Builder
	for index := 0; index < len(value); index++ {
		if value[index] != '~' {
			result.WriteByte(value[index])
			continue
		}
		if index+1 >= len(value) || value[index+1] != '0' && value[index+1] != '1' {
			return "", fmt.Errorf("invalid JSON Pointer escape")
		}
		index++
		if value[index] == '0' {
			result.WriteByte('~')
		} else {
			result.WriteByte('/')
		}
	}
	return result.String(), nil
}

func rewriteReferences(value any, sourcePath string, sourceKeys map[string]string) (any, error) {
	return rewriteReferencesAtDepth(value, sourcePath, sourceKeys, 0)
}

func rewriteReferencesAtDepth(value any, sourcePath string, sourceKeys map[string]string, depth int) (any, error) {
	if depth > maxDocumentDepth {
		return nil, fmt.Errorf("%w: reference rewrite exceeds depth %d", ErrResourceBudget, maxDocumentDepth)
	}
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			if key == "$ref" {
				reference, ok := child.(string)
				if !ok {
					return nil, fmt.Errorf("%w: non-string reference", ErrUnsafeReference)
				}
				target, pointer, err := resolvePackageReference(sourcePath, reference)
				if err != nil {
					return nil, err
				}
				key, exists := sourceKeys[target]
				if !exists {
					return nil, fmt.Errorf("%w: missing source key for %s", ErrUnsafeReference, target)
				}
				result["$ref"] = "#/x-sforum-sources/" + key + pointer
				continue
			}
			rewritten, err := rewriteReferencesAtDepth(child, sourcePath, sourceKeys, depth+1)
			if err != nil {
				return nil, err
			}
			result[key] = rewritten
		}
		return result, nil
	case []any:
		result := make([]any, len(typed))
		for index, child := range typed {
			rewritten, err := rewriteReferencesAtDepth(child, sourcePath, sourceKeys, depth+1)
			if err != nil {
				return nil, err
			}
			result[index] = rewritten
		}
		return result, nil
	default:
		return typed, nil
	}
}

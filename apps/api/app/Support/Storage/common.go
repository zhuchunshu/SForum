package storage

import (
	"net/url"
	"path"
	"strings"
)

func normalizeObjectKey(key string) (string, error) {
	key = strings.TrimSpace(strings.ReplaceAll(key, "\\", "/"))
	if key == "" || strings.HasPrefix(key, "/") {
		return "", ErrInvalidKey
	}
	for _, part := range strings.Split(key, "/") {
		if part == "." || part == ".." {
			return "", ErrInvalidKey
		}
	}
	cleaned := path.Clean("/" + key)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." || cleaned == "" || strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, "/../") {
		return "", ErrInvalidKey
	}
	return cleaned, nil
}

func joinRemotePath(root string, key string) (string, error) {
	key, err := normalizeObjectKey(key)
	if err != nil {
		return "", err
	}
	root = strings.TrimSpace(strings.ReplaceAll(root, "\\", "/"))
	if root == "" || root == "." {
		return key, nil
	}
	if strings.Contains(root, "..") {
		return "", ErrInvalidConfig
	}
	return path.Join(root, key), nil
}

func joinPublicURL(base string, key string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	key, err := normalizeObjectKey(key)
	if err != nil {
		return ""
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimRight(base, "/") + "/" + key
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + key
	return parsed.String()
}

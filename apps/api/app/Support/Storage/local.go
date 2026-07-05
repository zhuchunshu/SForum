package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalAdapter struct {
	root          string
	publicBaseURL string
}

func NewLocalAdapter(root string, publicBaseURL string) (*LocalAdapter, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, ErrInvalidConfig
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &LocalAdapter{root: abs, publicBaseURL: strings.TrimSpace(publicBaseURL)}, nil
}

func (a *LocalAdapter) Put(_ context.Context, key string, input PutInput) error {
	target, err := a.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, input.Reader); err != nil {
		_ = file.Close()
		_ = os.Remove(target)
		return err
	}
	return file.Close()
}

func (a *LocalAdapter) Open(_ context.Context, key string) (io.ReadCloser, error) {
	target, err := a.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(target)
}

func (a *LocalAdapter) Delete(_ context.Context, key string) error {
	target, err := a.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (a *LocalAdapter) Stat(_ context.Context, key string) (ObjectInfo, error) {
	target, err := a.path(key)
	if err != nil {
		return ObjectInfo{}, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return ObjectInfo{}, err
	}
	return ObjectInfo{Key: key, Size: info.Size(), ModifiedAt: info.ModTime()}, nil
}

func (a *LocalAdapter) Exists(ctx context.Context, key string) (bool, error) {
	_, err := a.Stat(ctx, key)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (a *LocalAdapter) PublicURL(key string) string {
	return joinPublicURL(a.publicBaseURL, key)
}

func (a *LocalAdapter) SignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return a.PublicURL(key), nil
}

func (a *LocalAdapter) Probe(_ context.Context) error {
	return os.MkdirAll(a.root, 0o750)
}

func (a *LocalAdapter) path(key string) (string, error) {
	key, err := normalizeObjectKey(key)
	if err != nil {
		return "", err
	}
	target := filepath.Join(a.root, filepath.FromSlash(key))
	if target != a.root && !strings.HasPrefix(target, a.root+string(os.PathSeparator)) {
		return "", ErrInvalidKey
	}
	return target, nil
}

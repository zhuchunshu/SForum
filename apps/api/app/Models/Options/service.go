package options

import (
	"context"
	"strings"
	"sync"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

const defaultCacheTTL = 30 * time.Second

type Service struct {
	store    Store
	cacheTTL time.Duration

	mu        sync.RWMutex
	cached    map[string]string
	expiresAt time.Time
}

func NewService(store Store) *Service {
	return NewServiceWithCacheTTL(store, defaultCacheTTL)
}

func NewServiceWithCacheTTL(store Store, cacheTTL time.Duration) *Service {
	if cacheTTL <= 0 {
		cacheTTL = defaultCacheTTL
	}
	return &Service{store: store, cacheTTL: cacheTTL}
}

func (s *Service) List(ctx context.Context) ([]Option, error) {
	values, err := s.loadMap(ctx)
	if err != nil {
		return nil, err
	}

	names := []string{NameSiteName}
	options := make([]Option, 0, len(names))
	for _, name := range names {
		options = append(options, Option{Name: name, Value: values[name]})
	}
	return options, nil
}

func (s *Service) Get(ctx context.Context, name string) (Option, error) {
	name = normalizeName(name)
	if !isKnownOption(name) {
		return Option{}, ErrInvalidOption
	}

	values, err := s.loadMap(ctx)
	if err != nil {
		return Option{}, err
	}
	return Option{Name: name, Value: values[name]}, nil
}

func (s *Service) WebOption(ctx context.Context, name string) (string, error) {
	option, err := s.Get(ctx, name)
	if err != nil {
		return "", err
	}
	return option.Value, nil
}

func (s *Service) SiteName(ctx context.Context) (string, error) {
	return s.WebOption(ctx, NameSiteName)
}

func (s *Service) Update(ctx context.Context, actor identity.Actor, input UpdateInput) (Option, error) {
	if !actor.Can(identity.PermissionSettingsManage) {
		return Option{}, identity.ErrPermissionDenied
	}

	input.Name = normalizeName(input.Name)
	input.Value = normalizeValue(input.Name, input.Value)
	if !isKnownOption(input.Name) || !isValidValue(input.Name, input.Value) {
		return Option{}, ErrInvalidOption
	}

	option, err := s.store.Upsert(ctx, input)
	if err != nil {
		return Option{}, err
	}
	s.Invalidate()
	return option, nil
}

func (s *Service) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cached = nil
	s.expiresAt = time.Time{}
}

func (s *Service) loadMap(ctx context.Context) (map[string]string, error) {
	now := time.Now()

	s.mu.RLock()
	if s.cached != nil && now.Before(s.expiresAt) {
		cached := copyValues(s.cached)
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cached != nil && now.Before(s.expiresAt) {
		return copyValues(s.cached), nil
	}

	rows, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}

	values := defaultValues()
	for _, row := range rows {
		name := normalizeName(row.Name)
		if isKnownOption(name) {
			values[name] = normalizeValue(name, row.Value)
		}
	}

	s.cached = values
	s.expiresAt = now.Add(s.cacheTTL)
	return copyValues(values), nil
}

func defaultValues() map[string]string {
	return map[string]string{
		NameSiteName: "SForum",
	}
}

func copyValues(values map[string]string) map[string]string {
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func normalizeName(name string) string {
	return strings.TrimSpace(name)
}

func normalizeValue(name string, value string) string {
	value = strings.TrimSpace(value)
	switch name {
	case NameSiteName:
		return value
	default:
		return value
	}
}

func isKnownOption(name string) bool {
	switch name {
	case NameSiteName:
		return true
	default:
		return false
	}
}

func isValidValue(name string, value string) bool {
	switch name {
	case NameSiteName:
		return value != "" && len([]rune(value)) <= 80
	default:
		return false
	}
}

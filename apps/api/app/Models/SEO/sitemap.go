package seo

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	SitemapCategories = "categories"
	SitemapTags       = "tags"
	SitemapTopics     = "topics"
	SitemapProfiles   = "profiles"
)

var ErrInvalidSitemapRequest = errors.New("seo: invalid sitemap request")

type SitemapEntry struct {
	Path         string    `json:"path"`
	LastModified time.Time `json:"lastModified"`
}

type SitemapListInput struct {
	Type          string
	Page, PerPage int
}
type SitemapEntries struct {
	Items   []SitemapEntry `json:"items"`
	Page    int            `json:"page"`
	PerPage int            `json:"perPage"`
	HasMore bool           `json:"hasMore"`
}

type SitemapStore interface {
	ListSitemapEntries(ctx context.Context, contentType, topicURLMode string, limit, offset int) ([]SitemapEntry, error)
}

type SitemapOptions interface {
	WebOption(context.Context, string) (string, error)
}

type SitemapService struct {
	store   SitemapStore
	options SitemapOptions
}

func NewSitemapService(store SitemapStore, options SitemapOptions) *SitemapService {
	return &SitemapService{store: store, options: options}
}

func (s *SitemapService) List(ctx context.Context, input SitemapListInput) (SitemapEntries, error) {
	if !validSitemapType(input.Type) {
		return SitemapEntries{}, ErrInvalidSitemapRequest
	}
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PerPage < 1 {
		input.PerPage = 5000
	}
	if input.PerPage > 10000 {
		input.PerPage = 10000
	}
	result := SitemapEntries{Items: []SitemapEntry{}, Page: input.Page, PerPage: input.PerPage}
	if !s.enabled(ctx, "seo.sitemap.enabled", true) || !s.enabled(ctx, "seo.sitemap.include_forum_content", false) {
		return result, nil
	}
	policyType := strings.TrimSuffix(input.Type, "s")
	if input.Type == SitemapCategories {
		policyType = "category"
	}
	if !s.enabled(ctx, "seo.content_type."+policyType+".include_in_sitemap", policyType != "profile") {
		return result, nil
	}
	mode, _ := s.options.WebOption(ctx, "seo.topic_url_mode")
	items, err := s.store.ListSitemapEntries(ctx, input.Type, mode, input.PerPage+1, (input.Page-1)*input.PerPage)
	if err != nil {
		return SitemapEntries{}, err
	}
	if len(items) > input.PerPage {
		result.HasMore = true
		items = items[:input.PerPage]
	}
	result.Items = items
	return result, nil
}

func (s *SitemapService) enabled(ctx context.Context, name string, fallback bool) bool {
	value, err := s.options.WebOption(ctx, name)
	if err != nil || strings.TrimSpace(value) == "" {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "enabled", "true", "1", "yes", "on":
		return true
	case "disabled", "false", "0", "no", "off":
		return false
	default:
		return fallback
	}
}

func validSitemapType(value string) bool {
	return value == SitemapCategories || value == SitemapTags || value == SitemapTopics || value == SitemapProfiles
}

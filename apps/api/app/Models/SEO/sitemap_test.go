package seo

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSitemapServiceAppliesSettingsAndBoundedPagination(t *testing.T) {
	store := &fakeSitemapStore{items: []SitemapEntry{{Path: "/t/1/hello", LastModified: time.Now()}}}
	service := NewSitemapService(store, fakeSitemapOptions{
		"seo.sitemap.enabled":                       "enabled",
		"seo.sitemap.include_forum_content":         "enabled",
		"seo.content_type.topic.include_in_sitemap": "enabled",
	})

	result, err := service.List(context.Background(), SitemapListInput{Type: SitemapTopics, Page: 0, PerPage: 50000})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if result.Page != 1 || result.PerPage != 10000 || store.limit != 10001 || store.offset != 0 {
		t.Fatalf("unexpected bounded pagination: %#v, %#v", result, store)
	}
}

func TestSitemapServiceRejectsUnknownType(t *testing.T) {
	service := NewSitemapService(&fakeSitemapStore{}, fakeSitemapOptions{})
	_, err := service.List(context.Background(), SitemapListInput{Type: "comments"})
	if !errors.Is(err, ErrInvalidSitemapRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestSitemapServiceSuppressesDisabledOrProfileContent(t *testing.T) {
	store := &fakeSitemapStore{items: []SitemapEntry{{Path: "/u/alice", LastModified: time.Now()}}}
	service := NewSitemapService(store, fakeSitemapOptions{
		"seo.sitemap.enabled":                         "enabled",
		"seo.sitemap.include_forum_content":           "enabled",
		"seo.content_type.profile.include_in_sitemap": "disabled",
	})
	result, err := service.List(context.Background(), SitemapListInput{Type: SitemapProfiles})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(result.Items) != 0 || store.calls != 0 {
		t.Fatalf("disabled profile sitemap must be empty: %#v", result)
	}
}

type fakeSitemapStore struct {
	items                []SitemapEntry
	calls, limit, offset int
}

func (f *fakeSitemapStore) ListSitemapEntries(_ context.Context, _, _ string, limit, offset int) ([]SitemapEntry, error) {
	f.calls++
	f.limit = limit
	f.offset = offset
	return f.items, nil
}

type fakeSitemapOptions map[string]string

func (f fakeSitemapOptions) WebOption(_ context.Context, name string) (string, error) {
	return f[name], nil
}

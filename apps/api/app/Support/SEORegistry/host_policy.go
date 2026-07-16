package seoregistry

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

var ErrHostPolicyInvalid = errors.New("seo registry Host final policy configuration is invalid")

// HostFinalPolicyConfig is populated from existing site/SEO options. It adds
// no product setting of its own; Core remains the final indexing authority.
type HostFinalPolicyConfig struct {
	SiteURL               string
	SupportedLocales      []string
	AllowIndexing         bool
	SitemapEnabled        bool
	StructuredDataEnabled bool
}

type HostFinalPolicy struct {
	origin                string
	locales               map[string]struct{}
	allowIndexing         bool
	sitemapEnabled        bool
	structuredDataEnabled bool
}

func NewHostFinalPolicy(config HostFinalPolicyConfig) (*HostFinalPolicy, error) {
	site, err := url.Parse(strings.TrimSpace(config.SiteURL))
	if err != nil || site.Scheme == "" || site.Host == "" || site.User != nil ||
		(site.Scheme != "http" && site.Scheme != "https") {
		return nil, ErrHostPolicyInvalid
	}
	origin := strings.ToLower(site.Scheme) + "://" + strings.ToLower(site.Host)
	locales := make(map[string]struct{}, len(config.SupportedLocales))
	for _, raw := range config.SupportedLocales {
		locale := strings.TrimSpace(raw)
		if !validCanonicalLocale(locale) || locale == "x-default" {
			return nil, ErrHostPolicyInvalid
		}
		locales[locale] = struct{}{}
	}
	if len(locales) == 0 {
		return nil, ErrHostPolicyInvalid
	}
	return &HostFinalPolicy{
		origin: origin, locales: locales, allowIndexing: config.AllowIndexing,
		sitemapEnabled: config.SitemapEnabled, structuredDataEnabled: config.StructuredDataEnabled,
	}, nil
}

func (p *HostFinalPolicy) ValidateSEO(ctx context.Context, request FinalPolicyRequest) error {
	if p == nil || ctx == nil || request.Scope == "" {
		return ErrHostPolicyInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateDocument(request.Base); err != nil {
		return err
	}
	if err := validateDocument(request.Document); err != nil {
		return err
	}
	if !robotsOnlyTightened(request.Base.Robots, request.Document.Robots) ||
		(!p.allowIndexing && request.Document.Robots.Indexing != RobotsNoIndex) {
		return ErrPolicyDenied
	}
	if reservedRobotsMeta(request.Document.Meta) {
		return ErrPolicyDenied
	}
	if request.Document.CanonicalURL != "" && !p.sameOrigin(request.Document.CanonicalURL) {
		return ErrPolicyDenied
	}
	for _, link := range request.Document.Hreflang {
		if link.Locale != "x-default" {
			if _, allowed := p.locales[link.Locale]; !allowed {
				return ErrPolicyDenied
			}
		}
		if !p.sameOrigin(link.URL) {
			return ErrPolicyDenied
		}
	}
	if (!p.sitemapEnabled || request.Document.Robots.Indexing == RobotsNoIndex) && len(request.Document.Sitemap) > 0 {
		return ErrPolicyDenied
	}
	for _, entry := range request.Document.Sitemap {
		if !p.sameOrigin(entry.URL) {
			return ErrPolicyDenied
		}
	}
	if !p.structuredDataEnabled && len(request.Document.JSONLD) > 0 {
		return ErrPolicyDenied
	}
	return nil
}

func robotsOnlyTightened(base, result RobotsDirectives) bool {
	if base.Indexing == RobotsNoIndex && result.Indexing != RobotsNoIndex {
		return false
	}
	if base.Following == RobotsNoFollow && result.Following != RobotsNoFollow {
		return false
	}
	if base.NoArchive && !result.NoArchive || base.NoImageIndex && !result.NoImageIndex || base.NoSnippet && !result.NoSnippet {
		return false
	}
	return true
}

func reservedRobotsMeta(values []MetaTag) bool {
	for _, item := range values {
		if item.Attribute != "name" {
			continue
		}
		switch strings.ToLower(item.Key) {
		case "robots", "googlebot", "bingbot", "baiduspider":
			return true
		}
	}
	return false
}

func (p *HostFinalPolicy) sameOrigin(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return strings.ToLower(parsed.Scheme)+"://"+strings.ToLower(parsed.Host) == p.origin
}

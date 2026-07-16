package seoregistry

import (
	"reflect"
)

func validateMutation(before, after Document, contribution Contribution) error {
	if !onlyKindChanged(before, after, contribution.Kind) {
		return ErrMutationDenied
	}
	switch contribution.Action {
	case ActionAdd:
		if !validAddMutation(before, after, contribution.Kind) {
			return ErrMutationDenied
		}
	case ActionReplace, ActionFilter:
		// Replacement and filter providers may deliberately produce a no-op, but
		// still cannot cross their single declared family boundary.
	default:
		return ErrMutationDenied
	}
	return nil
}

func onlyKindChanged(before, after Document, kind string) bool {
	if kind != KindTitle && before.Title != after.Title {
		return false
	}
	if kind != KindMeta && !reflect.DeepEqual(before.Meta, after.Meta) {
		return false
	}
	if kind != KindCanonical && before.CanonicalURL != after.CanonicalURL {
		return false
	}
	if kind != KindRobots && before.Robots != after.Robots {
		return false
	}
	if kind != KindHreflang && !reflect.DeepEqual(before.Hreflang, after.Hreflang) {
		return false
	}
	if kind != KindSitemap && !reflect.DeepEqual(before.Sitemap, after.Sitemap) {
		return false
	}
	if kind != KindJSONLD && !reflect.DeepEqual(before.JSONLD, after.JSONLD) {
		return false
	}
	return true
}

func validAddMutation(before, after Document, kind string) bool {
	switch kind {
	case KindTitle:
		return before.Title == "" && after.Title != ""
	case KindCanonical:
		return before.CanonicalURL == "" && after.CanonicalURL != ""
	case KindRobots:
		return before.Robots == (RobotsDirectives{}) && after.Robots != (RobotsDirectives{})
	case KindMeta:
		return appendedSlice(before.Meta, after.Meta)
	case KindHreflang:
		return appendedSlice(before.Hreflang, after.Hreflang)
	case KindSitemap:
		return appendedSlice(before.Sitemap, after.Sitemap)
	case KindJSONLD:
		return appendedSlice(before.JSONLD, after.JSONLD)
	default:
		return false
	}
}

func appendedSlice[T any](before, after []T) bool {
	if len(after) <= len(before) {
		return false
	}
	for index := range before {
		if !reflect.DeepEqual(before[index], after[index]) {
			return false
		}
	}
	return true
}

func documentKindEmpty(document Document, kind string) bool {
	switch kind {
	case KindTitle:
		return document.Title == ""
	case KindCanonical:
		return document.CanonicalURL == ""
	case KindRobots:
		return document.Robots == (RobotsDirectives{})
	case KindMeta:
		return len(document.Meta) == 0
	case KindHreflang:
		return len(document.Hreflang) == 0
	case KindSitemap:
		return len(document.Sitemap) == 0
	case KindJSONLD:
		return len(document.JSONLD) == 0
	default:
		return true
	}
}

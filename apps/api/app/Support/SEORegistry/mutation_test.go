package seoregistry

import "testing"

func TestMutationActionMatrixCoversEverySEOFamily(t *testing.T) {
	for _, kind := range executionKindOrder {
		t.Run(kind, func(t *testing.T) {
			empty := Document{}
			first := documentWithKindValue(kind, false)
			second := documentWithKindValue(kind, true)
			for _, test := range []struct {
				action string
				before Document
				after  Document
			}{
				{action: ActionAdd, before: empty, after: first},
				{action: ActionReplace, before: first, after: second},
				{action: ActionFilter, before: first, after: second},
			} {
				if err := validateDocument(test.after); err != nil {
					t.Fatalf("fixture: %v", err)
				}
				if err := validateMutation(test.before, test.after, Contribution{Declaration: Declaration{
					Kind: kind, Action: test.action,
				}}); err != nil {
					t.Fatalf("%s %s: %v", kind, test.action, err)
				}
			}
		})
	}
}

func documentWithKindValue(kind string, alternate bool) Document {
	suffix := "one"
	if alternate {
		suffix = "two"
	}
	switch kind {
	case KindTitle:
		return Document{Title: "Title " + suffix}
	case KindMeta:
		return Document{Meta: []MetaTag{{Attribute: "name", Key: "description", Content: suffix}}}
	case KindCanonical:
		return Document{CanonicalURL: "https://forum.example/" + suffix}
	case KindRobots:
		if alternate {
			return Document{Robots: RobotsDirectives{Indexing: RobotsNoIndex, Following: RobotsNoFollow}}
		}
		return Document{Robots: RobotsDirectives{Indexing: RobotsIndex, Following: RobotsFollow}}
	case KindHreflang:
		return Document{Hreflang: []HreflangLink{{Locale: "en-US", URL: "https://forum.example/" + suffix}}}
	case KindSitemap:
		return Document{Sitemap: []SitemapEntry{{URL: "https://forum.example/" + suffix}}}
	case KindJSONLD:
		return Document{JSONLD: []JSONLDDocument{{Context: "https://schema.org", Type: "WebPage", URL: "https://forum.example/" + suffix}}}
	default:
		return Document{}
	}
}

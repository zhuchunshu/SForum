package queryregistry

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestCanonicalSemanticCacheTagsSortsOwnedLogicalTags(t *testing.T) {
	input := []string{" OWNER.PLUGIN.TOPICS ", "owner.plugin.members"}
	got, err := CanonicalSemanticCacheTags(" OWNER.PLUGIN ", input)
	if err != nil {
		t.Fatalf("canonical tags: %v", err)
	}
	want := []string{"owner.plugin.members", "owner.plugin.topics"}
	if !slices.Equal(got, want) {
		t.Fatalf("tags=%#v want=%#v", got, want)
	}
	got[0] = "changed"
	if input[0] != " OWNER.PLUGIN.TOPICS " {
		t.Fatalf("input mutated: %#v", input)
	}
}

func TestCanonicalSemanticCacheTagsRejectsUnsafeSets(t *testing.T) {
	tooMany := make([]string, maxCacheTagsPerQuery+1)
	for index := range tooMany {
		tooMany[index] = "owner.plugin.tag." + strings.Repeat("a", index+1)
	}
	tests := []struct {
		name  string
		owner string
		tags  []string
	}{
		{name: "empty", owner: "owner.plugin"},
		{name: "invalid owner", owner: "owner/plugin", tags: []string{"owner.plugin.tag"}},
		{name: "bare core owner", owner: "core", tags: []string{"core.tag"}},
		{name: "foreign", owner: "owner.plugin", tags: []string{"other.plugin.tag"}},
		{name: "owner itself", owner: "owner.plugin", tags: []string{"owner.plugin"}},
		{name: "duplicate after canonicalization", owner: "owner.plugin", tags: []string{"owner.plugin.tag", " OWNER.PLUGIN.TAG "}},
		{name: "too many", owner: "owner.plugin", tags: tooMany},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := CanonicalSemanticCacheTags(test.owner, test.tags); !errors.Is(err, ErrInvalid) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestSharedSemanticCacheTagsAreOwnerScopedAndStable(t *testing.T) {
	first, err := sharedSemanticCacheTags("owner.plugin", []string{"owner.plugin.topics"})
	if err != nil {
		t.Fatalf("first tags: %v", err)
	}
	second, err := sharedSemanticCacheTags("owner.plugin", []string{" OWNER.PLUGIN.TOPICS "})
	if err != nil {
		t.Fatalf("second tags: %v", err)
	}
	foreign, err := sharedSemanticCacheTags("other.plugin", []string{"other.plugin.topics"})
	if err != nil {
		t.Fatalf("foreign tags: %v", err)
	}
	if !slices.Equal(first, second) || len(first) != 1 || first[0] == foreign[0] ||
		!strings.HasPrefix(first[0], "query:shared:") || len(first[0]) != len("query:shared:")+32 {
		t.Fatalf("first=%#v second=%#v foreign=%#v", first, second, foreign)
	}
}

func TestExecutionCacheTagsUseCanonicalSharedSemanticMapping(t *testing.T) {
	plan := QueryPlan{
		CacheKey: "cache-key", Revision: 1, Digest: strings.Repeat("a", 64),
		ActorFingerprint: "actor", PolicyFingerprint: "policy", Locale: "zh-CN", Scope: "public",
		CacheTags: []string{"owner.plugin.topics"},
		Query:     QueryContribution{Artifact: Artifact{ExtensionID: "owner.plugin"}},
	}
	got := executionCacheTags(plan, strings.Repeat("b", 64), strings.Repeat("c", 64))
	shared, err := sharedSemanticCacheTags("owner.plugin", plan.CacheTags)
	if err != nil {
		t.Fatalf("shared tags: %v", err)
	}
	if len(got) != 2 || got[0] != shared[0] || got[1] == shared[0] {
		t.Fatalf("execution tags=%#v shared=%#v", got, shared)
	}
}

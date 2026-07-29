package storage

import "testing"

func TestCoreCandidates(t *testing.T) {
	got := CoreCandidates()
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Value != ProviderLocal || got[0].Kind != SelectionKindCore || !got[0].Available {
		t.Fatalf("local: %#v", got[0])
	}
}

func TestPluginCandidateAndMerge(t *testing.T) {
	p := PluginCandidate("sforum.s3", "S3 Compatible", "/extensions/sforum.s3/pages/settings")
	if p.Value != "plugin:sforum.s3" || p.Kind != SelectionKindPlugin || p.ExtensionID != "sforum.s3" {
		t.Fatalf("%#v", p)
	}
	merged := MergeCandidates(CoreCandidates(), []Candidate{p})
	if len(merged) != 2 || merged[1].Value != "plugin:sforum.s3" {
		t.Fatalf("merge: %#v", merged)
	}
}

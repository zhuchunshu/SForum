package marketplace

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSignedIndexResolveAndPolicy(t *testing.T) {
	key := []byte("marketplace-test-key-32bytes-long!!")
	svc := New(key, OperatorPolicy{
		AllowedChannels:          []string{ChannelStable, ChannelBeta},
		MaxVulnerabilitySeverity: "high",
		DirectUploadFallback:     true,
	})
	index := Index{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		ExpiresAt:     time.Now().UTC().Add(time.Hour),
		SignerID:      "test",
		Entries: []Entry{
			{
				ExtensionID: "demo.seo", Version: "1.2.0",
				PackageDigest: strings.Repeat("ab", 32), Channel: ChannelStable,
				Dependencies: []string{"demo.base"},
			},
			{
				ExtensionID: "demo.vuln", Version: "1.0.0",
				PackageDigest: strings.Repeat("cd", 32), Channel: ChannelStable,
				Notices: []Notice{{Kind: NoticeVulnerability, Summary: "RCE", Severity: "critical"}},
			},
			{
				ExtensionID: "demo.gone", Version: "1.0.0",
				PackageDigest: strings.Repeat("ef", 32), Channel: ChannelStable, Withdrawn: true,
			},
		},
	}
	sig, err := SignIndex(key, index)
	if err != nil {
		t.Fatal(err)
	}
	index.Signature = sig
	if err := svc.LoadIndex(index); err != nil {
		t.Fatal(err)
	}

	// Tamper fails signature (deep-copy entries so original index stays valid).
	bad := index
	bad.Entries = append([]Entry(nil), index.Entries...)
	bad.Entries[0].Version = "9.9.9"
	if err := svc.LoadIndex(bad); !errors.Is(err, ErrSignature) {
		t.Fatalf("tamper = %v", err)
	}
	// Reload good index.
	if err := svc.LoadIndex(index); err != nil {
		t.Fatal(err)
	}

	res, err := svc.Resolve("demo.seo", ChannelStable)
	if err != nil || res.PackageDigest == "" || len(res.Order) != 2 || res.Order[1] != "demo.seo" {
		t.Fatalf("resolve = %#v err=%v", res, err)
	}
	if _, err := svc.Resolve("demo.vuln", ChannelStable); !errors.Is(err, ErrPolicy) {
		t.Fatalf("critical vuln = %v", err)
	}
	if _, err := svc.Resolve("demo.gone", ChannelStable); !errors.Is(err, ErrWithdrawn) {
		t.Fatalf("withdrawn = %v", err)
	}
	if _, err := svc.Resolve("demo.seo", ChannelDev); !errors.Is(err, ErrPolicy) {
		t.Fatalf("channel policy = %v", err)
	}
	if !svc.DirectUploadAvailable() {
		t.Fatal("direct upload fallback should remain available")
	}
	list := svc.List(false)
	if len(list) != 2 {
		t.Fatalf("list = %#v", list)
	}
}

func TestStaleIndex(t *testing.T) {
	key := []byte("stale-key-for-marketplace-tests!!")
	svc := New(key, OperatorPolicy{AllowedChannels: []string{ChannelStable}, AllowUnsigned: false})
	index := Index{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt:     time.Now().UTC().Add(-time.Hour),
		Entries: []Entry{{
			ExtensionID: "demo.x", Version: "1.0.0",
			PackageDigest: strings.Repeat("11", 32), Channel: ChannelStable,
		}},
	}
	sig, err := SignIndex(key, index)
	if err != nil {
		t.Fatal(err)
	}
	index.Signature = sig
	if err := svc.LoadIndex(index); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Resolve("demo.x", ChannelStable); !errors.Is(err, ErrStale) {
		t.Fatalf("stale = %v", err)
	}
}

func TestUnsignedAllowedInDev(t *testing.T) {
	svc := New(nil, OperatorPolicy{AllowedChannels: []string{ChannelStable}, AllowUnsigned: true})
	if err := svc.LoadIndex(Index{
		SchemaVersion: SchemaVersion,
		Entries: []Entry{{
			ExtensionID: "demo.x", Version: "1.0.0",
			PackageDigest: strings.Repeat("22", 32), Channel: ChannelStable,
		}},
	}); err != nil {
		t.Fatal(err)
	}
}

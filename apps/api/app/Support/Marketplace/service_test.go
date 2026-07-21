package marketplace

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
)

func testKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, *Ed25519Verifier) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	v, err := NewEd25519Verifier("test-key", pub)
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv, v
}

func digestOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func signedIndex(t *testing.T, priv ed25519.PrivateKey, index Index) Index {
	t.Helper()
	index.SignerKind = SignerKindEd25519
	index.SignerID = "test-key"
	sig, err := SignIndexEd25519(priv, index)
	if err != nil {
		t.Fatal(err)
	}
	index.Signature = sig
	return index
}

func TestEd25519SignedIndexResolveAndPolicy(t *testing.T) {
	_, priv, verifier := testKeyPair(t)
	svc := NewWithOptions(Options{
		Policy: OperatorPolicy{
			AllowedChannels:          []string{ChannelStable, ChannelBeta},
			MaxVulnerabilitySeverity: "high",
			HostSForumVersion:        "1.0.0",
		},
		Verifier: verifier,
	})
	baseDigest := digestOf("base-pkg")
	seoDigest := digestOf("seo-pkg")
	index := Index{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		ExpiresAt:     time.Now().UTC().Add(time.Hour),
		Entries: []Entry{
			{
				ExtensionID: "demo.base", Version: "1.0.0",
				PackageDigest: baseDigest, Channel: ChannelStable,
				MinSForumVersion: ">=1.0.0",
			},
			{
				ExtensionID: "demo.seo", Version: "1.2.0",
				PackageDigest: seoDigest, Channel: ChannelStable,
				MinSForumVersion: ">=1.0.0",
				Dependencies: []DependencyConstraint{
					{ExtensionID: "demo.base", Version: "^1.0.0"},
				},
			},
			{
				ExtensionID: "demo.vuln", Version: "1.0.0",
				PackageDigest: digestOf("vuln"), Channel: ChannelStable,
				Notices: []Notice{{Kind: NoticeVulnerability, Summary: "RCE", Severity: "critical"}},
			},
			{
				ExtensionID: "demo.gone", Version: "1.0.0",
				PackageDigest: digestOf("gone"), Channel: ChannelStable, Withdrawn: true,
			},
			{
				ExtensionID: "demo.revoked", Version: "1.0.0",
				PackageDigest: digestOf("revoked"), Channel: ChannelStable,
				Notices: []Notice{{Kind: NoticeRevocation, Summary: "revoked by publisher"}},
			},
		},
	}
	index = signedIndex(t, priv, index)
	if err := svc.LoadIndex(index); err != nil {
		t.Fatal(err)
	}

	// Tamper nested field fails signature.
	bad := cloneIndex(index)
	bad.Entries[0].Version = "9.9.9"
	if err := svc.LoadIndex(bad); !errors.Is(err, ErrSignature) {
		t.Fatalf("tamper = %v", err)
	}
	if err := svc.LoadIndex(index); err != nil {
		t.Fatal(err)
	}

	res, err := svc.Resolve("demo.seo", ChannelStable)
	if err != nil {
		t.Fatal(err)
	}
	if res.PackageDigest != seoDigest || len(res.Order) != 2 {
		t.Fatalf("resolve = %#v", res)
	}
	if res.Order[0].ExtensionID != "demo.base" || res.Order[1].ExtensionID != "demo.seo" {
		t.Fatalf("order = %#v", res.Order)
	}
	if _, err := svc.Resolve("demo.vuln", ChannelStable); !errors.Is(err, ErrPolicy) {
		t.Fatalf("critical vuln = %v", err)
	}
	if _, err := svc.Resolve("demo.gone", ChannelStable); !errors.Is(err, ErrWithdrawn) {
		t.Fatalf("withdrawn = %v", err)
	}
	if _, err := svc.Resolve("demo.revoked", ChannelStable); !errors.Is(err, ErrPolicy) {
		t.Fatalf("revocation = %v", err)
	}
	if _, err := svc.Resolve("demo.seo", ChannelDev); !errors.Is(err, ErrPolicy) {
		t.Fatalf("channel policy = %v", err)
	}
	if !svc.DirectUploadAvailable() {
		t.Fatal("direct upload fallback should remain available")
	}
	list := svc.List(false)
	if len(list) < 2 {
		t.Fatalf("list = %#v", list)
	}
}

func TestDeepCopyIsolationAfterVerify(t *testing.T) {
	_, priv, verifier := testKeyPair(t)
	svc := NewWithOptions(Options{
		Policy:   OperatorPolicy{AllowedChannels: []string{ChannelStable}},
		Verifier: verifier,
	})
	index := Index{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		ExpiresAt:     time.Now().UTC().Add(time.Hour),
		Entries: []Entry{{
			ExtensionID: "demo.x", Version: "1.0.0",
			PackageDigest: digestOf("x"), Channel: ChannelStable,
			Dependencies: []DependencyConstraint{{ExtensionID: "demo.y", Version: "^1.0.0"}},
			Notices:      []Notice{{Kind: NoticeVulnerability, Summary: "info", Severity: "low"}},
		}},
	}
	index = signedIndex(t, priv, index)
	if err := svc.LoadIndex(index); err != nil {
		t.Fatal(err)
	}

	// 调用方修改 LoadIndex 入参嵌套切片不得污染内部快照。
	index.Entries[0].Dependencies[0].ExtensionID = "evil.mutated"
	index.Entries[0].Notices[0].Summary = "mutated"
	index.Entries[0].Version = "9.9.9"

	list := svc.List(true)
	if len(list) != 1 {
		t.Fatalf("list = %#v", list)
	}
	if list[0].Version != "1.0.0" {
		t.Fatalf("internal version polluted: %s", list[0].Version)
	}
	if list[0].Dependencies[0].ExtensionID != "demo.y" {
		t.Fatalf("dependencies polluted: %#v", list[0].Dependencies)
	}
	if list[0].Notices[0].Summary != "info" {
		t.Fatalf("notices polluted: %#v", list[0].Notices)
	}

	// List 返回值与内部隔离。
	list[0].Dependencies[0].ExtensionID = "list.mutated"
	list[0].Notices[0].Summary = "list-mutated"
	again := svc.List(true)
	if again[0].Dependencies[0].ExtensionID != "demo.y" || again[0].Notices[0].Summary != "info" {
		t.Fatalf("list isolation failed: %#v", again[0])
	}
}

func TestResolveResultIsolation(t *testing.T) {
	_, priv, verifier := testKeyPair(t)
	svc := NewWithOptions(Options{
		Policy: OperatorPolicy{
			AllowedChannels:   []string{ChannelStable},
			HostSForumVersion: "1.0.0",
		},
		Verifier: verifier,
	})
	index := Index{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		ExpiresAt:     time.Now().UTC().Add(time.Hour),
		Entries: []Entry{
			{ExtensionID: "demo.base", Version: "1.0.0", PackageDigest: digestOf("b"), Channel: ChannelStable},
			{
				ExtensionID: "demo.app", Version: "2.0.0", PackageDigest: digestOf("a"), Channel: ChannelStable,
				Dependencies: []DependencyConstraint{{ExtensionID: "demo.base", Version: ">=1.0.0"}},
			},
		},
	}
	index = signedIndex(t, priv, index)
	if err := svc.LoadIndex(index); err != nil {
		t.Fatal(err)
	}
	res, err := svc.Resolve("demo.app", ChannelStable)
	if err != nil {
		t.Fatal(err)
	}
	res.Order[0].ExtensionID = "mutated"
	res2, err := svc.Resolve("demo.app", ChannelStable)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Order[0].ExtensionID != "demo.base" {
		t.Fatalf("resolve isolation failed: %#v", res2.Order)
	}
}

func TestDependencyCycleAndConflict(t *testing.T) {
	_, priv, verifier := testKeyPair(t)
	svc := NewWithOptions(Options{
		Policy:   OperatorPolicy{AllowedChannels: []string{ChannelStable}},
		Verifier: verifier,
	})
	index := Index{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		ExpiresAt:     time.Now().UTC().Add(time.Hour),
		Entries: []Entry{
			{
				ExtensionID: "demo.a", Version: "1.0.0", PackageDigest: digestOf("a"), Channel: ChannelStable,
				Dependencies: []DependencyConstraint{{ExtensionID: "demo.b", Version: "^1.0.0"}},
			},
			{
				ExtensionID: "demo.b", Version: "1.0.0", PackageDigest: digestOf("b"), Channel: ChannelStable,
				Dependencies: []DependencyConstraint{{ExtensionID: "demo.a", Version: "^1.0.0"}},
			},
		},
	}
	index = signedIndex(t, priv, index)
	if err := svc.LoadIndex(index); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Resolve("demo.a", ChannelStable); !errors.Is(err, ErrCycle) {
		t.Fatalf("cycle = %v", err)
	}
}

func TestHostIncompatibleAndTimeWindow(t *testing.T) {
	_, priv, verifier := testKeyPair(t)
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	svc := NewWithOptions(Options{
		Policy: OperatorPolicy{
			AllowedChannels:   []string{ChannelStable},
			HostSForumVersion: "1.0.0",
		},
		Verifier: verifier,
		Now:      func() time.Time { return now },
	})
	index := Index{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   now.Add(-time.Hour),
		ExpiresAt:     now.Add(time.Hour),
		Entries: []Entry{
			{
				ExtensionID: "demo.future", Version: "1.0.0", PackageDigest: digestOf("f"), Channel: ChannelStable,
				MinSForumVersion: ">=2.0.0",
			},
			{
				ExtensionID: "demo.window", Version: "1.0.0", PackageDigest: digestOf("w"), Channel: ChannelStable,
				AvailableFrom: now.Add(2 * time.Hour),
			},
		},
	}
	index = signedIndex(t, priv, index)
	if err := svc.LoadIndex(index); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Resolve("demo.future", ChannelStable); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("incompatible = %v", err)
	}
	if _, err := svc.Resolve("demo.window", ChannelStable); !errors.Is(err, ErrNotFound) {
		t.Fatalf("time window = %v", err)
	}
}

func TestStaleIndex(t *testing.T) {
	_, priv, verifier := testKeyPair(t)
	svc := NewWithOptions(Options{
		Policy:   OperatorPolicy{AllowedChannels: []string{ChannelStable}},
		Verifier: verifier,
	})
	index := Index{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt:     time.Now().UTC().Add(-time.Hour),
		Entries: []Entry{{
			ExtensionID: "demo.x", Version: "1.0.0",
			PackageDigest: digestOf("x"), Channel: ChannelStable,
		}},
	}
	index = signedIndex(t, priv, index)
	if err := svc.LoadIndex(index); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Resolve("demo.x", ChannelStable); !errors.Is(err, ErrStale) {
		t.Fatalf("stale = %v", err)
	}
}

func TestUnsignedAllowedInDev(t *testing.T) {
	svc := NewWithOptions(Options{
		Policy: OperatorPolicy{AllowedChannels: []string{ChannelStable}, AllowUnsigned: true},
	})
	if err := svc.LoadIndex(Index{
		SchemaVersion: SchemaVersion,
		Entries: []Entry{{
			ExtensionID: "demo.x", Version: "1.0.0",
			PackageDigest: digestOf("x"), Channel: ChannelStable,
		}},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestInstallStageActivateRollbackBinding(t *testing.T) {
	_, priv, verifier := testKeyPair(t)
	installer := NewMemoryInstaller()
	svc := NewWithOptions(Options{
		Policy: OperatorPolicy{
			AllowedChannels:   []string{ChannelStable},
			HostSForumVersion: "1.0.0",
		},
		Verifier:  verifier,
		Installer: installer,
	})
	pkgBody := []byte("exact-package-bytes-for-demo-app")
	sum := sha256.Sum256(pkgBody)
	digest := hex.EncodeToString(sum[:])
	index := Index{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		ExpiresAt:     time.Now().UTC().Add(time.Hour),
		Entries: []Entry{{
			ExtensionID: "demo.app", Version: "1.0.0",
			PackageDigest: digest, Channel: ChannelStable,
		}},
	}
	index = signedIndex(t, priv, index)
	if err := svc.LoadIndex(index); err != nil {
		t.Fatal(err)
	}
	res, err := svc.Resolve("demo.app", ChannelStable)
	if err != nil {
		t.Fatal(err)
	}
	plan := InstallPlan{
		ResolveResult: res, Action: "stage",
		SourceDigest: digestOf("previous"), Actor: "admin",
	}
	staged, err := svc.StageInstall(context.Background(), plan, pkgBody)
	if err != nil {
		t.Fatal(err)
	}
	if !staged.PreflightOK || staged.StagedDigest != digest || staged.RolloutPlanID == "" {
		t.Fatalf("staged = %#v", staged)
	}
	if err := svc.ActivateStaged(context.Background(), plan, staged); err != nil {
		t.Fatal(err)
	}
	if installer.ActiveDigest("demo.app") != digest {
		t.Fatalf("active = %s", installer.ActiveDigest("demo.app"))
	}
	if err := svc.RollbackInstall(context.Background(), plan, "regression"); err != nil {
		t.Fatal(err)
	}
	if installer.ActiveDigest("demo.app") != plan.SourceDigest {
		t.Fatalf("rollback active = %s", installer.ActiveDigest("demo.app"))
	}
	history := installer.History()
	if len(history) < 3 || !strings.Contains(history[2], "rollback") {
		t.Fatalf("history = %#v", history)
	}
}

func TestInvalidDigestAndExtensionIDRejected(t *testing.T) {
	_, priv, verifier := testKeyPair(t)
	svc := NewWithOptions(Options{
		Policy:   OperatorPolicy{AllowedChannels: []string{ChannelStable}, AllowUnsigned: false},
		Verifier: verifier,
	})
	index := Index{
		SchemaVersion: SchemaVersion,
		Entries: []Entry{{
			ExtensionID: "Bad ID", Version: "1.0.0",
			PackageDigest: "not-a-digest", Channel: ChannelStable,
		}},
	}
	index = signedIndex(t, priv, index)
	if err := svc.LoadIndex(index); err == nil {
		t.Fatal("expected invalid entry rejection")
	}
}

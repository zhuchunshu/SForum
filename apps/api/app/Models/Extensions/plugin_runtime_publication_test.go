package extensions

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPluginRuntimeMembersDigestMatchesMigrationVectors(t *testing.T) {
	empty, err := PluginRuntimeMembersDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	if empty != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("empty digest=%s", empty)
	}
	single := PluginRuntimeMember{
		ExtensionID: "fixture.plugin", ExtensionVersionID: 101,
		ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("b", 64),
	}
	digest, err := PluginRuntimeMembersDigest([]PluginRuntimeMember{single})
	if err != nil {
		t.Fatal(err)
	}
	if digest != "be6d586290d73f3688c04e16db1df46129e5dacf03cb924f0d43de39feb27b90" {
		t.Fatalf("single digest=%s", digest)
	}
}

func TestPluginRuntimeMembersDigestIsOrderedAndDoesNotMutateInput(t *testing.T) {
	left := PluginRuntimeMember{
		ExtensionID: "z.plugin", ExtensionVersionID: 12,
		ExtensionVersion: "2.0.0", PackageDigest: strings.Repeat("c", 64),
	}
	right := PluginRuntimeMember{
		ExtensionID: "a.plugin", ExtensionVersionID: 11,
		ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("b", 64),
	}
	input := []PluginRuntimeMember{left, right}
	forward, err := PluginRuntimeMembersDigest(input)
	if err != nil {
		t.Fatal(err)
	}
	reverse, err := PluginRuntimeMembersDigest([]PluginRuntimeMember{right, left})
	if err != nil {
		t.Fatal(err)
	}
	if forward != reverse {
		t.Fatalf("unordered digests differ: %s != %s", forward, reverse)
	}
	if input[0] != left || input[1] != right {
		t.Fatalf("digest mutated caller input: %#v", input)
	}
}

func TestPluginRuntimeMembersDigestRejectsInvalidAndDuplicateMembers(t *testing.T) {
	valid := PluginRuntimeMember{
		ExtensionID: "valid.plugin", ExtensionVersionID: 1,
		ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("a", 64),
	}
	tests := []struct {
		name    string
		members []PluginRuntimeMember
	}{
		{name: "duplicate id", members: []PluginRuntimeMember{valid, valid}},
		{name: "empty id", members: []PluginRuntimeMember{{
			ExtensionVersionID: 1, ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("a", 64),
		}}},
		{name: "untrimmed version", members: []PluginRuntimeMember{{
			ExtensionID: "valid.plugin", ExtensionVersionID: 1,
			ExtensionVersion: " 1.0.0", PackageDigest: strings.Repeat("a", 64),
		}}},
		{name: "uppercase digest", members: []PluginRuntimeMember{{
			ExtensionID: "valid.plugin", ExtensionVersionID: 1,
			ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("A", 64),
		}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := PluginRuntimeMembersDigest(test.members); !errors.Is(err, ErrPluginRuntimePublicationConflict) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestPluginRuntimeAppliedMembersRequireExactDesiredSet(t *testing.T) {
	desired := []PluginRuntimeMember{
		{
			ExtensionID: "b.plugin", ExtensionVersionID: 2,
			ExtensionVersion: "2.0.0", PackageDigest: strings.Repeat("b", 64),
		},
		{
			ExtensionID: "a.plugin", ExtensionVersionID: 1,
			ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("a", 64),
		},
	}
	applied := []PluginRuntimeAppliedMember{
		{PluginRuntimeMember: desired[0], RuntimeInstanceID: "runtime-b"},
		{PluginRuntimeMember: desired[1], RuntimeInstanceID: "runtime-a"},
	}
	canonical, digest, err := canonicalPluginRuntimeAppliedMembers(desired, applied)
	if err != nil {
		t.Fatal(err)
	}
	wantDigest, _ := PluginRuntimeMembersDigest(desired)
	if digest != wantDigest || canonical[0].ExtensionID != "a.plugin" || canonical[1].ExtensionID != "b.plugin" {
		t.Fatalf("canonical=%#v digest=%s want=%s", canonical, digest, wantDigest)
	}

	drifted := append([]PluginRuntimeAppliedMember(nil), applied...)
	drifted[0].PackageDigest = strings.Repeat("c", 64)
	if _, _, err := canonicalPluginRuntimeAppliedMembers(desired, drifted); !errors.Is(err, ErrPluginRuntimeAckConflict) {
		t.Fatalf("drift error=%v", err)
	}
	missingInstance := append([]PluginRuntimeAppliedMember(nil), applied...)
	missingInstance[0].RuntimeInstanceID = ""
	if _, _, err := canonicalPluginRuntimeAppliedMembers(desired, missingInstance); !errors.Is(err, ErrPluginRuntimeAckConflict) {
		t.Fatalf("runtime identity error=%v", err)
	}
}

func TestPluginRuntimeNodeIdentityAndLeaseValidation(t *testing.T) {
	valid := PluginRuntimeNodeIdentity{NodeID: "node-a", ProcessRole: PluginRuntimeProcessAPI, BootID: "boot-a"}
	if !validPluginRuntimeNodeIdentity(valid) || !validPluginRuntimeNodeLease(time.Minute) {
		t.Fatal("valid identity or lease was rejected")
	}
	for _, invalid := range []PluginRuntimeNodeIdentity{
		{NodeID: "", ProcessRole: PluginRuntimeProcessAPI, BootID: "boot-a"},
		{NodeID: "node-a", ProcessRole: "web", BootID: "boot-a"},
		{NodeID: "node-a", ProcessRole: PluginRuntimeProcessWorker, BootID: " boot-a"},
		{NodeID: strings.Repeat("n", 129), ProcessRole: PluginRuntimeProcessAPI, BootID: "boot-a"},
	} {
		if validPluginRuntimeNodeIdentity(invalid) {
			t.Fatalf("invalid identity accepted: %#v", invalid)
		}
	}
	if validPluginRuntimeNodeLease(0) || validPluginRuntimeNodeLease(maxPluginRuntimeNodeLease+time.Millisecond) {
		t.Fatal("invalid lease was accepted")
	}
}

package extensionmanifest

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestManifestVersionCompatibility(t *testing.T) {
	for _, version := range []int{0, ManifestVersionV1, ManifestVersionV2, ManifestVersionV3} {
		manifest := versionedTestManifest(version)
		if err := Validate(manifest); err != nil {
			t.Fatalf("manifest version %d should validate: %v", version, err)
		}
		want := version
		if want == 0 {
			want = ManifestVersionV1
		}
		if got := EffectiveManifestVersion(manifest); got != want {
			t.Fatalf("effective version = %d, want %d", got, want)
		}
		if got := ManifestContract(manifest); got != fmt.Sprintf("sforum.manifest@%d", want) {
			t.Fatalf("contract = %q", got)
		}
	}
}

func TestManifestVersionRejectsUnsupportedValues(t *testing.T) {
	for _, version := range []int{-1, ManifestVersionLatest + 1} {
		if err := Validate(versionedTestManifest(version)); err == nil {
			t.Fatalf("manifest version %d must be rejected", version)
		}
	}
}

func TestLegacyManifestMarshalDoesNotAddVersion(t *testing.T) {
	body, err := json.Marshal(versionedTestManifest(0))
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatal(err)
	}
	if _, exists := object["manifestVersion"]; exists {
		t.Fatalf("legacy manifest gained manifestVersion: %s", body)
	}
}

func versionedTestManifest(version int) Manifest {
	return Manifest{
		ManifestVersion: version,
		ID:              "version.test",
		Name:            "Version Test",
		Description:     "Manifest version compatibility fixture.",
		URL:             "https://example.com/version-test",
		Author:          ManifestAuthor{Name: "SForum"},
		Version:         "1.0.0",
		Type:            TypePlugin,
		SForumVersion:   "^1.0.0",
	}
}

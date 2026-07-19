package extensionsruntime

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLifecyclePackageSchemaLoaderFreezesCanonicalIdentityAndBytes(t *testing.T) {
	extension := lifecycleExecutableQuerySchemaExtension(t, "schemas/query-items.json", `{"type":"object"}`)
	file := extension.Manifest.PackageFiles[len(extension.Manifest.PackageFiles)-1]
	loader := newLifecyclePackageSchemaLoader(extension)

	byPath, err := loader.Load(file.Path)
	if err != nil {
		t.Fatal(err)
	}
	wireReference := file.ID + "@" + file.Version
	if byPath.ManifestReference != file.Path || byPath.WireReference != wireReference ||
		byPath.Digest != file.Digest || !bytes.Equal(byPath.Body, []byte(`{"type":"object"}`)) {
		t.Fatalf("path Schema = %#v", byPath)
	}
	byPath.Body[0] = 'x'
	if err := os.WriteFile(filepath.Join(extension.PackagePath, file.Path), []byte(`{"type":"array"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	byID, err := loader.Load(wireReference)
	if err != nil {
		t.Fatalf("cached ID@version alias: %v", err)
	}
	if byID.ManifestReference != wireReference || byID.WireReference != wireReference ||
		!bytes.Equal(byID.Body, []byte(`{"type":"object"}`)) {
		t.Fatalf("cached alias Schema = %#v", byID)
	}

	// A new publication loader must observe and reject package drift.
	if _, err := newLifecyclePackageSchemaLoader(extension).Load(file.Path); err == nil {
		t.Fatal("new loader accepted drifted package bytes")
	}
}

func TestLifecycleExecutableQueryKeepsPathSchemaWithoutWireVersion(t *testing.T) {
	extension := lifecycleExecutableQuerySchemaExtension(t, "schemas/query-items.json", `{"type":"object"}`)
	fileIndex := len(extension.Manifest.PackageFiles) - 1
	extension.Manifest.PackageFiles[fileIndex].Version = ""
	loader := newLifecyclePackageSchemaLoader(extension)
	loaded, err := loader.Load(extension.Manifest.Queries[0].ResultSchema)
	if err != nil || loaded.WireReference != "" {
		t.Fatalf("path-only Schema = %#v, %v", loaded, err)
	}
	publication, err := buildLifecycleQueryPublication(extension, lifecycleRegistryBinding(extension, "query-path-without-version"))
	if err != nil || publication == nil || publication.Queries[0].ResultSchemaDigest == "" {
		t.Fatalf("path-only Query publication = %#v, %v", publication, err)
	}
}

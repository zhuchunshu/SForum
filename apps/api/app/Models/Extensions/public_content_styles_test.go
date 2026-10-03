package extensions

import (
	"errors"
	"testing"
)

func TestPublicContentStylesReturnsOnlyTrustedExactSurfaceCSS(t *testing.T) {
	extension := publicAssetOnlyFixture(t, "demo.content-style")
	extension.Manifest.Assets[0].Scope = []string{PublicContentStyleScope}
	extension.Manifest.Assets[0].CSP = []string{"style-src 'self'"}

	reader := &fakeFrontendExtensionReader{item: extension}
	trust := NewExecutableTrustService(reader, &memoryExecutableTrustStore{})
	service := newAdmittedPublicFrontendService(reader, trust)
	grantPublicFrontend(t, trust, extension)
	publishTrustedPublicAssets(t, service, extension)

	catalog, err := service.PublicContentStyles(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if catalog.SchemaVersion != PublicContentStylesSchemaV1 || catalog.GraphDigest == "" || len(catalog.Styles) != 1 {
		t.Fatalf("unexpected content style catalog: %#v", catalog)
	}
	style := catalog.Styles[0]
	if style.Handle != extension.Manifest.Assets[0].Handle || style.Type != "style" ||
		style.Loading != "blocking" || style.AssetPath == "" || style.Integrity == "" {
		t.Fatalf("unexpected content style: %#v", style)
	}

	service.WithSafeMode(true)
	if _, err := service.PublicContentStyles(t.Context()); !errors.Is(err, ErrPublicFrontendUnavailable) {
		t.Fatalf("safe mode content styles err=%v", err)
	}
}

func TestPublicContentStylesRejectsExternalCSP(t *testing.T) {
	extension := publicAssetOnlyFixture(t, "demo.content-rejected")
	extension.Manifest.Assets[0].Scope = []string{PublicContentStyleScope}
	extension.Manifest.Assets[0].CSP = []string{"style-src https://cdn.example.com"}
	reader := &fakeFrontendExtensionReader{item: extension}
	trust := NewExecutableTrustService(reader, &memoryExecutableTrustStore{})
	service := newAdmittedPublicFrontendService(reader, trust)
	grantPublicFrontend(t, trust, extension)
	publishTrustedPublicAssets(t, service, extension)
	if _, err := service.PublicContentStyles(t.Context()); !errors.Is(err, ErrPublicFrontendUnavailable) {
		t.Fatalf("content styles err=%v", err)
	}
}

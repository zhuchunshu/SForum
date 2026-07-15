package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	assetregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/AssetRegistry"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

var ErrPublicFrontendUnavailable = errors.New("extensions: public frontend is unavailable")

const (
	publicL2EntrySuffix          = ".l2.entry"
	maxPublicL2Assets            = 256
	maxPublicPackageResourceSize = 8 * 1024 * 1024
)

var publicPackageAssetContentTypes = map[string]string{
	".js":    "application/javascript; charset=utf-8",
	".mjs":   "application/javascript; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".webp":  "image/webp",
	".avif":  "image/avif",
	".ico":   "image/x-icon",
	".woff":  "font/woff",
	".woff2": "font/woff2",
}

func (s *FrontendService) PublicComponent(ctx context.Context, extensionID, componentID string) (PublicFrontendComponent, error) {
	extension, identity, err := s.publicRuntimeExtension(ctx, extensionID)
	if err != nil {
		return PublicFrontendComponent{}, err
	}
	componentID = normalizeID(componentID)
	component, ok := publicManifestComponent(extension.Manifest, componentID)
	if !ok {
		return PublicFrontendComponent{}, ErrPublicFrontendUnavailable
	}
	if s.publicComponents == nil || !s.publicComponents.AdmitPublicComponent(extension, component) {
		return PublicFrontendComponent{}, ErrPublicFrontendUnavailable
	}
	if err := s.refreshPublicAssetSnapshot(ctx); err != nil {
		return PublicFrontendComponent{}, errors.Join(ErrPublicFrontendUnavailable, err)
	}
	entryHandle := publicL2EntryHandle(component)
	plan, err := s.publicAssets.Plan(assetregistry.PlanRequest{Handles: []string{entryHandle}})
	if err != nil {
		return PublicFrontendComponent{}, errors.Join(ErrPublicFrontendUnavailable, err)
	}

	assets := make([]PublicFrontendAssetReference, 0, len(plan))
	var entry PublicFrontendAssetReference
	cspSet := map[string]struct{}{}
	for _, asset := range plan {
		reference := publicAssetReference(asset)
		for _, declaration := range reference.CSP {
			cspSet[declaration] = struct{}{}
		}
		if asset.Handle == entryHandle {
			if !publicAssetMatchesRuntime(asset, extension, identity) {
				return PublicFrontendComponent{}, ErrPublicFrontendUnavailable
			}
			entry = reference
			continue
		}
		assets = append(assets, reference)
	}
	if entry.Handle == "" || entry.Type != "script" || !entry.Module {
		return PublicFrontendComponent{}, ErrPublicFrontendUnavailable
	}
	csp := make([]string, 0, len(cspSet))
	for declaration := range cspSet {
		csp = append(csp, declaration)
	}
	sort.Strings(csp)
	return PublicFrontendComponent{
		SchemaVersion: PublicFrontendSchemaV1, APIVersion: PublicFrontendAPIVersion,
		TrustNotice: PublicFrontendTrustNotice, ExtensionID: extension.ID,
		ExtensionVersion: extension.Version, PackageDigest: extension.PackageDigest,
		ImpactDigest: identity.ImpactDigest, ComponentID: component.ID,
		ContractVersion: component.ContractVersion, Action: component.Action,
		TargetID: component.TargetID, TargetContractVersion: component.TargetContractVersion,
		PropsSchema: component.PropsSchema, ResultSchema: component.ResultSchema,
		Entry: entry, Assets: assets, CSP: csp,
	}, nil
}

func (s *FrontendService) PublicAsset(
	ctx context.Context,
	extensionID, packageDigest, digest, handle string,
) (FrontendAsset, error) {
	extension, identity, err := s.publicRuntimeExtension(ctx, extensionID)
	if err != nil {
		return FrontendAsset{}, err
	}
	packageDigest = normalizedPublicDigest(packageDigest)
	digest = normalizedPublicDigest(digest)
	handle = normalizeID(handle)
	if packageDigest != extension.PackageDigest || handle == "" || digest == "" {
		return FrontendAsset{}, ErrPublicFrontendUnavailable
	}
	if err := s.refreshPublicAssetSnapshot(ctx); err != nil {
		return FrontendAsset{}, errors.Join(ErrPublicFrontendUnavailable, err)
	}
	asset, ok := s.publicAssets.Resolve(handle)
	if !ok || asset.Artifact.ExtensionID != extension.ID ||
		asset.Artifact.ExtensionVersion != extension.Version ||
		asset.Artifact.PackageDigest != extension.PackageDigest ||
		asset.Artifact.ImpactDigest != identity.ImpactDigest || asset.Digest != digest {
		return FrontendAsset{}, ErrPublicFrontendUnavailable
	}
	target, ok := installedFilePath(extension, asset.Path)
	if !ok {
		return FrontendAsset{}, ErrPublicFrontendUnavailable
	}
	info, statErr := os.Lstat(target)
	if statErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return FrontendAsset{}, ErrPublicFrontendUnavailable
	}
	limit := int64(maxPrebuiltAdminModuleBytes)
	contentType := "application/javascript; charset=utf-8"
	if asset.Type == "style" {
		limit = maxPrebuiltAdminCSSBytes
		contentType = "text/css; charset=utf-8"
	}
	if info.Size() > limit || !validPublicAssetExtension(asset.Type, target) {
		return FrontendAsset{}, ErrPublicFrontendUnavailable
	}
	body, readErr := os.ReadFile(target)
	if readErr != nil {
		return FrontendAsset{}, readErr
	}
	actual := sha256.Sum256(body)
	actualDigest := hex.EncodeToString(actual[:])
	if actualDigest != asset.Digest {
		return FrontendAsset{}, errors.Join(ErrPublicFrontendUnavailable, ErrFrontendPackageChanged)
	}
	return FrontendAsset{
		Body: body, ContentType: contentType, ETag: `"` + actualDigest + `"`,
		Digest: actualDigest, Integrity: asset.Integrity, CSP: append([]string(nil), asset.CSP...),
	}, nil
}

// PublicPackageAsset serves package-local resources from their original URL
// directory. Native ESM relative imports, import.meta.url, CSS url(), and
// package-local @import therefore keep browser semantics without a runtime
// source rewrite. Only exact manifest declarations cross this boundary.
func (s *FrontendService) PublicPackageAsset(
	ctx context.Context,
	extensionID, packageDigest, packagePath string,
) (FrontendAsset, error) {
	extension, _, err := s.publicRuntimeExtension(ctx, extensionID)
	if err != nil {
		return FrontendAsset{}, err
	}
	packageDigest = normalizedPublicDigest(packageDigest)
	packagePath = strings.TrimPrefix(strings.TrimSpace(packagePath), "/")
	if packageDigest != extension.PackageDigest || !safePublicPackagePath(packagePath) {
		return FrontendAsset{}, ErrPublicFrontendUnavailable
	}
	file, ok := publicPackageFileByPath(extension.Manifest, packagePath)
	if !ok {
		return FrontendAsset{}, ErrPublicFrontendUnavailable
	}
	contentType, ok := publicPackageFileContentType(file)
	if !ok {
		return FrontendAsset{}, ErrPublicFrontendUnavailable
	}
	return readExactPublicPackageFile(extension, file, contentType)
}

func (s *FrontendService) refreshPublicAssetSnapshot(ctx context.Context) error {
	catalog, ok := s.extensions.(PublicFrontendExtensionCatalog)
	if !ok || ctx == nil {
		return ErrPublicFrontendUnavailable
	}
	items, err := catalog.List(ctx)
	if err != nil {
		return err
	}
	publications := make([]assetregistry.Publication, 0, len(items))
	for _, extension := range items {
		if extension.Status != StatusEnabled || !hasL2Components(extension.Manifest) {
			continue
		}
		identity, identityErr := s.executableTrust.RuntimeIdentity(ctx, extension)
		if errors.Is(identityErr, ErrTrustGrantNotFound) || errors.Is(identityErr, ErrFrontendPackageChanged) {
			// An untrusted, revoked, or byte-drifted owner contributes nothing.
			// Required cross-owner dependencies then fail the complete graph.
			continue
		}
		if identityErr != nil {
			return identityErr
		}
		if identity.ImpactDigest == "" {
			return ErrPublicFrontendUnavailable
		}
		publication, publicationErr := publicAssetPublication(extension, identity)
		if publicationErr != nil {
			return publicationErr
		}
		publications = append(publications, publication)
	}
	_, err = s.publicAssets.ReplaceAll(publications)
	return err
}

func (s *FrontendService) publicRuntimeExtension(
	ctx context.Context,
	extensionID string,
) (Extension, RuntimeTrustIdentity, error) {
	extensionID = normalizeID(extensionID)
	if s == nil || ctx == nil || !s.publicL2 || s.safeMode || !s.v3TrustChallenges ||
		s.extensions == nil || s.executableTrust == nil || s.publicAssets == nil || extensionID == "" {
		if s != nil && s.publicAssets != nil && (s.safeMode || !s.publicL2) {
			_, _ = s.publicAssets.ReplaceAll(nil)
		}
		return Extension{}, RuntimeTrustIdentity{}, ErrPublicFrontendUnavailable
	}
	extension, err := s.extensions.Get(ctx, extensionID)
	if err != nil || extension.Status != StatusEnabled || !hasL2Components(extension.Manifest) {
		_ = s.refreshPublicAssetSnapshot(ctx)
		return Extension{}, RuntimeTrustIdentity{}, ErrPublicFrontendUnavailable
	}
	identity, err := s.executableTrust.RuntimeIdentity(ctx, extension)
	if err != nil || identity.ImpactDigest == "" {
		_ = s.refreshPublicAssetSnapshot(ctx)
		if errors.Is(err, ErrFrontendPackageChanged) {
			return Extension{}, RuntimeTrustIdentity{}, errors.Join(ErrPublicFrontendUnavailable, err)
		}
		return Extension{}, RuntimeTrustIdentity{}, ErrPublicFrontendUnavailable
	}
	return extension, identity, nil
}

func publicManifestComponent(manifest Manifest, componentID string) (ManifestComponent, bool) {
	manifest = extensionmanifest.Normalize(manifest)
	for _, component := range manifest.Components {
		if component.ID == componentID && component.L2Component != "" && component.Action != extensionmanifest.ComponentActionHide {
			return component, true
		}
	}
	return ManifestComponent{}, false
}

func publicAssetPublication(
	extension Extension,
	identity RuntimeTrustIdentity,
) (assetregistry.Publication, error) {
	manifest := extensionmanifest.Normalize(extension.Manifest)
	packageFiles := make(map[string]ManifestPackageFile, len(manifest.PackageFiles))
	for _, file := range manifest.PackageFiles {
		packageFiles[file.ID] = file
	}
	declarations := make([]assetregistry.Declaration, 0, len(manifest.Assets)+len(manifest.Components))
	for _, asset := range manifest.Assets {
		declaration := assetregistry.Declaration{
			Handle: asset.Handle, ContractVersion: asset.ContractVersion, Type: asset.Type,
			Path: asset.Path, Digest: asset.Digest, Dependencies: append([]string(nil), asset.Dependencies...),
			Scope: append([]string(nil), asset.Scope...), Module: asset.Module,
			Loading: asset.Loading, Integrity: asset.Integrity, CSP: append([]string(nil), asset.CSP...),
		}
		declarations = append(declarations, declaration)
	}
	for _, component := range manifest.Components {
		if component.L2Component == "" || component.Action == extensionmanifest.ComponentActionHide {
			continue
		}
		entryFile, ok := packageFiles[component.L2Component]
		if !ok || entryFile.Kind != "frontend" || !validPublicAssetExtension("script", entryFile.Path) {
			return assetregistry.Publication{}, ErrPublicFrontendUnavailable
		}
		scopes := []string{component.ID}
		if component.TargetID != "" {
			scopes = append(scopes, component.TargetID)
		}
		entryDependencies := make([]string, 0, len(manifest.Assets))
		for _, asset := range manifest.Assets {
			if len(asset.Scope) == 0 || stringSlicesIntersect(asset.Scope, scopes) {
				entryDependencies = append(entryDependencies, asset.Handle)
			}
		}
		declarations = append(declarations, assetregistry.Declaration{
			Handle: publicL2EntryHandle(component), ContractVersion: component.ContractVersion, Type: "script",
			Path: entryFile.Path, Digest: entryFile.Digest, Dependencies: entryDependencies,
			Scope: scopes, Module: true, Loading: "lazy",
		})
	}
	if len(declarations) == 0 || len(declarations) > maxPublicL2Assets {
		return assetregistry.Publication{}, ErrPublicFrontendUnavailable
	}
	return assetregistry.Publication{Artifact: assetregistry.Artifact{
		ExtensionID: extension.ID, ExtensionVersion: extension.Version,
		PackageDigest: extension.PackageDigest, ImpactDigest: identity.ImpactDigest,
	}, Assets: declarations}, nil
}

func publicL2EntryHandle(component ManifestComponent) string {
	return component.ID + publicL2EntrySuffix
}

func publicAssetReference(asset assetregistry.Asset) PublicFrontendAssetReference {
	assetPath := fmt.Sprintf(
		"/extensions/runtime/%s/packages/%s/%s",
		url.PathEscape(asset.Artifact.ExtensionID), url.PathEscape(asset.Artifact.PackageDigest),
		escapePublicPackagePath(asset.Path),
	)
	return PublicFrontendAssetReference{
		Handle: asset.Handle, ContractVersion: asset.ContractVersion,
		ExtensionID: asset.Artifact.ExtensionID, PackageDigest: asset.Artifact.PackageDigest,
		ImpactDigest: asset.Artifact.ImpactDigest, Type: asset.Type, Digest: asset.Digest,
		Integrity: asset.Integrity, Dependencies: append([]string(nil), asset.Dependencies...),
		Scope: append([]string(nil), asset.Scope...), Module: asset.Module, Loading: asset.Loading,
		CSP: append([]string(nil), asset.CSP...), AssetPath: assetPath,
	}
}

func publicAssetMatchesRuntime(
	asset assetregistry.Asset,
	extension Extension,
	identity RuntimeTrustIdentity,
) bool {
	return asset.Artifact.ExtensionID == extension.ID &&
		asset.Artifact.ExtensionVersion == extension.Version &&
		asset.Artifact.PackageDigest == extension.PackageDigest &&
		asset.Artifact.ImpactDigest == identity.ImpactDigest
}

func publicPackageFileByPath(manifest Manifest, packagePath string) (ManifestPackageFile, bool) {
	manifest = extensionmanifest.Normalize(manifest)
	for _, file := range manifest.PackageFiles {
		if file.Path == packagePath {
			return file, true
		}
	}
	return ManifestPackageFile{}, false
}

func publicPackageFileContentType(file ManifestPackageFile) (string, bool) {
	if file.Kind != "frontend" && file.Kind != "asset" {
		return "", false
	}
	extension := strings.ToLower(path.Ext(file.Path))
	contentType, ok := publicPackageAssetContentTypes[extension]
	if !ok {
		return "", false
	}
	if file.Kind == "frontend" && extension != ".js" && extension != ".mjs" {
		return "", false
	}
	return contentType, true
}

func readExactPublicPackageFile(
	extension Extension,
	file ManifestPackageFile,
	contentType string,
) (FrontendAsset, error) {
	target, ok := installedFilePath(extension, file.Path)
	if !ok {
		return FrontendAsset{}, ErrPublicFrontendUnavailable
	}
	info, err := os.Lstat(target)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
		info.Size() <= 0 || info.Size() > maxPublicPackageResourceSize {
		return FrontendAsset{}, ErrPublicFrontendUnavailable
	}
	body, err := os.ReadFile(target)
	if err != nil {
		return FrontendAsset{}, err
	}
	digest := sha256.Sum256(body)
	actualDigest := hex.EncodeToString(digest[:])
	if !strings.EqualFold(actualDigest, normalizedPublicDigest(file.Digest)) {
		return FrontendAsset{}, errors.Join(ErrPublicFrontendUnavailable, ErrFrontendPackageChanged)
	}
	integrity := "sha256-" + base64.StdEncoding.EncodeToString(digest[:])
	return FrontendAsset{
		Body: body, ContentType: contentType, ETag: `"` + actualDigest + `"`,
		Digest: actualDigest, Integrity: integrity,
	}, nil
}

func safePublicPackagePath(value string) bool {
	return value != "" && len(value) <= 512 && !strings.Contains(value, "\\") &&
		!strings.HasPrefix(value, "/") && path.Clean(value) == value &&
		value != "." && value != ".." && !strings.HasPrefix(value, "../")
}

func escapePublicPackagePath(value string) string {
	parts := strings.Split(value, "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}

func stringSlicesIntersect(left, right []string) bool {
	values := make(map[string]struct{}, len(left))
	for _, value := range left {
		values[normalizeID(value)] = struct{}{}
	}
	for _, value := range right {
		if _, ok := values[normalizeID(value)]; ok {
			return true
		}
	}
	return false
}

func validPublicAssetExtension(assetType, name string) bool {
	extension := strings.ToLower(filepath.Ext(name))
	if assetType == "style" {
		return extension == ".css"
	}
	return extension == ".js" || extension == ".mjs"
}

func normalizedPublicDigest(value string) string {
	value = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "sha256:")
	if len(value) != 64 {
		return ""
	}
	if _, err := hex.DecodeString(value); err != nil {
		return ""
	}
	return value
}

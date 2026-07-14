package pages

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	themecompiler "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeCompiler"
)

var (
	ErrThemeRuntimeInvalid  = errors.New("pages: invalid theme runtime snapshot")
	ErrThemeRuntimeConflict = errors.New("pages: theme runtime exact artifact conflict")
	ErrThemeRuntimeMissing  = errors.New("pages: theme runtime snapshot is missing")
)

type ThemeRuntimeBuildInput struct {
	Artifact      RuntimeArtifact
	PackageRoot   string
	Contributions []PageContribution
	SiteName      string
	Locales       []string
}

type ThemeRuntimeProviderBinding struct {
	PageID          string `json:"pageId"`
	ContributionID  string `json:"contributionId"`
	Template        string `json:"template"`
	ContractVersion string `json:"contractVersion"`
}

type ThemeRenderedPage struct {
	HTMLSegments []string                         `json:"htmlSegments"`
	Islands      []themecompiler.IslandDescriptor `json:"islands,omitempty"`
	SEO          themecompiler.PageSEOView        `json:"seo"`
}

type ThemeRuntimeSnapshot struct {
	artifact   RuntimeArtifact
	compiled   *themecompiler.Snapshot
	providers  map[string]ThemeRuntimeProviderBinding
	assets     ActiveSkinPublic
	locales    []string
	contracts  map[string]string
	islandTags map[string]string
}

func BuildThemeRuntimeSnapshot(input ThemeRuntimeBuildInput) (*ThemeRuntimeSnapshot, error) {
	if !validThemeRuntimeArtifact(input.Artifact) || strings.TrimSpace(input.PackageRoot) == "" {
		return nil, ErrThemeRuntimeInvalid
	}
	root, err := filepath.Abs(strings.TrimSpace(input.PackageRoot))
	if err != nil {
		return nil, fmt.Errorf("%w: package root: %v", ErrThemeRuntimeInvalid, err)
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("%w: package root: %v", ErrThemeRuntimeInvalid, err)
	}
	providers := make(map[string]ThemeRuntimeProviderBinding)
	pageBindings := make(map[string]themecompiler.PageTemplateBinding)
	selectedTemplates := make(map[string]struct{})
	contracts := make(map[string]string)
	for _, contribution := range input.Contributions {
		if contribution.Action != ActionReplace || strings.TrimSpace(contribution.Template) == "" {
			continue
		}
		page, ok := Find(contribution.Target)
		if !ok || page.ContractVersion != contribution.Contract || contribution.ExtensionID != input.Artifact.ExtensionID ||
			contribution.Version != input.Artifact.ExtensionVersion || contribution.PackageDigest != input.Artifact.PackageDigest {
			return nil, ErrThemeRuntimeConflict
		}
		templateName := filepath.ToSlash(strings.TrimSpace(contribution.Template))
		if previous, exists := providers[page.ID]; exists && previous.ContributionID != contribution.ID {
			return nil, ErrThemeRuntimeConflict
		}
		if _, exists := pageBindings[templateName]; exists {
			return nil, ErrThemeRuntimeConflict
		}
		providers[page.ID] = ThemeRuntimeProviderBinding{
			PageID: page.ID, ContributionID: contribution.ID, Template: templateName,
			ContractVersion: page.ContractVersion,
		}
		pageBindings[templateName] = themecompiler.PageTemplateBinding{PageID: page.ID, SchemaVersion: page.ContractVersion}
		selectedTemplates[templateName] = struct{}{}
		contracts[page.ID] = page.ContractVersion
	}
	if len(providers) == 0 {
		return nil, ErrThemeRuntimeMissing
	}
	assets, err := SkinFromPackage(
		input.Artifact.ExtensionID, input.Artifact.ExtensionVersion, input.Artifact.PackageDigest, realRoot,
	)
	if err != nil {
		return nil, err
	}
	routes := make(map[string]string)
	for _, page := range Catalog() {
		routes[page.ID] = page.PathPattern
	}
	islands := productionThemeIslandBindings()
	bindingRevision, err := themeBindingRevision(input.Artifact, providers, assets, input.Locales, contracts, islands)
	if err != nil {
		return nil, err
	}
	compiled, err := themecompiler.NewCompiler(themecompiler.Limits{}).CompileFS(
		selectedThemeFS{FS: os.DirFS(realRoot), selected: selectedTemplates},
		input.Artifact.PackageDigest,
		themecompiler.Bindings{
			BindingRevision: bindingRevision, SiteName: strings.TrimSpace(input.SiteName),
			Assets: themeAssetBindings(assets), Routes: routes, PageViewModels: pageBindings, Islands: islands,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: compile exact theme: %v", ErrThemeRuntimeInvalid, err)
	}
	islandTags := make(map[string]string, len(islands))
	for tag, binding := range islands {
		islandTags[binding.ComponentID] = tag
	}
	return &ThemeRuntimeSnapshot{
		artifact: input.Artifact, compiled: compiled, providers: providers, assets: assets,
		locales: normalizedLocales(input.Locales), contracts: contracts, islandTags: islandTags,
	}, nil
}

func (s *ThemeRuntimeSnapshot) Artifact() RuntimeArtifact {
	if s == nil {
		return RuntimeArtifact{}
	}
	return s.artifact
}

func (s *ThemeRuntimeSnapshot) RuntimeKey() string {
	if s == nil || s.compiled == nil {
		return ""
	}
	return s.compiled.RuntimeKey()
}

func (s *ThemeRuntimeSnapshot) Covers(pageID, contributionID string) bool {
	if s == nil {
		return false
	}
	binding, ok := s.providers[pageID]
	return ok && binding.ContributionID == contributionID
}

func (s *ThemeRuntimeSnapshot) Render(
	ctx context.Context,
	request CorePageViewModelRequest,
	contributionID string,
) (ThemeRenderedPage, error) {
	if s == nil || s.compiled == nil || ctx == nil {
		return ThemeRenderedPage{}, ErrThemeRuntimeMissing
	}
	binding, ok := s.providers[request.PageID]
	if !ok || binding.ContributionID != contributionID {
		return ThemeRenderedPage{}, ErrThemeRuntimeMissing
	}
	value, err := BuildCorePageViewModel(request)
	if err != nil {
		return ThemeRenderedPage{}, err
	}
	bound, err := themecompiler.CorePageViewModelRegistry().Bind(
		binding.PageID, binding.ContractVersion, s.artifact.PackageDigest, value,
	)
	if err != nil {
		return ThemeRenderedPage{}, err
	}
	output, err := s.compiled.Render(ctx, binding.Template, bound)
	if err != nil {
		return ThemeRenderedPage{}, err
	}
	segments := output.HTMLSegments()
	result := ThemeRenderedPage{HTMLSegments: make([]string, len(segments)), Islands: output.Islands(), SEO: output.SEO()}
	for index := range segments {
		result.HTMLSegments[index] = segments[index].String()
	}
	return result, nil
}

func (s *ThemeRuntimeSnapshot) LegacyHTML(output ThemeRenderedPage) string {
	if s == nil {
		return ""
	}
	value := strings.Join(output.HTMLSegments, "")
	for _, island := range output.Islands {
		tag := s.islandTags[island.ComponentID]
		if tag == "" {
			return ""
		}
		placeholder := `<template data-sforum-island="` + island.ID + `"></template>`
		value = strings.Replace(value, placeholder, "<"+tag+"></"+tag+">", 1)
	}
	return value
}

type ThemeRuntimeRegistry struct {
	mu        sync.RWMutex
	revision  uint64
	active    RuntimeArtifact
	snapshots map[RuntimeArtifact]*ThemeRuntimeSnapshot
}

func NewThemeRuntimeRegistry() *ThemeRuntimeRegistry {
	return &ThemeRuntimeRegistry{snapshots: make(map[RuntimeArtifact]*ThemeRuntimeSnapshot)}
}

func (r *ThemeRuntimeRegistry) Publish(snapshot *ThemeRuntimeSnapshot) uint64 {
	if r == nil || snapshot == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureSnapshots()
	r.snapshots[snapshot.artifact] = snapshot
	r.active = snapshot.artifact
	r.revision++
	return r.revision
}

// Stage makes a target artifact resolvable before the Page Registry switches.
// It does not change which snapshot is reported as active.
func (r *ThemeRuntimeRegistry) Stage(snapshot *ThemeRuntimeSnapshot) (uint64, bool, error) {
	if r == nil || snapshot == nil {
		return 0, false, ErrThemeRuntimeMissing
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureSnapshots()
	if current, exists := r.snapshots[snapshot.artifact]; exists {
		if current.RuntimeKey() != snapshot.RuntimeKey() {
			return r.revision, false, ErrThemeRuntimeConflict
		}
		return r.revision, false, nil
	}
	r.snapshots[snapshot.artifact] = snapshot
	r.revision++
	return r.revision, true, nil
}

func (r *ThemeRuntimeRegistry) ActivateExact(artifact RuntimeArtifact) (uint64, error) {
	if r == nil {
		return 0, ErrThemeRuntimeMissing
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.snapshots[artifact]; !exists {
		return r.revision, ErrThemeRuntimeMissing
	}
	if r.active == artifact {
		return r.revision, nil
	}
	r.active = artifact
	r.revision++
	return r.revision, nil
}

func (r *ThemeRuntimeRegistry) RemoveExact(artifact RuntimeArtifact) (uint64, error) {
	if r == nil {
		return 0, ErrThemeRuntimeMissing
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	snapshot, exists := r.snapshots[artifact]
	if !exists {
		if active := r.snapshots[r.active]; active != nil && active.artifact.ExtensionID == artifact.ExtensionID {
			return r.revision, ErrThemeRuntimeConflict
		}
		return r.revision, nil
	}
	delete(r.snapshots, snapshot.artifact)
	if r.active == artifact {
		r.active = RuntimeArtifact{}
	}
	r.revision++
	return r.revision, nil
}

func (r *ThemeRuntimeRegistry) ClearExtension(extensionID string) uint64 {
	if r == nil || strings.TrimSpace(extensionID) == "" {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	changed := false
	for artifact := range r.snapshots {
		if artifact.ExtensionID == extensionID {
			delete(r.snapshots, artifact)
			changed = true
		}
	}
	if r.active.ExtensionID == extensionID {
		r.active = RuntimeArtifact{}
		changed = true
	}
	if !changed {
		return r.revision
	}
	r.revision++
	return r.revision
}

func (r *ThemeRuntimeRegistry) Resolve(artifact RuntimeArtifact, pageID, contributionID string) (*ThemeRuntimeSnapshot, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	snapshot := r.snapshots[artifact]
	if snapshot == nil || !snapshot.Covers(pageID, contributionID) {
		return nil, false
	}
	return snapshot, true
}

// Claims reports whether a compiled snapshot owns the provider identity even
// when the caller supplied a stale artifact. Controllers use it to fail closed
// instead of falling back to request-time package I/O.
func (r *ThemeRuntimeRegistry) Claims(extensionID, pageID, contributionID string) bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for artifact, snapshot := range r.snapshots {
		if artifact.ExtensionID == extensionID && snapshot.Covers(pageID, contributionID) {
			return true
		}
	}
	return false
}

func (r *ThemeRuntimeRegistry) Active() (*ThemeRuntimeSnapshot, uint64, bool) {
	if r == nil {
		return nil, 0, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	snapshot := r.snapshots[r.active]
	return snapshot, r.revision, snapshot != nil
}

func (r *ThemeRuntimeRegistry) ensureSnapshots() {
	if r.snapshots == nil {
		r.snapshots = make(map[RuntimeArtifact]*ThemeRuntimeSnapshot)
	}
}

func validThemeRuntimeArtifact(artifact RuntimeArtifact) bool {
	return strings.TrimSpace(artifact.ExtensionID) != "" && strings.TrimSpace(artifact.ExtensionVersion) != "" &&
		len(artifact.PackageDigest) == 64 && artifact.PackageDigest == strings.ToLower(artifact.PackageDigest)
}

func themeBindingRevision(
	artifact RuntimeArtifact,
	providers map[string]ThemeRuntimeProviderBinding,
	assets ActiveSkinPublic,
	locales []string,
	contracts map[string]string,
	islands map[string]themecompiler.IslandBinding,
) (string, error) {
	document := struct {
		Schema    string                                 `json:"schema"`
		Artifact  RuntimeArtifact                        `json:"artifact"`
		Providers map[string]ThemeRuntimeProviderBinding `json:"providers"`
		Assets    ActiveSkinPublic                       `json:"assets"`
		Locales   []string                               `json:"locales"`
		Contracts map[string]string                      `json:"contracts"`
		Islands   map[string]themecompiler.IslandBinding `json:"islands"`
	}{"sforum.theme-runtime-binding@1", artifact, providers, assets, normalizedLocales(locales), contracts, islands}
	raw, err := json.Marshal(document)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func normalizedLocales(input []string) []string {
	seen := make(map[string]struct{}, len(input))
	result := make([]string, 0, len(input))
	for _, value := range input {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func themeAssetBindings(skin ActiveSkinPublic) map[string]string {
	result := make(map[string]string, len(skin.CSS)+1)
	for index, value := range skin.CSS {
		result[fmt.Sprintf("theme.css.%d", index+1)] = value
	}
	if skin.Tokens != "" {
		result["theme.tokens"] = skin.Tokens
	}
	return result
}

func productionThemeIslandBindings() map[string]themecompiler.IslandBinding {
	return map[string]themecompiler.IslandBinding{
		"sf-home-page":       {ComponentID: "forum.component.home_page"},
		"sf-navbar":          {ComponentID: "navigation.component.navbar"},
		"sf-footer":          {ComponentID: "navigation.component.footer"},
		"sf-home-navigation": {ComponentID: "navigation.component.home"},
	}
}

type selectedThemeFS struct {
	fs.FS
	selected map[string]struct{}
}

func (s selectedThemeFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(s.FS, name)
	if err != nil {
		return nil, err
	}
	if name != "templates" && !strings.HasPrefix(name, "templates/") {
		return entries, nil
	}
	result := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		child := strings.TrimPrefix(filepath.ToSlash(filepath.Join(name, entry.Name())), "./")
		if _, selected := s.selected[child]; selected || entry.IsDir() && selectedThemePrefix(s.selected, child+"/") {
			result = append(result, entry)
		}
	}
	return result, nil
}

func selectedThemePrefix(selected map[string]struct{}, prefix string) bool {
	for name := range selected {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

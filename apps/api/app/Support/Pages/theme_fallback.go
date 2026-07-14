package pages

import (
	"context"
	"errors"
	"fmt"
	"html"
	"path"
	"strings"

	themecompiler "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeCompiler"
)

const (
	ThemeRenderSourceActiveOverride = "active_theme_override"
	ThemeRenderSourcePlugin         = "plugin_template"
	ThemeRenderSourceActiveTheme    = "active_theme"
	ThemeRenderSourceDefaultTheme   = "default_theme"
	ThemeRenderSourceEmergency      = "host_emergency"
)

type themeRenderCandidate struct {
	source   string
	snapshot *ThemeRuntimeSnapshot
	binding  ThemeRuntimeProviderBinding
}

type themeRenderPlan struct {
	pageID         string
	contributionID string
	candidates     []themeRenderCandidate
}

func (s *ThemeRuntimeSnapshot) renderPlan(
	ctx context.Context,
	request CorePageViewModelRequest,
	contributionID string,
) (ThemeRenderedPage, error) {
	if s.plan == nil || request.PageID != s.plan.pageID || contributionID != s.plan.contributionID {
		return ThemeRenderedPage{}, ErrThemeRuntimeMissing
	}
	value, err := BuildCorePageViewModel(request)
	if err != nil {
		return ThemeRenderedPage{}, err
	}
	attempts := make([]ThemeRenderAttempt, 0, len(s.plan.candidates)+1)
	failed := false
	for _, candidate := range s.plan.candidates {
		if candidate.snapshot == nil || candidate.snapshot.compiled == nil {
			continue
		}
		attempt := ThemeRenderAttempt{
			Source: candidate.source, ExtensionID: candidate.snapshot.artifact.ExtensionID,
			PackageDigest: candidate.snapshot.artifact.PackageDigest, Template: candidate.binding.Template,
		}
		bound, bindErr := themecompiler.CorePageViewModelRegistry().Bind(
			candidate.binding.PageID, candidate.binding.ContractVersion,
			candidate.snapshot.artifact.PackageDigest, value,
		)
		if bindErr != nil {
			attempt.Outcome = "failed"
			attempt.FailureCode = themeRenderFailureCode(bindErr)
			attempts = append(attempts, attempt)
			failed = true
			continue
		}
		output, renderErr := candidate.snapshot.compiled.Render(ctx, candidate.binding.Template, bound)
		if renderErr != nil {
			if errors.Is(renderErr, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
				return ThemeRenderedPage{}, context.Canceled
			}
			attempt.Outcome = "failed"
			attempt.FailureCode = themeRenderFailureCode(renderErr)
			attempts = append(attempts, attempt)
			failed = true
			continue
		}
		attempt.Outcome = "rendered"
		attempts = append(attempts, attempt)
		segments := output.HTMLSegments()
		result := ThemeRenderedPage{
			HTMLSegments: make([]string, len(segments)), Islands: output.Islands(), SEO: output.SEO(),
			Source: candidate.source, Fallback: failed, Attempts: attempts,
		}
		for index := range segments {
			result.HTMLSegments[index] = segments[index].String()
		}
		return result, nil
	}
	return emergencyThemeOutput(request, attempts), nil
}

func emergencyThemeOutput(request CorePageViewModelRequest, attempts []ThemeRenderAttempt) ThemeRenderedPage {
	title := strings.TrimSpace(request.SEO.Title)
	if title == "" {
		title = request.PageID
	}
	attempts = append(attempts, ThemeRenderAttempt{Source: ThemeRenderSourceEmergency, Outcome: "rendered"})
	return ThemeRenderedPage{
		HTMLSegments: []string{`<main data-sforum-emergency="true"><h1>` + html.EscapeString(title) + `</h1></main>`},
		SEO:          request.SEO, Source: ThemeRenderSourceEmergency, Fallback: true, Attempts: attempts,
	}
}

func themeRenderFailureCode(err error) string {
	switch {
	case errors.Is(err, themecompiler.ErrRenderTimeout):
		return "render_timeout"
	case errors.Is(err, themecompiler.ErrOutputLimit):
		return "output_limit"
	case errors.Is(err, themecompiler.ErrMissingValue):
		return "missing_value"
	case errors.Is(err, themecompiler.ErrViewModelSchema):
		return "view_model_contract"
	case errors.Is(err, themecompiler.ErrRequiredIsland), errors.Is(err, themecompiler.ErrInvalidIsland):
		return "island_contract"
	case errors.Is(err, themecompiler.ErrExecution):
		return "execution"
	default:
		return "render_failed"
	}
}

func pageForTemplateContract(contract string) (PageDefinition, bool) {
	contract = strings.TrimSpace(contract)
	for _, page := range Catalog() {
		if page.ContractVersion == contract {
			return page, true
		}
	}
	return PageDefinition{}, false
}

func validPluginOverridePath(name, targetID string) bool {
	name = path.Clean(strings.TrimSpace(name))
	if !strings.HasPrefix(name, "templates/plugins/") || !strings.HasSuffix(name, ".html") {
		return false
	}
	parts := strings.Split(name, "/")
	if len(parts) < 4 || parts[2] == "" || strings.Contains(parts[2], "..") {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(targetID), parts[2]+".")
}

func runtimeRenderKey(artifact RuntimeArtifact, pageID, contributionID string) string {
	return fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s\x00%s", artifact.ExtensionID, artifact.ExtensionVersion,
		artifact.PackageDigest, artifact.RuntimeInstanceID, pageID, contributionID)
}

func (r *ThemeRuntimeRegistry) rebuildRenderersLocked() {
	r.ensureSnapshots()
	next := make(map[string]*ThemeRuntimeSnapshot)
	active := r.snapshots[r.active]
	defaultTheme := r.snapshots[r.defaultArtifact]
	for artifact, owner := range r.snapshots {
		if owner == nil {
			continue
		}
		for pageID, provider := range owner.providers {
			plan := &themeRenderPlan{pageID: pageID, contributionID: provider.ContributionID}
			if owner.kind == RuntimeTemplatePlugin {
				if active != nil && provider.TemplateID != "" {
					if override, ok := active.overrides[provider.TemplateID]; ok {
						plan.addCandidate(ThemeRenderSourceActiveOverride, active, override)
					}
				}
				plan.addCandidate(ThemeRenderSourcePlugin, owner, provider)
				if activeProvider, ok := providerForContract(active, pageID, provider.ContractVersion); ok {
					plan.addCandidate(ThemeRenderSourceActiveTheme, active, activeProvider)
				}
				if defaultProvider, ok := providerForContract(defaultTheme, pageID, provider.ContractVersion); ok {
					plan.addCandidate(ThemeRenderSourceDefaultTheme, defaultTheme, defaultProvider)
				}
			} else {
				source := ThemeRenderSourceActiveTheme
				if artifact == r.defaultArtifact && artifact != r.active {
					source = ThemeRenderSourceDefaultTheme
				}
				plan.addCandidate(source, owner, provider)
				if defaultProvider, ok := providerForContract(defaultTheme, pageID, provider.ContractVersion); ok {
					plan.addCandidate(ThemeRenderSourceDefaultTheme, defaultTheme, defaultProvider)
				}
			}
			next[runtimeRenderKey(artifact, pageID, provider.ContributionID)] = runtimeSnapshotWithPlan(owner, plan, active)
		}
	}
	r.renderers = next
}

func (p *themeRenderPlan) addCandidate(source string, snapshot *ThemeRuntimeSnapshot, binding ThemeRuntimeProviderBinding) {
	if p == nil || snapshot == nil || binding.PageID != p.pageID {
		return
	}
	for _, current := range p.candidates {
		if current.snapshot.artifact == snapshot.artifact && current.binding.Template == binding.Template {
			return
		}
	}
	p.candidates = append(p.candidates, themeRenderCandidate{source: source, snapshot: snapshot, binding: binding})
}

func providerForContract(snapshot *ThemeRuntimeSnapshot, pageID, contract string) (ThemeRuntimeProviderBinding, bool) {
	if snapshot == nil {
		return ThemeRuntimeProviderBinding{}, false
	}
	binding, ok := snapshot.providers[pageID]
	return binding, ok && binding.ContractVersion == contract
}

func runtimeSnapshotWithPlan(owner *ThemeRuntimeSnapshot, plan *themeRenderPlan, active *ThemeRuntimeSnapshot) *ThemeRuntimeSnapshot {
	islandTags := owner.islandTags
	if active != nil {
		islandTags = active.islandTags
	}
	return &ThemeRuntimeSnapshot{
		artifact: owner.artifact, compiled: owner.compiled, providers: owner.providers,
		assets: owner.assets, locales: owner.locales, contracts: owner.contracts,
		islandTags: islandTags, kind: owner.kind, overrides: owner.overrides, plan: plan,
	}
}

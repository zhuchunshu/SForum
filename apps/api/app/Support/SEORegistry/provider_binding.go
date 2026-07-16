package seoregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type staticProviderResolver struct {
	bindings map[providerBindingIdentity]ProviderBinding
}

func newStaticProviderResolver(bindings []ProviderBinding) (ProviderResolver, error) {
	result := &staticProviderResolver{bindings: make(map[providerBindingIdentity]ProviderBinding, len(bindings))}
	for _, raw := range bindings {
		binding, err := normalizeProviderBinding(raw)
		if err != nil {
			return nil, err
		}
		key := providerBindingKey(binding.ContributionID, binding.Artifact)
		if _, duplicate := result.bindings[key]; duplicate {
			return nil, fmt.Errorf("%w: duplicate provider binding %s", ErrExecutionInvalid, binding.ContributionID)
		}
		result.bindings[key] = binding
	}
	return result, nil
}

func (r *staticProviderResolver) ResolveSEOProvider(_ context.Context, contribution Contribution) (ProviderBinding, error) {
	if r == nil {
		return ProviderBinding{}, ErrProviderUnavailable
	}
	binding, found := r.bindings[providerBindingKey(contribution.ID, contribution.Artifact)]
	if !found || !bindingMatchesContribution(binding, contribution) {
		return ProviderBinding{}, ErrProviderUnavailable
	}
	return binding, nil
}

func normalizeProviderBinding(input ProviderBinding) (ProviderBinding, error) {
	input.ContributionID = strings.ToLower(strings.TrimSpace(input.ContributionID))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.Handler = strings.ToLower(strings.TrimSpace(input.Handler))
	input.ProviderDigest = normalizeDigest(input.ProviderDigest)
	artifact, err := normalizeArtifact(input.Artifact)
	input.Artifact = artifact
	if input.ProviderDigest == "" && err == nil {
		input.ProviderDigest = defaultProviderDigest(input)
	}
	if err != nil || !idPattern.MatchString(input.ContributionID) || !contractPattern.MatchString(input.ContractVersion) ||
		!idPattern.MatchString(input.Handler) || !digestPattern.MatchString(input.ProviderDigest) || input.Provider == nil {
		return ProviderBinding{}, ErrExecutionInvalid
	}
	return input, nil
}

func defaultProviderDigest(binding ProviderBinding) string {
	material := SchemaVersion + "\x00provider\x00" + binding.ContributionID + "\x00" + binding.ContractVersion +
		"\x00" + binding.Handler + "\x00" + binding.Artifact.ExtensionID + "\x00" + binding.Artifact.ExtensionVersion +
		"\x00" + binding.Artifact.PackageDigest + "\x00" + binding.Artifact.ImpactDigest + "\x00" +
		fmt.Sprint(binding.Artifact.VersionID) + "\x00" + binding.Artifact.RuntimeInstanceID
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:])
}

func bindingMatchesContribution(binding ProviderBinding, contribution Contribution) bool {
	return binding.ContributionID == contribution.ID && binding.ContractVersion == contribution.ContractVersion &&
		binding.Handler == contribution.Handler && binding.Artifact == contribution.Artifact &&
		digestPattern.MatchString(binding.ProviderDigest) && binding.Provider != nil
}

func providerBindingKey(contributionID string, artifact Artifact) providerBindingIdentity {
	return providerBindingIdentity{contributionID: contributionID, artifact: artifact}
}

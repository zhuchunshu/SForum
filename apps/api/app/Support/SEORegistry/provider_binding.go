package seoregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

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

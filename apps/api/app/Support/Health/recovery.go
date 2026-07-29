package health

import "strings"

const PluginRuntimeRecoveryCode = "plugin_runtime_convergence_failed"

type RecoveryArtifact struct {
	ExtensionID string `json:"extensionId"`
	Version     string `json:"version"`
	Digest      string `json:"digest"`
}

// RecoveryRequirement describes a Host-owned recovery-only boot. It is
// immutable after bootstrap and never changes extension lifecycle authority.
type RecoveryRequirement struct {
	Code                string             `json:"code"`
	Component           string             `json:"component"`
	Message             string             `json:"message"`
	PublicationRevision int64              `json:"publicationRevision,omitempty"`
	Artifacts           []RecoveryArtifact `json:"artifacts,omitempty"`
}

func (r *RecoveryRequirement) Active() bool {
	return r != nil && strings.TrimSpace(r.Code) != ""
}

func ApplyRecoveryRequirement(report ReadyReport, requirement *RecoveryRequirement) ReadyReport {
	if !requirement.Active() {
		return report
	}
	report.Ready = false
	report.Status = "not_ready"
	report.Recovery = requirement
	report.Components = append(report.Components, ComponentResult{
		Name:     requirement.Component,
		Status:   StatusError,
		Required: true,
		Error:    requirement.Message,
	})
	return report
}

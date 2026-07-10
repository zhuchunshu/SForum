package webreleaseruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"

	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

func dependencySnapshotDigest(id string, summary extensionpackage.DependencySummary, bunVersion string, peers extensionpackage.HostPeers) string {
	body, _ := json.Marshal(struct {
		ExtensionID string                        `json:"extensionId"`
		LockDigest  string                        `json:"lockDigest"`
		Resolved    []extensionpackage.Dependency `json:"resolved"`
		BunVersion  string                        `json:"bunVersion"`
		HostPeers   extensionpackage.HostPeers    `json:"hostPeers"`
	}{id, summary.LockDigest, summary.Resolved, bunVersion, peers})
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func boundedLog(parts ...string) string {
	const limit = 1024 * 1024
	joined := strings.TrimSpace(redactBuildLog(strings.Join(parts, "\n")))
	if len(joined) > limit {
		joined = joined[len(joined)-limit:]
	}
	if joined == "" {
		return ""
	}
	return joined + "\n"
}

var (
	urlCredentialPattern = regexp.MustCompile(`(?i)(https?://)[^/@\s]+:[^/@\s]+@`)
	secretValuePattern   = regexp.MustCompile(`(?i)(password|secret|token|api[_-]?key)=([^\s&]+)`)
)

func redactBuildLog(value string) string {
	value = urlCredentialPattern.ReplaceAllString(value, `${1}[redacted]@`)
	return secretValuePattern.ReplaceAllString(value, `${1}=[redacted]`)
}

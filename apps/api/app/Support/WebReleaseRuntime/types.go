package webreleaseruntime

import (
	"context"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

const ReleaseManifestSchemaVersion = 1

type Config struct {
	ReleaseRoot       string
	WebRoot           string
	ExtensionRoot     string
	BunPath           string
	BuildTimeout      time.Duration
	PreviewTimeout    time.Duration
	PreviewPath       string
	HostPeers         extensionpackage.HostPeers
	SourceEnvironment []string
	Runner            CommandRunner
}

type Command struct {
	Path string
	Args []string
	Dir  string
	Env  []string
}

type CommandRunner interface {
	Run(context.Context, Command) (string, error)
}

type PreparedRelease struct {
	Detail          extensions.WebReleaseDetail
	Composition     extensions.WebComposition
	ReleaseDir      string
	Workspace       string
	DevInput        string
	RegistryRoot    string
	ThemeLayer      string
	PluginRoots     map[string]string
	PluginFrontends map[string]string
}

type DependencySnapshot struct {
	ExtensionID  string
	Dependencies []extensions.Dependency
	Digest       string
}

type BuildResult struct {
	ArtifactPath   string
	ArtifactDigest string
	ServerEntry    string
	BuildLog       string
	ManifestPath   string
}

type ReleaseManifest struct {
	SchemaVersion   int       `json:"schemaVersion"`
	ReleaseID       int64     `json:"releaseId"`
	CompositionHash string    `json:"compositionHash"`
	ArtifactPath    string    `json:"artifactPath"`
	ArtifactDigest  string    `json:"artifactDigest"`
	ServerEntry     string    `json:"serverEntry"`
	ThemeID         string    `json:"themeId"`
	ThemeVersion    string    `json:"themeVersion"`
	ThemeLayer      string    `json:"themeLayer"`
	DevInput        string    `json:"devInput"`
	RegistryRoot    string    `json:"registryRoot"`
	ReloadMode      string    `json:"reloadMode"`
	BuiltAt         time.Time `json:"builtAt"`
}

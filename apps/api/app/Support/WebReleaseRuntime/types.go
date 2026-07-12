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
	DefaultThemeLayer string
	BunPath           string
	NodePath          string
	BuildTimeout      time.Duration
	PreviewTimeout    time.Duration
	PreviewPath       string
	// TypecheckFail 是解析器不可用时的回退：true 则 typecheck 失败阻断构建。
	// 默认 false。运行时优先读 TypecheckPolicy（通常来自 web_options）。
	TypecheckFail bool
	// TypecheckPolicy 在每次 Build 时解析是否硬失败；nil 时用 TypecheckFail。
	TypecheckPolicy TypecheckPolicy
	HostPeers         extensionpackage.HostPeers
	SourceEnvironment []string
	Runner            CommandRunner
}

// TypecheckPolicy 决定 Web Release typecheck 失败是否阻断（后台可配置）。
type TypecheckPolicy interface {
	// TypecheckFail 返回 true 表示 typecheck 失败应中止 release。
	TypecheckFail(ctx context.Context) bool
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
	Detail            extensions.WebReleaseDetail
	Composition       extensions.WebComposition
	ReleaseDir        string
	Workspace         string
	DevInput          string
	RegistryRoot      string
	ThemeLayer        string
	DefaultThemeLayer string
	PluginRoots       map[string]string
	PluginFrontends   map[string]string
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

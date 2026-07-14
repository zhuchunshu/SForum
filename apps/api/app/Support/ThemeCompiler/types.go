package themecompiler

import (
	"errors"
	"time"
)

const (
	// CompilerVersion participates in every snapshot identity. Any parsing,
	// helper, or output-contract change must bump this value.
	CompilerVersion = "sforum.theme-compiler@1"

	DefaultMaxSourceBytes = 256 * 1024
	DefaultMaxTotalBytes  = 4 * 1024 * 1024
	DefaultMaxFiles       = 256
	DefaultMaxOutputBytes = 1024 * 1024
	DefaultMaxCallDepth   = 32
	DefaultRenderTimeout  = 2 * time.Second
)

var (
	ErrInvalidInput           = errors.New("themecompiler: invalid input")
	ErrInvalidDigest          = errors.New("themecompiler: invalid package digest")
	ErrInvalidBindingRevision = errors.New("themecompiler: invalid binding revision")
	ErrNoTemplates            = errors.New("themecompiler: no page templates")
	ErrInvalidTemplate        = errors.New("themecompiler: invalid template")
	ErrUnsafeStaticHTML       = errors.New("themecompiler: unsafe static HTML")
	ErrForbiddenHelper        = errors.New("themecompiler: forbidden helper")
	ErrInvalidPartial         = errors.New("themecompiler: invalid partial")
	ErrTemplateRecursion      = errors.New("themecompiler: template recursion limit")
	ErrTemplateNotFound       = errors.New("themecompiler: template not found")
	ErrMissingValue           = errors.New("themecompiler: required template value missing")
	ErrOutputLimit            = errors.New("themecompiler: rendered output limit exceeded")
	ErrRenderTimeout          = errors.New("themecompiler: render timeout")
	ErrExecution              = errors.New("themecompiler: template execution failed")
	ErrHelperValueMissing     = errors.New("themecompiler: helper value not found")
	ErrInvalidViewModel       = errors.New("themecompiler: view model must be a passive DTO")
)

// Limits are copied into a Compiler and then into every compiled snapshot.
// Zero values select conservative host defaults.
type Limits struct {
	MaxSourceBytes int64
	MaxTotalBytes  int64
	MaxFiles       int
	MaxOutputBytes int64
	MaxCallDepth   int
	RenderTimeout  time.Duration
}

func (l Limits) normalized() Limits {
	if l.MaxSourceBytes <= 0 {
		l.MaxSourceBytes = DefaultMaxSourceBytes
	}
	if l.MaxTotalBytes <= 0 {
		l.MaxTotalBytes = DefaultMaxTotalBytes
	}
	if l.MaxFiles <= 0 {
		l.MaxFiles = DefaultMaxFiles
	}
	if l.MaxOutputBytes <= 0 {
		l.MaxOutputBytes = DefaultMaxOutputBytes
	}
	if l.MaxCallDepth <= 0 {
		l.MaxCallDepth = DefaultMaxCallDepth
	}
	if l.RenderTimeout <= 0 {
		l.RenderTimeout = DefaultRenderTimeout
	}
	return l
}

// Bindings back the only Host helpers visible to a theme. Values remain plain
// strings so html/template always applies the current output context.
type Bindings struct {
	// BindingRevision 是上层对 registry/provider/assets/locales/contracts
	// canonical manifest 计算的 SHA-256；编译器不对闭包或 map 自行猜测身份。
	BindingRevision string
	Assets          map[string]string
	Routes          map[string]string
	Translations    map[string]map[string]string
}

// CompiledTemplateKey 只标识可复用的模板编译产物。
type CompiledTemplateKey struct {
	PackageDigest   string `json:"packageDigest"`
	CompilerVersion string `json:"compilerVersion"`
}

// SnapshotKey 标识含精确运行时绑定的不可变发布快照。
type SnapshotKey struct {
	PackageDigest   string `json:"packageDigest"`
	CompilerVersion string `json:"compilerVersion"`
	BindingRevision string `json:"bindingRevision"`
}

type TemplateKind string

const (
	KindLayout  TemplateKind = "layout"
	KindPartial TemplateKind = "partial"
	KindPage    TemplateKind = "template"
)

// TemplateInfo is immutable source metadata. It never exposes a parse tree.
type TemplateInfo struct {
	Name   string       `json:"name"`
	Kind   TemplateKind `json:"kind"`
	Digest string       `json:"digest"`
	Bytes  int64        `json:"bytes"`
}

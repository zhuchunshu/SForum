package editordocument

import "errors"

// StorageVersion is the Host-owned paired schema for accepted editor documents.
// Bumps require an explicit migration path; clients cannot invent versions.
const StorageVersion = "sforum.editor-document@1"

// Pipeline stages are ordered Host contracts. Callers may stop after a stage
// for inspection, but Store always runs the full ordered pipeline.
const (
	StageParse     = "parse"
	StageValidate  = "validate"
	StageNormalize = "normalize"
	StageStore     = "store"
	StageRender    = "render"
	StageSanitize  = "sanitize"
	StageEmbed     = "embed"
	StageSEO       = "seo"
)

var (
	ErrInvalid          = errors.New("editor document is invalid")
	ErrUnsupportedNode  = errors.New("editor document contains an unsupported node or mark")
	ErrStorageVersion   = errors.New("editor document storage version is unsupported")
	ErrPipeline         = errors.New("editor document pipeline stage failed")
	ErrDisabledFallback = errors.New("editor document fell back for disabled plugin content")
)

// Node is a Tiptap JSON node. Attrs remain untyped maps after Host allowlist
// filtering; raw HTML strings are never trusted as executable content.
type Node struct {
	Type    string         `json:"type"`
	Attrs   map[string]any `json:"attrs,omitempty"`
	Content []Node         `json:"content,omitempty"`
	Marks   []Mark         `json:"marks,omitempty"`
	Text    string         `json:"text,omitempty"`
}

type Mark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// Document is the canonical native Tiptap JSON root (usually type=doc).
type Document struct {
	Type    string `json:"type"`
	Content []Node `json:"content,omitempty"`
}

// Schema describes which node/mark types the Host currently admits. Core types
// are always present; plugin types come from Editor/Content Registry catalogs.
type Schema struct {
	Nodes map[string]NodeSpec
	Marks map[string]MarkSpec
}

type NodeSpec struct {
	// Atom nodes have no nested content (emoji, hard break).
	Atom bool
	// FallbackHTML is used when a node type is later disabled; source JSON is
	// preserved in storage even if render falls back.
	FallbackHTML string
	// AllowAttrs is an exact allowlist. Empty means no attributes.
	AllowAttrs map[string]bool
}

type MarkSpec struct {
	AllowAttrs map[string]bool
}

// Accepted is the Host-owned storage triple after the full pipeline.
type Accepted struct {
	StorageVersion string   `json:"storageVersion"`
	Native         Document `json:"native"`
	Markdown       string   `json:"markdown"`
	HTMLSanitized  string   `json:"htmlSanitized"`
	PlainText      string   `json:"plainText"`
	Excerpt        string   `json:"excerpt"`
	SearchText     string   `json:"searchText"`
	ContentHash    string   `json:"contentHash"`
	// Fallbacks lists disabled/unknown node ids that rendered as stable fallbacks.
	Fallbacks []string `json:"fallbacks,omitempty"`
}

// Input is the untrusted client submission.
type Input struct {
	// NativeJSON is preferred. When empty, Markdown may be parsed into a
	// minimal doc of paragraphs for compatibility with markdown-only clients.
	NativeJSON []byte `json:"nativeJson,omitempty"`
	Markdown   string `json:"markdown,omitempty"`
	// Schema is required for validate/normalize against admitted types.
	Schema Schema `json:"-"`
	// ExcerptLimit bounds excerpt runes (Host policy).
	ExcerptLimit int `json:"excerptLimit,omitempty"`
}

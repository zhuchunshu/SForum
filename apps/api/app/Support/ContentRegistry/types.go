package contentregistry

import "errors"

// SchemaVersion is the stable identity of this immutable registry snapshot.
const SchemaVersion = "sforum.content-registry@1"

// Frozen ManifestContent.kind values. Do not invent content-type composition,
// editor toolbar, or pipeline stage kinds here.
const (
	KindBlock        = "block"
	KindShortcode    = "shortcode"
	KindEmbed        = "embed"
	KindNode         = "node"
	KindMark         = "mark"
	KindRenderFilter = "render_filter"
	KindSanitizer    = "sanitizer"
)

var (
	ErrInvalid          = errors.New("content registry declaration is invalid")
	ErrConflict         = errors.New("content registry conflicts with the active graph")
	ErrArtifactConflict = errors.New("content registry artifact does not own the active publication")
	ErrRevisionConflict = errors.New("content registry revision changed during replacement")
	ErrSafeMode         = errors.New("content registry rejects third-party publication in safe mode")
	ErrNotFound         = errors.New("content registry declaration is not found")
)

// Artifact binds every contribution to one exact package. Core publications
// carry a package-private Host seal from NewCoreArtifact. Third-party packages
// always require VersionID; publications with backend handlers additionally
// require RuntimeInstanceID, while renderer-only packages remain inert.
type Artifact struct {
	ExtensionID       string `json:"extensionId"`
	ExtensionVersion  string `json:"extensionVersion"`
	PackageDigest     string `json:"packageDigest"`
	VersionID         int64  `json:"versionId,omitempty"`
	RuntimeInstanceID string `json:"runtimeInstanceId,omitempty"`
	Core              bool   `json:"core,omitempty"`
	// coreSeal is deliberately not serializable or constructible by callers in
	// another package. Core authority must enter through NewCoreArtifact.
	coreSeal [32]byte
}

// Declaration is the frozen ManifestContent surface. Handler/Renderer/
// Migration remain opaque publication references. Executor binds providers to
// an exact active declaration without adding fields to this frozen shape.
type Declaration struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Kind            string `json:"kind"`
	Handler         string `json:"handler,omitempty"`
	Schema          string `json:"schema"`
	Renderer        string `json:"renderer,omitempty"`
	Migration       string `json:"migration,omitempty"`
}

// Publication is one exact-artifact owner of zero or more content declarations.
type Publication struct {
	Artifact Artifact      `json:"artifact"`
	Content  []Declaration `json:"content,omitempty"`
}

// Contribution exposes a frozen declaration together with its exact owner.
type Contribution struct {
	Declaration
	Artifact Artifact `json:"artifact"`
}

// Tombstone permanently reserves one stable content identity and contract for
// its first owner. DefinitionDigest prevents a retained document from being
// reinterpreted under different semantics without a contract-version change.
type Tombstone struct {
	ID               string `json:"id"`
	ContractVersion  string `json:"contractVersion"`
	OwnerExtensionID string `json:"ownerExtensionId"`
	DefinitionDigest string `json:"definitionDigest"`
}

// PublicationRecord remembers the normalized declaration set carried by one
// immutable package. Runtime instance ids are deliberately excluded so a
// process restart can republish the same package, but Safe Mode cannot erase
// the same-artifact drift fence.
type PublicationRecord struct {
	ExtensionID      string `json:"extensionId"`
	ExtensionVersion string `json:"extensionVersion"`
	PackageDigest    string `json:"packageDigest"`
	ContentDigest    string `json:"contentDigest"`
}

// Snapshot is an immutable, inspectable view of the active content graph.
type Snapshot struct {
	SchemaVersion string              `json:"schemaVersion"`
	Revision      uint64              `json:"revision"`
	Digest        string              `json:"digest"`
	SafeMode      bool                `json:"safeMode,omitempty"`
	Publications  []Publication       `json:"publications"`
	Content       []Contribution      `json:"content"`
	Tombstones    []Tombstone         `json:"tombstones"`
	History       []PublicationRecord `json:"history"`
}

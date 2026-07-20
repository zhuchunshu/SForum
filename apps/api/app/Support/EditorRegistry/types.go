package editorregistry

import "errors"

// SchemaVersion is the stable identity of this immutable editor graph.
const SchemaVersion = "sforum.editor-registry@1"

// Frozen ManifestEditor.kind values for Tiptap trusted L2 surfaces.
const (
	KindNode    = "node"
	KindMark    = "mark"
	KindCommand = "command"
	KindToolbar = "toolbar"
)

var (
	ErrInvalid          = errors.New("editor registry declaration is invalid")
	ErrConflict         = errors.New("editor registry conflicts with the active graph")
	ErrArtifactConflict = errors.New("editor registry artifact does not own the active publication")
	ErrRevisionConflict = errors.New("editor registry revision changed during replacement")
	ErrSafeMode         = errors.New("editor registry rejects third-party publication in safe mode")
	ErrNotFound         = errors.New("editor registry declaration is not found")
)

// Artifact binds every contribution to one exact package. Core publications
// carry a package-private Host seal from NewCoreArtifact.
type Artifact struct {
	ExtensionID       string `json:"extensionId"`
	ExtensionVersion  string `json:"extensionVersion"`
	PackageDigest     string `json:"packageDigest"`
	VersionID         int64  `json:"versionId,omitempty"`
	RuntimeInstanceID string `json:"runtimeInstanceId,omitempty"`
	Core              bool   `json:"core,omitempty"`
	// coreSeal is deliberately not serializable. Core authority must enter
	// through NewCoreArtifact only.
	coreSeal [32]byte
}

// Declaration is one frozen Tiptap editor surface contribution.
//
// node/mark require Schema + ExtensionName + L2Module/L2Digest (prebuilt ESM).
// command requires CommandKey (Tiptap command name) and may share L2 module.
// toolbar references CommandID and carries host-rendered chrome metadata.
type Declaration struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Kind            string `json:"kind"`
	// Schema is the paired editor/content schema identity for node/mark.
	Schema string `json:"schema,omitempty"`
	// ExtensionName is the Tiptap extension name registered by the L2 module.
	ExtensionName string `json:"extensionName,omitempty"`
	// L2Module is a package-local archive path to the prebuilt ESM entry.
	L2Module string `json:"l2Module,omitempty"`
	// L2Digest is the sha256 of the prebuilt module bytes (exact-package bind).
	L2Digest string `json:"l2Digest,omitempty"`
	// CommandKey is the Tiptap command name for kind=command.
	CommandKey string `json:"commandKey,omitempty"`
	// CommandID is the stable editor command declaration id for kind=toolbar.
	CommandID string `json:"commandId,omitempty"`
	Label     string `json:"label,omitempty"`
	Icon      string `json:"icon,omitempty"`
	Group     string `json:"group,omitempty"`
	Order     int    `json:"order,omitempty"`
	// Priority orders competing toolbar/command providers. Higher wins.
	Priority   int    `json:"priority,omitempty"`
	Permission string `json:"permission,omitempty"`
}

// Publication is one exact-artifact owner of zero or more editor declarations.
type Publication struct {
	Artifact Artifact      `json:"artifact"`
	Editor   []Declaration `json:"editor,omitempty"`
}

// Contribution exposes a frozen declaration together with its exact owner.
type Contribution struct {
	Declaration
	Artifact Artifact `json:"artifact"`
}

// Snapshot is an immutable, inspectable view of the active editor graph.
type Snapshot struct {
	SchemaVersion string         `json:"schemaVersion"`
	Revision      uint64         `json:"revision"`
	Digest        string         `json:"digest"`
	SafeMode      bool           `json:"safeMode,omitempty"`
	Publications  []Publication  `json:"publications"`
	Editor        []Contribution `json:"editor"`
}

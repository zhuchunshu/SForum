package queryregistry

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

// Registry publishes a complete immutable query graph. Writers validate and
// compose off to the side; concurrent readers observe the previous or next
// snapshot, never a half-removed declaration or partial Safe Mode filter.
type Registry struct {
	mu          sync.Mutex
	state       atomic.Pointer[registryState]
	costPolicy  CostPolicy
	cursorCodec CursorCodec
	admissionMu sync.RWMutex
	admission   func(Artifact) bool
}

type Option func(*Registry)

// WithCostPolicy installs the reviewed Host-owned deterministic cost policy.
// Omitting it keeps planning fail-closed while publication/inspection remains
// available.
func WithCostPolicy(policy CostPolicy) Option {
	return func(registry *Registry) {
		registry.costPolicy = policy
	}
}

// WithCursorCodec installs the Host-owned authenticated cursor codec. Cursor
// continuation remains fail-closed when no codec is configured.
func WithCursorCodec(codec CursorCodec) Option {
	return func(registry *Registry) {
		registry.cursorCodec = codec
	}
}

func New(options ...Option) *Registry {
	registry := &Registry{}
	for _, option := range options {
		if option != nil {
			option(registry)
		}
	}
	registry.state.Store(emptyState())
	return registry
}

// WithPluginAdmission binds planning and release to the Host's exact runtime
// availability gate. It mirrors Route Registry admission: snapshots remain
// inspectable while crashed, staged, or drained runtimes are not executable.
// A nil/missing callback fails third-party planning closed. Sealed Core
// artifacts are Host-owned and bypass the plugin runtime gate.
func (r *Registry) WithPluginAdmission(admission func(Artifact) bool) *Registry {
	if r == nil {
		return r
	}
	r.admissionMu.Lock()
	r.admission = admission
	r.admissionMu.Unlock()
	return r
}

func (r *Registry) artifactAdmitted(artifact Artifact) bool {
	if validCoreArtifactSeal(artifact) {
		return true
	}
	if r == nil || r.load().safeMode {
		return false
	}
	r.admissionMu.RLock()
	admission := r.admission
	r.admissionMu.RUnlock()
	return admission != nil && admission(artifact)
}

func (r *Registry) requireArtifactAdmitted(artifact Artifact) error {
	if !r.artifactAdmitted(artifact) {
		return ErrArtifactUnavailable
	}
	return nil
}

// Publish atomically adds one extension publication or replays the exact active
// artifact idempotently. Artifact changes must use PublishIfArtifact so a stale
// runtime cannot overwrite a newer publication without an exact CAS check.
func (r *Registry) Publish(publication Publication) (uint64, error) {
	return r.publish(nil, publication)
}

// PublishIfArtifact replaces one extension publication only while expected is
// still the exact active artifact. Package rollback uses the same CAS contract.
func (r *Registry) PublishIfArtifact(expected Artifact, publication Publication) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	normalizedExpected, err := normalizeArtifact(expected)
	if err != nil {
		return r.load().revision, ErrInvalid
	}
	return r.publish(&normalizedExpected, publication)
}

func (r *Registry) publish(expected *Artifact, publication Publication) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	// Safe Mode rejects every ordinary/unsealed publication before parsing its
	// extension-controlled declarations. A Core bool or core.* prefix is not a
	// bypass; only NewCoreArtifact carries the private Host seal.
	if current := r.load(); current.safeMode && !validCoreArtifactSeal(publication.Artifact) {
		return current.revision, ErrSafeMode
	}
	normalized, err := normalizePublication(publication)
	if err != nil {
		return r.load().revision, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	current := r.load()
	if current.safeMode && !normalized.Artifact.Core {
		return current.revision, ErrSafeMode
	}
	active, found := current.publications[normalized.Artifact.ExtensionID]
	if expected != nil {
		if expected.ExtensionID != normalized.Artifact.ExtensionID || !found || active.Artifact != *expected {
			return current.revision, ErrArtifactConflict
		}
	} else if found && active.Artifact != normalized.Artifact {
		return current.revision, ErrArtifactConflict
	}
	// 同 exact artifact 仅允许语义等价重放；声明漂移 fail-closed。
	if found && active.Artifact == normalized.Artifact && !equalPublications(active, normalized) {
		return current.revision, ErrArtifactConflict
	}
	publications := clonePublicationMap(current.publications)
	publications[normalized.Artifact.ExtensionID] = normalized
	next, err := buildState(current.revision+1, publicationValues(publications), current.safeMode)
	if err != nil {
		return current.revision, err
	}
	if equalPublicationMaps(current.publications, next.publications) && current.safeMode == next.safeMode {
		return current.revision, nil
	}
	r.state.Store(next)
	return next.revision, nil
}

// ReplaceAll builds one complete immutable snapshot after full graph preflight.
// Input order cannot change provider winners, query ordering, or the stable
// digest. Invalid graphs leave the previous revision untouched.
//
// When safeMode is true, non-core publications are filtered from the input
// before validation so corrupt third-party queries cannot block Host recovery.
func (r *Registry) ReplaceAll(publications []Publication, safeMode bool) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	return r.ReplaceAllIfRevision(r.load().revision, publications, safeMode)
}

// ReplaceAllIfRevision publishes one fully validated graph only while the
// caller's observed revision remains current. Startup restoration uses this
// fence so a concurrent lifecycle publication or removal cannot be overwritten.
func (r *Registry) ReplaceAllIfRevision(
	expectedRevision uint64,
	publications []Publication,
	safeMode bool,
) (uint64, error) {
	if r == nil {
		return 0, ErrInvalid
	}
	// 完整图先在锁外构建，锁内只做 revision CAS、exact replay 校验和发布。
	next, err := buildState(0, publications, safeMode)
	if err != nil {
		return r.load().revision, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	if current.revision != expectedRevision {
		return current.revision, ErrRevisionConflict
	}
	if err := validateExactPublicationReplay(current.publications, next.publications); err != nil {
		return current.revision, err
	}
	if equalPublicationMaps(current.publications, next.publications) && current.safeMode == next.safeMode {
		return current.revision, nil
	}
	next.revision = current.revision + 1
	r.state.Store(next)
	return next.revision, nil
}

// Remove disables one exact artifact publication. A stale shutdown cannot remove
// a replacement artifact.
func (r *Registry) Remove(artifact Artifact) (uint64, bool, error) {
	if r == nil {
		return 0, false, ErrInvalid
	}
	normalized, err := normalizeArtifact(artifact)
	if err != nil {
		return r.load().revision, false, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	publication, found := current.publications[normalized.ExtensionID]
	if !found {
		return current.revision, false, nil
	}
	if publication.Artifact != normalized {
		return current.revision, false, ErrArtifactConflict
	}
	publications := clonePublicationMap(current.publications)
	delete(publications, normalized.ExtensionID)
	next, err := buildState(current.revision+1, publicationValues(publications), current.safeMode)
	if err != nil {
		return current.revision, false, err
	}
	r.state.Store(next)
	return next.revision, true, nil
}

func equalPublicationMaps(left, right map[string]Publication) bool {
	if len(left) != len(right) {
		return false
	}
	for extensionID, publication := range left {
		candidate, found := right[extensionID]
		if !found || !equalPublications(publication, candidate) {
			return false
		}
	}
	return true
}

func equalPublications(left, right Publication) bool {
	return reflect.DeepEqual(publicationContract(left), publicationContract(right))
}

func publicationContract(value Publication) Publication {
	value = clonePublication(value)
	for index := range value.Queries {
		// equality/digest 只比较稳定公开 metadata 与 digests，禁止 callable 指针入合同。
		value.Queries[index].boundResultSchema = nil
		value.Queries[index].boundProvider = nil
	}
	for index := range value.ResultFilters {
		value.ResultFilters[index].boundFilter = nil
		// Identity is derived from the target query in each complete graph. It can
		// change when that owner upgrades without changing the filter artifact.
		value.ResultFilters[index].IdentityFields = nil
	}
	return value
}

func validateExactPublicationReplay(current, next map[string]Publication) error {
	for extensionID, active := range current {
		candidate, found := next[extensionID]
		if !found || active.Artifact != candidate.Artifact {
			continue
		}
		if !equalPublications(active, candidate) {
			return fmt.Errorf("%w: exact artifact %s changed declarations", ErrArtifactConflict, extensionID)
		}
	}
	return nil
}

func (r *Registry) Revision() uint64 {
	return r.load().revision
}

func (r *Registry) CacheState() CacheState {
	state := r.load()
	return CacheState{Revision: state.revision, Digest: state.digest, SafeMode: state.safeMode}
}

// CacheInvalidated lets callers fence a cached plan with both the local
// monotonic revision and restart-stable graph digest.
func (r *Registry) CacheInvalidated(previous CacheState) bool {
	current := r.CacheState()
	return previous.Revision != current.Revision || previous.Digest != current.Digest || previous.SafeMode != current.SafeMode
}

func (r *Registry) Snapshot() Snapshot {
	return cloneSnapshot(snapshotFromState(r.load()))
}

// SnapshotPublication returns one extension's exact active publication, if any.
// Empty query publications remain inspectable.
func (r *Registry) SnapshotPublication(extensionID string) (Publication, bool) {
	if r == nil {
		return Publication{}, false
	}
	publication, ok := r.load().publications[strings.ToLower(strings.TrimSpace(extensionID))]
	if !ok {
		return Publication{}, false
	}
	return clonePublication(publication), true
}

// Resolve returns one exact query contribution from the active snapshot.
func (r *Registry) Resolve(queryID string) (QueryContribution, error) {
	if r == nil {
		return QueryContribution{}, ErrInvalid
	}
	contribution, ok := r.load().queries[strings.ToLower(strings.TrimSpace(queryID))]
	if !ok {
		return QueryContribution{}, ErrNotFound
	}
	return cloneContribution(contribution), nil
}

func (r *Registry) load() *registryState {
	if r != nil {
		if state := r.state.Load(); state != nil {
			return state
		}
	}
	return emptyState()
}

func (r *Registry) String() string {
	state := r.load()
	return fmt.Sprintf("QueryRegistry(revision=%d,digest=%s,safeMode=%t)", state.revision, state.digest, state.safeMode)
}

package extensionopenapi

// ArtifactLoader memoizes loaded and digest-verified extension artifacts for the
// duration of one lifecycle gate, which compiles several candidate aggregates
// over the same immutable artifact set.
//
// A cache hit returns the exact contracts that were already parsed and verified
// for that identity. The cost being skipped is re-reading and re-hashing a
// package tree that cannot have changed inside one synchronous gate, because
// loading a single package artifact verifies its complete tree digest and its
// declared package files.
//
// The cache deliberately carries no TTL and is never process-global: it lives
// exactly as long as the gate that created it, so exact-artifact verification
// still runs at least once per validation fence. Callers that need a fresh
// verification build a new loader. A nil *ArtifactLoader disables reuse.
type ArtifactLoader struct {
	entries map[string]*loadedArtifact
}

// NewArtifactLoader returns an empty loader.
func NewArtifactLoader() *ArtifactLoader {
	return &ArtifactLoader{entries: make(map[string]*loadedArtifact)}
}

// key covers every field that participates in the loaded result. Version is
// part of the identity because it is published into fragment source identity
// even though the package digest already pins the tree contents.
func (l *ArtifactLoader) key(input Artifact) string {
	return input.ExtensionID + "\x00" + input.Version + "\x00" + input.PackageDigest + "\x00" + input.Root
}

func (l *ArtifactLoader) get(input Artifact) (*loadedArtifact, bool) {
	if l == nil || l.entries == nil {
		return nil, false
	}
	entry, ok := l.entries[l.key(input)]
	return entry, ok
}

func (l *ArtifactLoader) put(input Artifact, entry *loadedArtifact) {
	if l == nil || l.entries == nil || entry == nil {
		return
	}
	l.entries[l.key(input)] = entry
}

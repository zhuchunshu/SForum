package settingslifecycle

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 与 extension_settings 共用的元数据键；字段名本身仍是普通 settings 键。
const (
	metaDataVersionKey = "__sforum.data_version"
	metaRevisionKey    = "__sforum.revision"
	metaUpdatedAtKey   = "__sforum.updated_at"
	metaUpdatedByKey   = "__sforum.updated_by"
	metaSecretSetKey   = "__sforum.secret_set" // JSON map[string]bool
)

// DocumentStore is the durable authority for versioned settings documents.
// Production should back this with extension_settings (SettingsKVStore).
type DocumentStore interface {
	// Load returns the document and optimistic revision (0 if absent).
	Load(ctx context.Context, extensionID string) (Document, int64, error)
	// Save persists the document. expectedRevision 0 means create-or-overwrite
	// without CAS; positive requires matching current revision (ErrConflict).
	// Returns the new revision.
	Save(ctx context.Context, extensionID string, doc Document, expectedRevision int64) (int64, error)
}

// SettingsKV is the extension_settings persistence surface (Models/Extensions Store subset).
// Implemented by extensions.PostgresStore and test fakes.
type SettingsKV interface {
	ListSettings(ctx context.Context, extensionID string) (map[string]string, error)
	ReplaceSettings(ctx context.Context, extensionID string, values map[string]string) error
	CompareAndSwapSetting(ctx context.Context, extensionID, name, oldValue, newValue string) (bool, error)
}

// SettingsKVStore maps Document <-> extension_settings rows.
// Secret fields store only sforum.secret:// references, never plaintext.
type SettingsKVStore struct {
	kv SettingsKV
}

// NewSettingsKVStore wraps an extension settings backend.
func NewSettingsKVStore(kv SettingsKV) (*SettingsKVStore, error) {
	if kv == nil {
		return nil, ErrInvalid
	}
	return &SettingsKVStore{kv: kv}, nil
}

// Load implements DocumentStore.
func (s *SettingsKVStore) Load(ctx context.Context, extensionID string) (Document, int64, error) {
	if s == nil || s.kv == nil || ctx == nil {
		return Document{}, 0, ErrInvalid
	}
	extensionID = normalizeID(extensionID)
	if extensionID == "" {
		return Document{}, 0, ErrInvalid
	}
	raw, err := s.kv.ListSettings(ctx, extensionID)
	if err != nil {
		return Document{}, 0, err
	}
	if len(raw) == 0 {
		return Document{}, 0, ErrNotFound
	}
	doc, rev := documentFromKV(extensionID, raw)
	return doc, rev, nil
}

// Save implements DocumentStore with optional CAS on __sforum.revision.
func (s *SettingsKVStore) Save(ctx context.Context, extensionID string, doc Document, expectedRevision int64) (int64, error) {
	if s == nil || s.kv == nil || ctx == nil {
		return 0, ErrInvalid
	}
	extensionID = normalizeID(extensionID)
	if extensionID == "" {
		return 0, ErrInvalid
	}
	// CAS path: claim next revision before full replace.
	currentRaw, err := s.kv.ListSettings(ctx, extensionID)
	if err != nil {
		return 0, err
	}
	currentRev := parseRevision(currentRaw[metaRevisionKey])
	if expectedRevision > 0 && currentRev != expectedRevision {
		return 0, ErrConflict
	}
	nextRev := currentRev + 1
	if nextRev < 1 {
		nextRev = 1
	}
	if expectedRevision > 0 {
		old := currentRaw[metaRevisionKey]
		ok, casErr := s.kv.CompareAndSwapSetting(ctx, extensionID, metaRevisionKey, old, strconv.FormatInt(nextRev, 10))
		if casErr != nil {
			return 0, casErr
		}
		if !ok {
			// 首次创建时 old 为空：CAS 可能失败，走 Replace 路径校验。
			if old != "" || currentRev != 0 {
				return 0, ErrConflict
			}
		}
	}
	payload := documentToKV(doc, nextRev)
	if err := s.kv.ReplaceSettings(ctx, extensionID, payload); err != nil {
		return 0, err
	}
	return nextRev, nil
}

func documentFromKV(extensionID string, raw map[string]string) (Document, int64) {
	doc := Document{
		SchemaVersion: SchemaVersion,
		ExtensionID:   extensionID,
		DataVersion:   1,
		Values:        map[string]string{},
		SecretRefs:    map[string]string{},
		SecretSet:     map[string]bool{},
	}
	if v := strings.TrimSpace(raw[metaDataVersionKey]); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			doc.DataVersion = n
		}
	}
	if v := strings.TrimSpace(raw[metaUpdatedByKey]); v != "" {
		doc.UpdatedBy = v
	}
	if v := strings.TrimSpace(raw[metaUpdatedAtKey]); v != "" {
		if ts, err := time.Parse(time.RFC3339Nano, v); err == nil {
			doc.UpdatedAt = ts
		}
	}
	if v := strings.TrimSpace(raw[metaSecretSetKey]); v != "" {
		_ = json.Unmarshal([]byte(v), &doc.SecretSet)
	}
	rev := parseRevision(raw[metaRevisionKey])
	for key, value := range raw {
		if isMetaKey(key) {
			continue
		}
		if strings.HasPrefix(value, "sforum.secret://") {
			doc.SecretRefs[key] = value
			doc.SecretSet[key] = true
			continue
		}
		doc.Values[key] = value
	}
	return doc, rev
}

func documentToKV(doc Document, revision int64) map[string]string {
	out := make(map[string]string, len(doc.Values)+len(doc.SecretRefs)+8)
	for k, v := range doc.Values {
		if isMetaKey(k) {
			continue
		}
		// 禁止把密文或 secret ref 误写入普通 values。
		if strings.HasPrefix(v, "enc::") || strings.HasPrefix(v, "sforum.secret://") {
			continue
		}
		out[k] = v
	}
	for k, ref := range doc.SecretRefs {
		if strings.TrimSpace(ref) == "" {
			continue
		}
		out[k] = ref
	}
	out[metaDataVersionKey] = strconv.Itoa(doc.DataVersion)
	out[metaRevisionKey] = strconv.FormatInt(revision, 10)
	out[metaUpdatedByKey] = doc.UpdatedBy
	if !doc.UpdatedAt.IsZero() {
		out[metaUpdatedAtKey] = doc.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	if len(doc.SecretSet) > 0 {
		if blob, err := json.Marshal(doc.SecretSet); err == nil {
			out[metaSecretSetKey] = string(blob)
		}
	}
	return out
}

func isMetaKey(key string) bool {
	return strings.HasPrefix(key, "__sforum.")
}

func parseRevision(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// MemoryDocumentStore is process-local for unit tests only.
type MemoryDocumentStore struct {
	mu   sync.Mutex
	docs map[string]storedDoc
}

type storedDoc struct {
	doc      Document
	revision int64
}

// NewMemoryDocumentStore builds an empty in-memory document store.
func NewMemoryDocumentStore() *MemoryDocumentStore {
	return &MemoryDocumentStore{docs: make(map[string]storedDoc)}
}

// Load implements DocumentStore.
func (s *MemoryDocumentStore) Load(_ context.Context, extensionID string) (Document, int64, error) {
	if s == nil {
		return Document{}, 0, ErrInvalid
	}
	extensionID = normalizeID(extensionID)
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.docs[extensionID]
	if !ok {
		return Document{}, 0, ErrNotFound
	}
	return cloneDocument(stored.doc), stored.revision, nil
}

// Save implements DocumentStore.
func (s *MemoryDocumentStore) Save(_ context.Context, extensionID string, doc Document, expectedRevision int64) (int64, error) {
	if s == nil {
		return 0, ErrInvalid
	}
	extensionID = normalizeID(extensionID)
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.docs[extensionID]
	if expectedRevision > 0 {
		if !ok || current.revision != expectedRevision {
			return 0, ErrConflict
		}
	}
	next := current.revision + 1
	if next < 1 {
		next = 1
	}
	s.docs[extensionID] = storedDoc{doc: cloneDocument(doc), revision: next}
	return next, nil
}

// Seed inserts a document at revision 1 without CAS (tests).
func (s *MemoryDocumentStore) Seed(extensionID string, doc Document) {
	if s == nil {
		return
	}
	extensionID = normalizeID(extensionID)
	s.mu.Lock()
	s.docs[extensionID] = storedDoc{doc: cloneDocument(doc), revision: 1}
	s.mu.Unlock()
}

// MemorySettingsKV is a test double for SettingsKV.
type MemorySettingsKV struct {
	mu   sync.Mutex
	data map[string]map[string]string
}

// NewMemorySettingsKV builds an empty settings map store.
func NewMemorySettingsKV() *MemorySettingsKV {
	return &MemorySettingsKV{data: make(map[string]map[string]string)}
}

// ListSettings implements SettingsKV.
func (s *MemorySettingsKV) ListSettings(_ context.Context, extensionID string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src := s.data[extensionID]
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out, nil
}

// ReplaceSettings implements SettingsKV.
func (s *MemorySettingsKV) ReplaceSettings(_ context.Context, extensionID string, values map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := make(map[string]string, len(values))
	for k, v := range values {
		cloned[k] = v
	}
	s.data[extensionID] = cloned
	return nil
}

// CompareAndSwapSetting implements SettingsKV.
func (s *MemorySettingsKV) CompareAndSwapSetting(_ context.Context, extensionID, name, oldValue, newValue string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row := s.data[extensionID]
	if row == nil {
		if oldValue != "" {
			return false, nil
		}
		row = map[string]string{}
		s.data[extensionID] = row
	}
	if row[name] != oldValue {
		return false, nil
	}
	row[name] = newValue
	return true, nil
}

// Ensure DocumentStore implementations stay honest about errors.
var (
	_ DocumentStore = (*MemoryDocumentStore)(nil)
	_ DocumentStore = (*SettingsKVStore)(nil)
	_ SettingsKV    = (*MemorySettingsKV)(nil)
)

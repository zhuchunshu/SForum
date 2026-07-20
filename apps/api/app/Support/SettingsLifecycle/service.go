package settingslifecycle

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	secretstore "github.com/zhuchunshu/sforum/apps/api/app/Support/SecretStore"
)

// Service is the Host settings lifecycle runtime (process-local document store).
type Service struct {
	mu         sync.Mutex
	documents  map[string]Document // extensionID -> document
	schemas    map[string][]FieldSchema
	migrations map[string][]Migration // extensionID -> ordered migrations
	secrets    *secretstore.Service
	targetVer  map[string]int // extensionID -> current schema data version
}

// New builds a settings lifecycle service. secrets may be nil (refs still work as strings).
func New(secrets *secretstore.Service) *Service {
	return &Service{
		documents:  make(map[string]Document),
		schemas:    make(map[string][]FieldSchema),
		migrations: make(map[string][]Migration),
		secrets:    secrets,
		targetVer:  make(map[string]int),
	}
}

// RegisterSchema installs field schema and target data version for an extension.
func (s *Service) RegisterSchema(extensionID string, dataVersion int, fields []FieldSchema) error {
	if s == nil {
		return ErrInvalid
	}
	extensionID = normalizeID(extensionID)
	if extensionID == "" || dataVersion < 1 || len(fields) == 0 {
		return ErrInvalid
	}
	cloned := make([]FieldSchema, len(fields))
	copy(cloned, fields)
	s.mu.Lock()
	s.schemas[extensionID] = cloned
	s.targetVer[extensionID] = dataVersion
	s.mu.Unlock()
	return nil
}

// RegisterMigration adds a data migration step.
func (s *Service) RegisterMigration(extensionID string, migration Migration) error {
	if s == nil || migration.Apply == nil || migration.From < 0 || migration.To <= migration.From {
		return ErrInvalid
	}
	extensionID = normalizeID(extensionID)
	if extensionID == "" {
		return ErrInvalid
	}
	s.mu.Lock()
	s.migrations[extensionID] = append(s.migrations[extensionID], migration)
	s.mu.Unlock()
	return nil
}

// Put replaces values (non-secret) and optional secret plaintexts (via Secret Store).
// Empty secret submissions preserve existing secrets when preserveSecrets is true.
func (s *Service) Put(extensionID, actor string, values map[string]string, preserveSecrets bool) (Document, error) {
	if s == nil {
		return Document{}, ErrInvalid
	}
	extensionID = normalizeID(extensionID)
	actor = strings.TrimSpace(actor)
	if extensionID == "" || actor == "" {
		return Document{}, ErrPermissionDenied
	}
	s.mu.Lock()
	fields := s.schemas[extensionID]
	target := s.targetVer[extensionID]
	current, hasCurrent := s.documents[extensionID]
	s.mu.Unlock()
	if len(fields) == 0 || target < 1 {
		return Document{}, ErrNotFound
	}
	if values == nil {
		values = map[string]string{}
	}
	doc := Document{
		SchemaVersion: SchemaVersion, ExtensionID: extensionID, DataVersion: target,
		Values: make(map[string]string), SecretRefs: map[string]string{}, SecretSet: map[string]bool{},
		UpdatedAt: time.Now().UTC(), UpdatedBy: actor,
	}
	if hasCurrent {
		for k, v := range current.Values {
			doc.Values[k] = v
		}
		for k, v := range current.SecretRefs {
			doc.SecretRefs[k] = v
		}
		for k, v := range current.SecretSet {
			doc.SecretSet[k] = v
		}
		doc.DataVersion = current.DataVersion
	}
	for _, field := range fields {
		raw, present := values[field.Name]
		if field.Secret || field.Type == "secret" {
			if !present || secretstore.ShouldPreserve(raw) {
				if preserveSecrets {
					continue
				}
				// Explicit clear only when preserveSecrets is false and empty submitted.
				if present && secretstore.ShouldPreserve(raw) && !preserveSecrets {
					delete(doc.SecretRefs, field.Name)
					doc.SecretSet[field.Name] = false
				}
				continue
			}
			if s.secrets == nil {
				return Document{}, ErrInvalid
			}
			ref := secretstore.Ref{Namespace: extensionID, SecretID: field.Name}
			meta, err := s.secrets.Put(context.Background(), ref, []byte(raw), secretstore.PutOptions{
				Actor: actor, Purposes: []string{"settings"},
			})
			if err != nil {
				return Document{}, err
			}
			doc.SecretRefs[field.Name] = meta.Reference
			doc.SecretSet[field.Name] = true
			continue
		}
		if present {
			doc.Values[field.Name] = raw
		}
	}
	// Migrate if needed.
	migrated, err := s.migrateLocked(extensionID, doc)
	if err != nil {
		return Document{}, err
	}
	s.mu.Lock()
	s.documents[extensionID] = migrated
	s.mu.Unlock()
	return cloneDocument(migrated), nil
}

// Get returns the current document (secrets masked).
func (s *Service) Get(extensionID string) (Document, error) {
	if s == nil {
		return Document{}, ErrInvalid
	}
	extensionID = normalizeID(extensionID)
	s.mu.Lock()
	doc, ok := s.documents[extensionID]
	s.mu.Unlock()
	if !ok {
		return Document{}, ErrNotFound
	}
	return cloneDocument(doc), nil
}

// ResetDefaults restores field defaults and clears non-preserved secrets.
func (s *Service) ResetDefaults(extensionID, actor string) (Document, error) {
	if s == nil {
		return Document{}, ErrInvalid
	}
	extensionID = normalizeID(extensionID)
	actor = strings.TrimSpace(actor)
	if extensionID == "" || actor == "" {
		return Document{}, ErrPermissionDenied
	}
	s.mu.Lock()
	fields := append([]FieldSchema(nil), s.schemas[extensionID]...)
	target := s.targetVer[extensionID]
	s.mu.Unlock()
	if len(fields) == 0 {
		return Document{}, ErrNotFound
	}
	values := make(map[string]string, len(fields))
	for _, field := range fields {
		if field.Secret || field.Type == "secret" {
			continue
		}
		values[field.Name] = field.Default
	}
	// Replace document with defaults; secrets cleared.
	doc := Document{
		SchemaVersion: SchemaVersion, ExtensionID: extensionID, DataVersion: target,
		Values: values, SecretRefs: map[string]string{}, SecretSet: map[string]bool{},
		UpdatedAt: time.Now().UTC(), UpdatedBy: actor,
	}
	s.mu.Lock()
	s.documents[extensionID] = doc
	s.mu.Unlock()
	return cloneDocument(doc), nil
}

// Preview validates values and evaluates conditionals without persistence.
func (s *Service) Preview(extensionID string, values map[string]string) (ValidationPreview, error) {
	if s == nil {
		return ValidationPreview{}, ErrInvalid
	}
	extensionID = normalizeID(extensionID)
	s.mu.Lock()
	fields := append([]FieldSchema(nil), s.schemas[extensionID]...)
	s.mu.Unlock()
	if len(fields) == 0 {
		return ValidationPreview{}, ErrNotFound
	}
	if values == nil {
		values = map[string]string{}
	}
	preview := ValidationPreview{OK: true, Errors: map[string]string{}, Warnings: map[string]string{}}
	for _, field := range fields {
		if !fieldVisible(field, values) {
			continue
		}
		preview.VisibleFields = append(preview.VisibleFields, field.Name)
		raw := values[field.Name]
		if field.Required && strings.TrimSpace(raw) == "" && !field.Secret && field.Type != "secret" {
			preview.OK = false
			preview.Errors[field.Name] = "required"
			continue
		}
		if field.Type == "select" && len(field.Options) > 0 && strings.TrimSpace(raw) != "" {
			if !contains(field.Options, raw) {
				preview.OK = false
				preview.Errors[field.Name] = "invalid_option"
			}
		}
	}
	return preview, nil
}

// Export returns a masked bundle (no secret plaintext).
func (s *Service) Export(extensionID string) (ExportBundle, error) {
	doc, err := s.Get(extensionID)
	if err != nil {
		return ExportBundle{}, err
	}
	return ExportBundle{
		SchemaVersion: SchemaVersion, ExtensionID: doc.ExtensionID, DataVersion: doc.DataVersion,
		Values: cloneMap(doc.Values), SecretRefs: cloneMap(doc.SecretRefs), SecretSet: cloneBoolMap(doc.SecretSet),
		SecretsNeverIncluded: true,
	}, nil
}

// Import applies a bundle. Secret refs are kept; plaintext secrets are rejected.
func (s *Service) Import(extensionID, actor string, bundle ExportBundle) (Document, error) {
	if s == nil {
		return Document{}, ErrInvalid
	}
	extensionID = normalizeID(extensionID)
	actor = strings.TrimSpace(actor)
	if extensionID == "" || actor == "" || bundle.ExtensionID != "" && normalizeID(bundle.ExtensionID) != extensionID {
		return Document{}, ErrInvalid
	}
	// Reject any value that looks like a leaked secret payload.
	for _, value := range bundle.Values {
		if strings.HasPrefix(value, "enc::") {
			return Document{}, fmt.Errorf("%w: ciphertext in values", ErrValidation)
		}
	}
	s.mu.Lock()
	target := s.targetVer[extensionID]
	fields := s.schemas[extensionID]
	s.mu.Unlock()
	if len(fields) == 0 {
		return Document{}, ErrNotFound
	}
	doc := Document{
		SchemaVersion: SchemaVersion, ExtensionID: extensionID,
		DataVersion: bundle.DataVersion, Values: cloneMap(bundle.Values),
		SecretRefs: cloneMap(bundle.SecretRefs), SecretSet: cloneBoolMap(bundle.SecretSet),
		UpdatedAt: time.Now().UTC(), UpdatedBy: actor,
	}
	if doc.DataVersion == 0 {
		doc.DataVersion = 1
	}
	if doc.DataVersion != target {
		migrated, err := s.migrateLocked(extensionID, doc)
		if err != nil {
			return Document{}, err
		}
		doc = migrated
	}
	s.mu.Lock()
	s.documents[extensionID] = doc
	s.mu.Unlock()
	return cloneDocument(doc), nil
}

func (s *Service) migrateLocked(extensionID string, doc Document) (Document, error) {
	s.mu.Lock()
	target := s.targetVer[extensionID]
	steps := append([]Migration(nil), s.migrations[extensionID]...)
	s.mu.Unlock()
	if target < 1 {
		return doc, nil
	}
	// Upgrade only (From -> To ascending).
	guard := 0
	for doc.DataVersion < target {
		guard++
		if guard > 64 {
			return Document{}, fmt.Errorf("%w: migration loop", ErrMigration)
		}
		applied := false
		for _, step := range steps {
			if step.From == doc.DataVersion && step.To > step.From {
				next, err := step.Apply(cloneMap(doc.Values))
				if err != nil {
					return Document{}, fmt.Errorf("%w: %v", ErrMigration, err)
				}
				doc.Values = next
				doc.DataVersion = step.To
				applied = true
				break
			}
		}
		if !applied {
			return Document{}, fmt.Errorf("%w: missing step from %d to %d", ErrMigration, doc.DataVersion, target)
		}
	}
	return doc, nil
}

func fieldVisible(field FieldSchema, values map[string]string) bool {
	cond := strings.TrimSpace(field.VisibleWhen)
	if cond == "" {
		return true
	}
	parts := strings.SplitN(cond, "=", 2)
	if len(parts) != 2 {
		return true
	}
	return values[strings.TrimSpace(parts[0])] == strings.TrimSpace(parts[1])
}

func normalizeID(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func cloneDocument(doc Document) Document {
	out := doc
	out.Values = cloneMap(doc.Values)
	out.SecretRefs = cloneMap(doc.SecretRefs)
	out.SecretSet = cloneBoolMap(doc.SecretSet)
	return out
}

func cloneMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneBoolMap(in map[string]bool) map[string]bool {
	if in == nil {
		return map[string]bool{}
	}
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

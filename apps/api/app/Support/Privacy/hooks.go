// Package privacy is the Host privacy export/erase registry for V3 P12.
// Contributions may be published via lifecycle publication / Protocol V2;
// Host orchestrates export/erase with permission, audit, deadline, partial
// failure, and retained external-resource warnings.
package privacy

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// SchemaVersion is the privacy inventory contract.
const SchemaVersion = "sforum.privacy@1"

const (
	KindPersonalData = "personal_data"
	KindAttachment   = "attachment"
	KindExternal     = "external_resource"
	KindLog          = "log"

	// DefaultDeadline is used when the request context has no deadline.
	DefaultDeadline = 30 * time.Second
)

var (
	ErrInvalid          = errors.New("privacy input is invalid")
	ErrNotFound         = errors.New("privacy contribution is not found")
	ErrPermissionDenied = errors.New("privacy permission denied")
	ErrHookFailed       = errors.New("privacy hook failed")
	ErrPartial          = errors.New("privacy operation partially failed")
	ErrDeadline         = errors.New("privacy operation deadline exceeded")
)

// Contribution is one plugin's privacy surface declaration.
type Contribution struct {
	ExtensionID   string `json:"extensionId"`
	PackageDigest string `json:"packageDigest,omitempty"`
	// Inventory describes categories of data the plugin holds.
	Inventory []InventoryItem `json:"inventory"`
	// SupportsExport/Erase declare Host-invokable hooks.
	SupportsExport bool `json:"supportsExport"`
	SupportsErase  bool `json:"supportsErase"`
}

// InventoryItem is one data category.
type InventoryItem struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	// RetentionDays 0 means follow Host default / until erase.
	RetentionDays int `json:"retentionDays,omitempty"`
	// External means Host cannot fully erase without operator action.
	External bool `json:"external,omitempty"`
}

// ExportHook produces a portable export blob for one user (no secrets).
type ExportHook func(ctx context.Context, userID string) (ExportArtifact, error)

// EraseHook erases or anonymizes one user; returns retained external warnings.
type EraseHook func(ctx context.Context, userID string) (EraseResult, error)

// ExportArtifact is a Host-collected export piece.
type ExportArtifact struct {
	ExtensionID string `json:"extensionId"`
	// MediaType e.g. application/json
	MediaType string `json:"mediaType"`
	// Body is plugin-owned portable data (never Host secrets).
	Body []byte `json:"-"`
}

// EraseResult reports erase outcome.
type EraseResult struct {
	ExtensionID string   `json:"extensionId"`
	Erased      bool     `json:"erased"`
	// RetainedExternal lists resources Host cannot delete (CDN, 3rd party).
	RetainedExternal []string `json:"retainedExternal,omitempty"`
	Warnings         []string `json:"warnings,omitempty"`
	// Error is set on partial failure for this extension.
	Error string `json:"error,omitempty"`
}

// ExportBundle is the full user export.
type ExportBundle struct {
	SchemaVersion string           `json:"schemaVersion"`
	UserID        string           `json:"userId"`
	GeneratedAt   time.Time        `json:"generatedAt"`
	Artifacts     []ExportArtifact `json:"artifacts"`
	Inventory     []InventoryItem  `json:"inventory"`
	// Partial is true when some hooks failed but others succeeded.
	Partial  bool     `json:"partial,omitempty"`
	Failures []string `json:"failures,omitempty"`
}

// EraseReport is the full erase report for operators and the user.
type EraseReport struct {
	SchemaVersion    string        `json:"schemaVersion"`
	UserID           string        `json:"userId"`
	At               time.Time     `json:"at"`
	Results          []EraseResult `json:"results"`
	RetainedExternal []string      `json:"retainedExternal,omitempty"`
	Partial          bool          `json:"partial,omitempty"`
	Failures         []string      `json:"failures,omitempty"`
}

// AuditEvent is one export/erase audit record (no payload bodies).
type AuditEvent struct {
	AuditID   string    `json:"auditId"`
	Operation string    `json:"operation"` // export|erase
	Actor     string    `json:"actor"`
	UserID    string    `json:"userId"`
	Status    string    `json:"status"` // ok|partial|failed
	Detail    string    `json:"detail,omitempty"`
	At        time.Time `json:"at"`
}

// Auditor persists privacy operation audits.
type Auditor interface {
	Append(ctx context.Context, event AuditEvent) error
}

// MemoryAuditor is tests-only audit ring.
type MemoryAuditor struct {
	mu     sync.Mutex
	events []AuditEvent
}

// Append implements Auditor.
func (a *MemoryAuditor) Append(_ context.Context, event AuditEvent) error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	a.events = append(a.events, event)
	a.mu.Unlock()
	return nil
}

// Events returns a copy of audit events.
func (a *MemoryAuditor) Events() []AuditEvent {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]AuditEvent(nil), a.events...)
}

// PermissionCheck gates export/erase. Production wires Host RBAC.
type PermissionCheck func(ctx context.Context, actor, userID, operation string) error

// Registry holds privacy contributions and hooks.
type Registry struct {
	mu            sync.Mutex
	contributions map[string]Contribution
	exportHooks   map[string]ExportHook
	eraseHooks    map[string]EraseHook
	// protocolHandlers are non-Go-callback contributions (Protocol V2 / publication).
	protocolHandlers map[string]Contribution
	auditor          Auditor
	// allow is optional; when nil, non-empty actor is sufficient.
	allow PermissionCheck
}

// New builds an empty privacy registry.
func New() *Registry {
	return &Registry{
		contributions:    make(map[string]Contribution),
		exportHooks:      make(map[string]ExportHook),
		eraseHooks:       make(map[string]EraseHook),
		protocolHandlers: make(map[string]Contribution),
		auditor:          &MemoryAuditor{},
	}
}

// SetAuditor replaces the audit sink.
func (r *Registry) SetAuditor(auditor Auditor) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.auditor = auditor
	r.mu.Unlock()
}

// SetPermissionCheck installs Host RBAC gate.
func (r *Registry) SetPermissionCheck(check PermissionCheck) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.allow = check
	r.mu.Unlock()
}

// Register installs a contribution and optional process-local hooks.
func (r *Registry) Register(c Contribution, export ExportHook, erase EraseHook) error {
	if r == nil {
		return ErrInvalid
	}
	c.ExtensionID = strings.ToLower(strings.TrimSpace(c.ExtensionID))
	if c.ExtensionID == "" || len(c.Inventory) == 0 {
		return ErrInvalid
	}
	for i := range c.Inventory {
		c.Inventory[i].ID = strings.TrimSpace(c.Inventory[i].ID)
		c.Inventory[i].Kind = strings.ToLower(strings.TrimSpace(c.Inventory[i].Kind))
		if c.Inventory[i].ID == "" || c.Inventory[i].Kind == "" {
			return ErrInvalid
		}
	}
	if export != nil {
		c.SupportsExport = true
	}
	if erase != nil {
		c.SupportsErase = true
	}
	r.mu.Lock()
	r.contributions[c.ExtensionID] = c
	if export != nil {
		r.exportHooks[c.ExtensionID] = export
	}
	if erase != nil {
		r.eraseHooks[c.ExtensionID] = erase
	}
	r.mu.Unlock()
	return nil
}

// PublishContribution registers inventory from lifecycle publication / Protocol V2
// without a process-local Go callback (hooks may arrive later via protocol).
func (r *Registry) PublishContribution(c Contribution) error {
	if r == nil {
		return ErrInvalid
	}
	c.ExtensionID = strings.ToLower(strings.TrimSpace(c.ExtensionID))
	if c.ExtensionID == "" || len(c.Inventory) == 0 {
		return ErrInvalid
	}
	r.mu.Lock()
	r.protocolHandlers[c.ExtensionID] = c
	// Merge inventory into contributions for Inventory() visibility.
	if existing, ok := r.contributions[c.ExtensionID]; ok {
		existing.Inventory = append(existing.Inventory, c.Inventory...)
		existing.PackageDigest = c.PackageDigest
		r.contributions[c.ExtensionID] = existing
	} else {
		r.contributions[c.ExtensionID] = c
	}
	r.mu.Unlock()
	return nil
}

// Inventory lists all registered inventory items.
func (r *Registry) Inventory() []InventoryItem {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]InventoryItem, 0)
	for _, c := range r.contributions {
		out = append(out, c.Inventory...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ExportUser runs all export hooks with permission, deadline, and partial failure.
func (r *Registry) ExportUser(ctx context.Context, actor, userID string) (ExportBundle, error) {
	if r == nil {
		return ExportBundle{}, ErrInvalid
	}
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(userID) == "" {
		return ExportBundle{}, ErrPermissionDenied
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := ensureDeadline(ctx, DefaultDeadline)
	defer cancel()

	if err := r.checkPermission(ctx, actor, userID, "export"); err != nil {
		return ExportBundle{}, err
	}

	r.mu.Lock()
	hooks := make(map[string]ExportHook, len(r.exportHooks))
	for k, v := range r.exportHooks {
		hooks[k] = v
	}
	inventory := r.inventoryLocked()
	auditor := r.auditor
	r.mu.Unlock()

	bundle := ExportBundle{
		SchemaVersion: SchemaVersion, UserID: userID,
		GeneratedAt: time.Now().UTC(), Inventory: inventory,
	}
	ids := make([]string, 0, len(hooks))
	for id := range hooks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			bundle.Partial = true
			bundle.Failures = append(bundle.Failures, id+": deadline")
			_ = r.appendAudit(ctx, auditor, "export", actor, userID, "partial", "deadline")
			return bundle, fmt.Errorf("%w: %v", ErrDeadline, err)
		}
		artifact, err := hooks[id](ctx, userID)
		if err != nil {
			bundle.Partial = true
			bundle.Failures = append(bundle.Failures, id+": "+err.Error())
			continue
		}
		artifact.ExtensionID = id
		if artifact.MediaType == "" {
			artifact.MediaType = "application/json"
		}
		bundle.Artifacts = append(bundle.Artifacts, artifact)
	}
	status := "ok"
	if bundle.Partial {
		status = "partial"
	}
	_ = r.appendAudit(ctx, auditor, "export", actor, userID, status, strings.Join(bundle.Failures, ";"))
	if bundle.Partial && len(bundle.Artifacts) == 0 {
		return bundle, ErrHookFailed
	}
	if bundle.Partial {
		return bundle, ErrPartial
	}
	return bundle, nil
}

// EraseUser runs all erase hooks and aggregates external retention warnings.
func (r *Registry) EraseUser(ctx context.Context, actor, userID string) (EraseReport, error) {
	if r == nil {
		return EraseReport{}, ErrInvalid
	}
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(userID) == "" {
		return EraseReport{}, ErrPermissionDenied
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := ensureDeadline(ctx, DefaultDeadline)
	defer cancel()

	if err := r.checkPermission(ctx, actor, userID, "erase"); err != nil {
		return EraseReport{}, err
	}

	r.mu.Lock()
	hooks := make(map[string]EraseHook, len(r.eraseHooks))
	for k, v := range r.eraseHooks {
		hooks[k] = v
	}
	var externalFromInventory []string
	for _, c := range r.contributions {
		for _, item := range c.Inventory {
			if item.External || item.Kind == KindExternal {
				externalFromInventory = append(externalFromInventory,
					c.ExtensionID+":"+item.ID)
			}
		}
	}
	auditor := r.auditor
	r.mu.Unlock()

	report := EraseReport{SchemaVersion: SchemaVersion, UserID: userID, At: time.Now().UTC()}
	ids := make([]string, 0, len(hooks))
	for id := range hooks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			report.Partial = true
			report.Failures = append(report.Failures, id+": deadline")
			_ = r.appendAudit(ctx, auditor, "erase", actor, userID, "partial", "deadline")
			return report, fmt.Errorf("%w: %v", ErrDeadline, err)
		}
		result, err := hooks[id](ctx, userID)
		if err != nil {
			report.Partial = true
			report.Failures = append(report.Failures, id+": "+err.Error())
			report.Results = append(report.Results, EraseResult{
				ExtensionID: id, Erased: false, Error: err.Error(),
			})
			continue
		}
		result.ExtensionID = id
		report.Results = append(report.Results, result)
		report.RetainedExternal = append(report.RetainedExternal, result.RetainedExternal...)
	}
	report.RetainedExternal = append(report.RetainedExternal, externalFromInventory...)
	report.RetainedExternal = uniqueStrings(report.RetainedExternal)
	status := "ok"
	if report.Partial {
		status = "partial"
	}
	_ = r.appendAudit(ctx, auditor, "erase", actor, userID, status, strings.Join(report.Failures, ";"))
	if report.Partial {
		return report, ErrPartial
	}
	return report, nil
}

func (r *Registry) checkPermission(ctx context.Context, actor, userID, operation string) error {
	r.mu.Lock()
	allow := r.allow
	r.mu.Unlock()
	if allow == nil {
		return nil
	}
	return allow(ctx, actor, userID, operation)
}

func (r *Registry) appendAudit(ctx context.Context, auditor Auditor, op, actor, userID, status, detail string) error {
	if auditor == nil {
		return nil
	}
	return auditor.Append(ctx, AuditEvent{
		AuditID:   fmt.Sprintf("%s-%d", op, time.Now().UTC().UnixNano()),
		Operation: op, Actor: actor, UserID: userID, Status: status, Detail: detail,
		At: time.Now().UTC(),
	})
}

func ensureDeadline(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, d)
}

func (r *Registry) inventoryLocked() []InventoryItem {
	out := make([]InventoryItem, 0)
	for _, c := range r.contributions {
		out = append(out, c.Inventory...)
	}
	return out
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

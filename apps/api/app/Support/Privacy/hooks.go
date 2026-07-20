// Package privacy is the Host privacy export/erase registry for V3 P12.
// Plugins declare inventory and hooks; Host orchestrates export/erase and
// surfaces retained external-resource warnings.
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
)

var (
	ErrInvalid          = errors.New("privacy input is invalid")
	ErrNotFound         = errors.New("privacy contribution is not found")
	ErrPermissionDenied = errors.New("privacy permission denied")
	ErrHookFailed       = errors.New("privacy hook failed")
)

// Contribution is one plugin's privacy surface declaration.
type Contribution struct {
	ExtensionID   string   `json:"extensionId"`
	PackageDigest string   `json:"packageDigest,omitempty"`
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
}

// ExportBundle is the full user export.
type ExportBundle struct {
	SchemaVersion string           `json:"schemaVersion"`
	UserID        string           `json:"userId"`
	GeneratedAt   time.Time        `json:"generatedAt"`
	Artifacts     []ExportArtifact `json:"artifacts"`
	Inventory     []InventoryItem  `json:"inventory"`
}

// EraseReport is the full erase report for operators and the user.
type EraseReport struct {
	SchemaVersion    string        `json:"schemaVersion"`
	UserID           string        `json:"userId"`
	At               time.Time     `json:"at"`
	Results          []EraseResult `json:"results"`
	RetainedExternal []string      `json:"retainedExternal,omitempty"`
}

// Registry holds privacy contributions and hooks.
type Registry struct {
	mu            sync.Mutex
	contributions map[string]Contribution
	exportHooks   map[string]ExportHook
	eraseHooks    map[string]EraseHook
}

// New builds an empty privacy registry.
func New() *Registry {
	return &Registry{
		contributions: make(map[string]Contribution),
		exportHooks:   make(map[string]ExportHook),
		eraseHooks:    make(map[string]EraseHook),
	}
}

// Register installs a contribution and optional hooks.
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

// ExportUser runs all export hooks for a user.
func (r *Registry) ExportUser(ctx context.Context, actor, userID string) (ExportBundle, error) {
	if r == nil {
		return ExportBundle{}, ErrInvalid
	}
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(userID) == "" {
		return ExportBundle{}, ErrPermissionDenied
	}
	r.mu.Lock()
	hooks := make(map[string]ExportHook, len(r.exportHooks))
	for k, v := range r.exportHooks {
		hooks[k] = v
	}
	inventory := r.inventoryLocked()
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
		artifact, err := hooks[id](ctx, userID)
		if err != nil {
			return ExportBundle{}, fmt.Errorf("%w: %s: %v", ErrHookFailed, id, err)
		}
		artifact.ExtensionID = id
		if artifact.MediaType == "" {
			artifact.MediaType = "application/json"
		}
		bundle.Artifacts = append(bundle.Artifacts, artifact)
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
	r.mu.Lock()
	hooks := make(map[string]EraseHook, len(r.eraseHooks))
	for k, v := range r.eraseHooks {
		hooks[k] = v
	}
	// Also warn about inventory marked external even without erase hook.
	var externalFromInventory []string
	for _, c := range r.contributions {
		for _, item := range c.Inventory {
			if item.External || item.Kind == KindExternal {
				externalFromInventory = append(externalFromInventory,
					c.ExtensionID+":"+item.ID)
			}
		}
	}
	r.mu.Unlock()

	report := EraseReport{SchemaVersion: SchemaVersion, UserID: userID, At: time.Now().UTC()}
	ids := make([]string, 0, len(hooks))
	for id := range hooks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		result, err := hooks[id](ctx, userID)
		if err != nil {
			return EraseReport{}, fmt.Errorf("%w: %s: %v", ErrHookFailed, id, err)
		}
		result.ExtensionID = id
		report.Results = append(report.Results, result)
		report.RetainedExternal = append(report.RetainedExternal, result.RetainedExternal...)
	}
	report.RetainedExternal = append(report.RetainedExternal, externalFromInventory...)
	report.RetainedExternal = uniqueStrings(report.RetainedExternal)
	return report, nil
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

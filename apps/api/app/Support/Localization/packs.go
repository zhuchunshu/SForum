package localization

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// PackStatus tracks language pack enablement.
const (
	PackStatusInstalled = "installed"
	PackStatusEnabled   = "enabled"
	PackStatusDisabled  = "disabled"
)

// LanguagePack is a Host-registered translation pack (plugin or operator upload).
type LanguagePack struct {
	ID           string            `json:"id"`
	Domain       Domain            `json:"domain"`
	Version      string            `json:"version"`
	Locales      []string          `json:"locales"`
	Status       string            `json:"status"`
	// Messages is locale -> key -> text for backend domains.
	Messages     map[string]map[string]string `json:"-"`
	InstalledAt  time.Time         `json:"installedAt,omitempty"`
	EnabledAt    time.Time         `json:"enabledAt,omitempty"`
	ExtensionID  string            `json:"extensionId,omitempty"`
	PackageDigest string           `json:"packageDigest,omitempty"`
}

// PackRegistry manages pack install/enable against a Catalog.
type PackRegistry struct {
	mu      sync.Mutex
	catalog *Catalog
	packs   map[string]LanguagePack
}

// NewPackRegistry binds packs to a catalog.
func NewPackRegistry(catalog *Catalog) *PackRegistry {
	if catalog == nil {
		catalog = NewCatalog()
	}
	return &PackRegistry{catalog: catalog, packs: make(map[string]LanguagePack)}
}

// Install stores a pack without enabling its messages.
func (r *PackRegistry) Install(pack LanguagePack) error {
	if r == nil {
		return errCatalogInvalid
	}
	pack.ID = strings.ToLower(strings.TrimSpace(pack.ID))
	pack.Domain = Domain(strings.ToLower(strings.TrimSpace(string(pack.Domain))))
	if pack.ID == "" || pack.Domain == "" || pack.Version == "" {
		return errCatalogInvalid
	}
	if pack.Status == "" {
		pack.Status = PackStatusInstalled
	}
	if pack.InstalledAt.IsZero() {
		pack.InstalledAt = time.Now().UTC()
	}
	// Normalize locales.
	locales := make([]string, 0, len(pack.Locales))
	seen := map[string]bool{}
	for _, locale := range pack.Locales {
		locale = Normalize(locale, nil)
		if locale == "" || seen[locale] {
			continue
		}
		seen[locale] = true
		locales = append(locales, locale)
	}
	pack.Locales = locales
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.packs[pack.ID]; ok && existing.Status == PackStatusEnabled {
		return fmt.Errorf("%w: disable pack before reinstall", errCatalogInvalid)
	}
	r.packs[pack.ID] = clonePack(pack)
	return nil
}

// Enable publishes pack messages into the catalog (strict collision).
func (r *PackRegistry) Enable(packID string) error {
	if r == nil {
		return errCatalogInvalid
	}
	packID = strings.ToLower(strings.TrimSpace(packID))
	r.mu.Lock()
	pack, ok := r.packs[packID]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("%w: pack not found", errCatalogInvalid)
	}
	if pack.Status == PackStatusEnabled {
		r.mu.Unlock()
		return nil
	}
	// Register messages with collision detection.
	for locale, entries := range pack.Messages {
		for key, text := range entries {
			if err := r.catalog.Register(pack.Domain, locale, key, text, true); err != nil {
				r.mu.Unlock()
				return err
			}
		}
	}
	pack.Status = PackStatusEnabled
	pack.EnabledAt = time.Now().UTC()
	r.packs[packID] = pack
	r.mu.Unlock()
	return nil
}

// Disable marks a pack disabled. Messages remain until process restart unless
// the Host rebuilds the catalog from enabled packs only.
func (r *PackRegistry) Disable(packID string) error {
	if r == nil {
		return errCatalogInvalid
	}
	packID = strings.ToLower(strings.TrimSpace(packID))
	r.mu.Lock()
	defer r.mu.Unlock()
	pack, ok := r.packs[packID]
	if !ok {
		return fmt.Errorf("%w: pack not found", errCatalogInvalid)
	}
	pack.Status = PackStatusDisabled
	pack.EnabledAt = time.Time{}
	r.packs[packID] = pack
	return nil
}

// List returns installed packs (metadata only).
func (r *PackRegistry) List() []LanguagePack {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]LanguagePack, 0, len(r.packs))
	for _, pack := range r.packs {
		out = append(out, clonePack(pack))
	}
	return out
}

// Get returns one pack by id.
func (r *PackRegistry) Get(packID string) (LanguagePack, bool) {
	if r == nil {
		return LanguagePack{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	pack, ok := r.packs[strings.ToLower(strings.TrimSpace(packID))]
	if !ok {
		return LanguagePack{}, false
	}
	return clonePack(pack), true
}

func clonePack(pack LanguagePack) LanguagePack {
	out := pack
	if len(pack.Locales) > 0 {
		out.Locales = append([]string(nil), pack.Locales...)
	}
	if pack.Messages != nil {
		out.Messages = make(map[string]map[string]string, len(pack.Messages))
		for locale, entries := range pack.Messages {
			cp := make(map[string]string, len(entries))
			for k, v := range entries {
				cp[k] = v
			}
			out.Messages[locale] = cp
		}
	}
	return out
}

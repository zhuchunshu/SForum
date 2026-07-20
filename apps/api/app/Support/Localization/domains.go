package localization

import (
	"fmt"
	"strings"
	"sync"
)

// Domain is a translation namespace (core, plugin id, or pack domain).
// Keys are always looked up as domain-local; collision across domains is allowed.
type Domain string

const (
	// DomainCore is the built-in API envelope catalog.
	DomainCore Domain = "core"
)

// Catalog is a process-local multi-domain message store with locale fallback.
type Catalog struct {
	mu       sync.RWMutex
	messages map[Domain]map[string]map[string]string // domain -> locale -> key -> text
	// overrides holds controlled Host/operator overrides (highest priority).
	overrides map[Domain]map[string]map[string]string
}

// NewCatalog builds an empty catalog. Callers typically SeedCore().
func NewCatalog() *Catalog {
	return &Catalog{
		messages:  make(map[Domain]map[string]map[string]string),
		overrides: make(map[Domain]map[string]map[string]string),
	}
}

// SeedCore loads the built-in envelope messages under DomainCore.
func (c *Catalog) SeedCore() {
	if c == nil {
		return
	}
	for locale, entries := range messages {
		for key, text := range entries {
			_ = c.Register(DomainCore, locale, key, text, false)
		}
	}
}

// Register adds a message. If conflictStrict and the key already exists in the
// same domain+locale with a different text, ErrCatalogCollision is returned.
func (c *Catalog) Register(domain Domain, locale, key, text string, conflictStrict bool) error {
	if c == nil {
		return errCatalogInvalid
	}
	domain = Domain(strings.ToLower(strings.TrimSpace(string(domain))))
	locale = Normalize(locale, nil)
	key = strings.TrimSpace(key)
	if domain == "" || key == "" || strings.TrimSpace(text) == "" {
		return errCatalogInvalid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	byLocale := c.messages[domain]
	if byLocale == nil {
		byLocale = make(map[string]map[string]string)
		c.messages[domain] = byLocale
	}
	byKey := byLocale[locale]
	if byKey == nil {
		byKey = make(map[string]string)
		byLocale[locale] = byKey
	}
	if existing, ok := byKey[key]; ok && conflictStrict && existing != text {
		return fmt.Errorf("%w: %s/%s/%s", ErrCatalogCollision, domain, locale, key)
	}
	byKey[key] = text
	return nil
}

// SetOverride installs a controlled Host override (wins over packs and core).
func (c *Catalog) SetOverride(domain Domain, locale, key, text string) error {
	if c == nil {
		return errCatalogInvalid
	}
	domain = Domain(strings.ToLower(strings.TrimSpace(string(domain))))
	locale = Normalize(locale, nil)
	key = strings.TrimSpace(key)
	if domain == "" || key == "" {
		return errCatalogInvalid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	byLocale := c.overrides[domain]
	if byLocale == nil {
		byLocale = make(map[string]map[string]string)
		c.overrides[domain] = byLocale
	}
	byKey := byLocale[locale]
	if byKey == nil {
		byKey = make(map[string]string)
		byLocale[locale] = byKey
	}
	byKey[key] = text
	return nil
}

// T resolves domain.locale.key with fallback: override → locale → DefaultLocale → key.
func (c *Catalog) T(domain Domain, locale, key string) string {
	if c == nil {
		return Message(locale, key)
	}
	domain = Domain(strings.ToLower(strings.TrimSpace(string(domain))))
	if domain == "" {
		domain = DomainCore
	}
	locale = Normalize(locale, nil)
	key = strings.TrimSpace(key)
	c.mu.RLock()
	defer c.mu.RUnlock()
	if text, ok := lookup(c.overrides, domain, locale, key); ok {
		return text
	}
	if text, ok := lookup(c.messages, domain, locale, key); ok {
		return text
	}
	// DomainCore also falls through to the package-level Message helper for
	// callers that never SeedCore.
	if domain == DomainCore {
		return Message(locale, key)
	}
	return key
}

// TP resolves a pluralized message. count selects one/other (and zh zero/other).
// Message templates may use {count} and {n} placeholders.
func (c *Catalog) TP(domain Domain, locale, key string, count int) string {
	form := PluralForm(locale, count)
	// Prefer key.form then key.
	candidates := []string{key + "." + form, key}
	for _, candidate := range candidates {
		text := c.T(domain, locale, candidate)
		if text != candidate {
			return strings.NewReplacer(
				"{count}", fmt.Sprintf("%d", count),
				"{n}", fmt.Sprintf("%d", count),
			).Replace(text)
		}
	}
	return key
}

func lookup(store map[Domain]map[string]map[string]string, domain Domain, locale, key string) (string, bool) {
	byLocale := store[domain]
	if byLocale == nil {
		return "", false
	}
	if byKey := byLocale[locale]; byKey != nil {
		if text, ok := byKey[key]; ok {
			return text, true
		}
	}
	if locale != DefaultLocale {
		if byKey := byLocale[DefaultLocale]; byKey != nil {
			if text, ok := byKey[key]; ok {
				return text, true
			}
		}
	}
	return "", false
}

var (
	ErrCatalogCollision = fmt.Errorf("localization catalog collision")
	errCatalogInvalid   = fmt.Errorf("localization catalog input is invalid")
)

// DefaultCatalog is the process-wide domain catalog (seeded lazily).
var (
	defaultCatalogOnce sync.Once
	defaultCatalog     *Catalog
)

// Default returns the process catalog with core messages seeded.
func Default() *Catalog {
	defaultCatalogOnce.Do(func() {
		defaultCatalog = NewCatalog()
		defaultCatalog.SeedCore()
	})
	return defaultCatalog
}

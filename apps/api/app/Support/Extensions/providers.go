package extensionsruntime

import "sync"

type ProviderSelection struct {
	ExtensionID string
	Label       string
}

type ProviderRegistry struct {
	mu         sync.RWMutex
	defaults   map[string]ProviderSelection
	selections map[string]ProviderSelection
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		defaults: map[string]ProviderSelection{
			"search.provider":             {Label: "Built-in search"},
			"attachment.storage.provider": {Label: "Built-in attachment storage"},
			"human_verification.provider": {Label: "Built-in human verification"},
			"auth.risk.provider":          {Label: "Built-in auth risk checks"},
			"editor.sanitizer.provider":   {Label: "Built-in sanitizer"},
		},
		selections: map[string]ProviderSelection{},
	}
}

func (r *ProviderRegistry) Selected(slot string) ProviderSelection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if selected, ok := r.selections[slot]; ok {
		return selected
	}
	return r.defaults[slot]
}

func (r *ProviderRegistry) Select(slot string, selection ProviderSelection) {
	r.mu.Lock()
	r.selections[slot] = selection
	r.mu.Unlock()
}

func (r *ProviderRegistry) RestoreDefault(slot string) {
	r.mu.Lock()
	delete(r.selections, slot)
	r.mu.Unlock()
}

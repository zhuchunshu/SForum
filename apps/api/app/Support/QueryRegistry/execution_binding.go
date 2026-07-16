package queryregistry

// BoundToRegistry lets Host bootstrap prove that an execution outlet and its
// contract resolver share one immutable Registry owner. It exposes no plan or
// provider authority and exists only to prevent split-brain wiring.
func (r *ExecutionRuntime) BoundToRegistry(registry *Registry) bool {
	return r != nil && registry != nil && r.registry == registry
}

// NormalizeExecutionContext exposes the same locale/scope rules used by Plan
// so Host protocol adapters cannot sign a context that planning will reject.
func NormalizeExecutionContext(locale, scope string) (string, string, error) {
	normalizedLocale, err := normalizeLocale(locale)
	if err != nil {
		return "", "", err
	}
	normalizedScope, err := normalizeScope(scope)
	if err != nil {
		return "", "", err
	}
	return normalizedLocale, normalizedScope, nil
}

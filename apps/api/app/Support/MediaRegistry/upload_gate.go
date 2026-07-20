package mediaregistry

import (
	"path/filepath"
	"strings"
)

// CheckUploadMIME applies the selected Media Registry MIME policy when one is
// published for purpose (or "*"). When no policy is published, returns nil so
// Host AttachmentSettings remain the sole gate — empty Media Registry must not
// break core uploads.
//
// contentType is the server-sniffed MIME; extension is the lower-case file
// extension without a leading dot (or with; both accepted).
func (r *Registry) CheckUploadMIME(purpose, contentType, extension string) error {
	if r == nil {
		return nil
	}
	mime, err := normalizeExactMIME(contentType)
	if err != nil {
		return ErrInvalid
	}
	state := r.load()
	policy, found := selectPolicy(state, strings.TrimSpace(purpose))
	if !found {
		return nil
	}
	if matchesAnyMIME(policy.DeniedMIMEs, mime) || !matchesAnyMIME(policy.AllowedMIMEs, mime) {
		return ErrMediaRejected
	}
	ext := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(extension)), ".")
	if ext == "" {
		ext = strings.TrimPrefix(strings.ToLower(filepath.Ext(extension)), ".")
	}
	if len(policy.AllowedExtensions) > 0 && ext != "" && !containsString(policy.AllowedExtensions, ext) {
		return ErrMediaRejected
	}
	return nil
}

// Package contentsecurity documents Host-final rich-content and attachment
// security boundaries for the Trusted Plugin Platform V3 content plane.
//
// Authority lives in existing packages:
//   - EditorDocument.SanitizeHTML — editor storage HTML is Host-sanitized
//   - ContentRegistry execution sanitize — plugin render/filter HTML is re-sanitized
//   - options.IsAttachmentActiveContentType — active MIME types cannot inline
//   - Attachments controller Content-Disposition — response-layer XSS fallback
//
// Plugins may contribute content/media/editor declarations but must never weaken
// these Host-final policies. Tests in this package prove the joined product gate.
package contentsecurity

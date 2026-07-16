package extensions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// ErrStoredTrustImpactInvalid marks a stored impact_document that fails
// production canonical digest integrity or required field presence.
var ErrStoredTrustImpactInvalid = errors.New("extensions: stored trust impact is invalid")

var trustImpactDigestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ValidateStoredTrustImpact decodes a durable impact_document into the typed
// TrustImpact schema and recomputes its canonical digest with the same
// algorithm used at grant time. expectedDigest is the grant column value.
//
// Both the embedded document digest field and the recomputed digest must equal
// expectedDigest. This rejects subtree-correct forgeries that only rewrite
// top-level action/package/schema fields without updating the digest column.
//
// Decoding uses DisallowUnknownFields and requires exactly one JSON value so
// extra keys (or trailing payloads) that would not participate in the typed
// canonical digest cannot pass integrity.
//
// Production Identity Registry adoption receives this function as an
// instance-scoped PostgresStore dependency (never a package global).
func ValidateStoredTrustImpact(document []byte, expectedDigest string) error {
	expectedDigest = strings.ToLower(strings.TrimSpace(expectedDigest))
	if len(document) == 0 || !trustImpactDigestPattern.MatchString(expectedDigest) {
		return ErrStoredTrustImpactInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()
	var impact TrustImpact
	if err := decoder.Decode(&impact); err != nil {
		return fmt.Errorf("%w: decode impact document: %v", ErrStoredTrustImpactInvalid, err)
	}
	// Exactly one JSON value: any trailing token is fail-closed.
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("%w: trailing JSON after impact document", ErrStoredTrustImpactInvalid)
		}
		return fmt.Errorf("%w: decode impact document: %v", ErrStoredTrustImpactInvalid, err)
	}
	if impact.Digest != expectedDigest {
		return fmt.Errorf("%w: document digest does not match grant impact_digest", ErrStoredTrustImpactInvalid)
	}
	recomputed, err := canonicalTrustImpactDigest(impact)
	if err != nil {
		return fmt.Errorf("%w: recompute digest: %v", ErrStoredTrustImpactInvalid, err)
	}
	if recomputed != expectedDigest {
		return fmt.Errorf("%w: canonical digest mismatch", ErrStoredTrustImpactInvalid)
	}
	return nil
}

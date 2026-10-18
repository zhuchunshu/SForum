// Package timezone owns IANA timezone resolution for host-side code.
//
// The blank import below embeds the IANA database into every binary that links
// this package. Runtime images (alpine, scratch, distroless) do not ship
// /usr/share/zoneinfo, and the API/worker binaries are built with
// CGO_ENABLED=0, so without this import time.LoadLocation rejects every
// non-UTC zone. That previously made "Site Settings -> timezone" unsaveable in
// containers (HTTP 422 options.invalid) and rejected plugin schedule timezones.
// Do not remove it.
package timezone

import (
	"strings"
	"time"
	_ "time/tzdata"
)

// Load resolves an IANA timezone name, trimming surrounding whitespace.
// Unknown names return an error. An empty name resolves to UTC, matching
// time.LoadLocation; callers that require an explicit zone must reject empty
// input themselves.
func Load(name string) (*time.Location, error) {
	return time.LoadLocation(strings.TrimSpace(name))
}

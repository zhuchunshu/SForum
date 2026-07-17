package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const routeRuntimeIncidentsMigration = "202607180035_extension_route_runtime_incidents.sql"

func TestRouteRuntimeIncidentsMigrationKeepsExactPayloadFreeEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), routeRuntimeIncidentsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("route runtime incidents migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE extension_route_runtime_incidents",
		"incident_key TEXT NOT NULL UNIQUE",
		"route_revision BIGINT NOT NULL",
		"invocation_stage TEXT NOT NULL",
		"mode TEXT NOT NULL",
		"failure_code TEXT NOT NULL",
		"cause_class TEXT NOT NULL",
		"runtime_execution_observed BOOLEAN NOT NULL",
		"extension_id TEXT NOT NULL",
		"extension_version_id BIGINT NOT NULL",
		"extension_version TEXT NOT NULL",
		"package_digest TEXT NOT NULL",
		"runtime_instance_id TEXT NOT NULL",
		"audit_event_id BIGINT NOT NULL UNIQUE",
		"local_quarantine_result TEXT NOT NULL DEFAULT 'pending'",
		"resolved_at TIMESTAMPTZ",
		"CREATE INDEX extension_route_runtime_incidents_artifact_idx",
		"CREATE INDEX extension_route_runtime_incidents_route_idx",
		"CREATE INDEX extension_route_runtime_incidents_pending_idx",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("route runtime incidents migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"request_body", "response_body", "raw_error", "headers JSON", "query JSON",
		"REFERENCES extensions", "REFERENCES extension_versions", "REFERENCES audit_events",
		"ON DELETE CASCADE", "force_cancel",
	} {
		if strings.Contains(strings.ToLower(up), strings.ToLower(forbidden)) {
			t.Fatalf("route runtime incidents migration must not use %q", forbidden)
		}
	}
}

func TestRouteRuntimeIncidentsMigrationDownFailsClosed(t *testing.T) {
	body, err := fs.ReadFile(Files(), routeRuntimeIncidentsMigration)
	if err != nil {
		t.Fatal(err)
	}
	down := strings.Join(strings.Fields(strings.SplitN(string(body), "-- +goose Down", 2)[1]), " ")
	for _, clause := range []string{
		"IF EXISTS (SELECT 1 FROM extension_route_runtime_incidents)",
		"RAISE EXCEPTION 'cannot remove route runtime incident evidence'",
		"DROP TABLE IF EXISTS extension_route_runtime_incidents",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("route runtime incidents migration Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("route runtime incidents migration Down contains %q", forbidden)
		}
	}
}

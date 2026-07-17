package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const routeRuntimeIncidentImmutabilityMigration = "202607180036_route_runtime_incident_immutability.sql"

func TestRouteRuntimeIncidentImmutabilityAllowsOnlyOneLocalResolution(t *testing.T) {
	body, err := fs.ReadFile(Files(), routeRuntimeIncidentImmutabilityMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("route runtime incident immutability migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"RENAME COLUMN quarantine_result TO local_quarantine_result",
		"ADD COLUMN IF NOT EXISTS incident_key TEXT",
		"ADD COLUMN IF NOT EXISTS extension_version_id BIGINT",
		"ADD COLUMN IF NOT EXISTS audit_event_id BIGINT",
		"legacy force-cancel route incidents require manual review",
		"version.package_digest = incident.package_digest",
		"'routes.runtime_incident'",
		"'legacyRepair', true",
		"ALTER COLUMN local_quarantine_result SET DEFAULT 'pending'",
		"extension_route_runtime_incidents_state_check",
		"CREATE INDEX IF NOT EXISTS extension_route_runtime_incidents_pending_idx",
		"CHECK (resolved_at IS NULL OR resolved_at >= created_at)",
		"CHECK (invocation_stage <> 'response' OR response_status IS NOT NULL)",
		"CREATE FUNCTION validate_route_runtime_incident_audit()",
		"metadata ->> 'incidentKey'",
		"stored_action IS DISTINCT FROM 'routes.runtime_incident'",
		"BEFORE INSERT ON extension_route_runtime_incidents",
		"CREATE FUNCTION enforce_route_runtime_incident_immutability()",
		"TG_OP IN ('DELETE', 'TRUNCATE')",
		"OLD.local_quarantine_result <> 'pending'",
		"NEW.local_quarantine_result = 'pending'",
		"to_jsonb(NEW) - ARRAY['local_quarantine_result', 'resolved_at']",
		"BEFORE UPDATE OR DELETE ON extension_route_runtime_incidents",
		"BEFORE TRUNCATE ON extension_route_runtime_incidents",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("route runtime incident immutability migration missing %q", clause)
		}
	}
}

func TestRouteRuntimeIncidentImmutabilityDownFailsClosed(t *testing.T) {
	body, err := fs.ReadFile(Files(), routeRuntimeIncidentImmutabilityMigration)
	if err != nil {
		t.Fatal(err)
	}
	down := strings.Join(strings.Fields(strings.SplitN(string(body), "-- +goose Down", 2)[1]), " ")
	for _, clause := range []string{
		"LOCK TABLE extension_route_runtime_incidents IN ACCESS EXCLUSIVE MODE",
		"IF EXISTS (SELECT 1 FROM extension_route_runtime_incidents)",
		"RAISE EXCEPTION 'cannot remove route runtime incident immutability'",
		"DROP TRIGGER IF EXISTS extension_route_runtime_incident_no_truncate",
		"DROP TRIGGER IF EXISTS extension_route_runtime_incident_resolve_once",
		"DROP FUNCTION IF EXISTS enforce_route_runtime_incident_immutability()",
		"DROP TRIGGER IF EXISTS extension_route_runtime_incident_audit_valid",
		"DROP FUNCTION IF EXISTS validate_route_runtime_incident_audit()",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("route runtime incident immutability Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE extension_route_runtime_incidents"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("route runtime incident immutability Down contains %q", forbidden)
		}
	}
}

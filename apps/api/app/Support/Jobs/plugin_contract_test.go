package jobs

import (
	"slices"
	"testing"
)

func TestDecidePluginJobExecution(t *testing.T) {
	source := pluginJobContractFixture("1.0.0", "digest-v1", "demo.payload", "1")
	target := pluginJobContractFixture("2.0.0", "digest-v2", "demo.payload", "2")

	if got := DecidePluginJobExecution(source, source, nil); got.Action != PluginJobExecute || got.Reason != PluginJobReasonExactMatch {
		t.Fatalf("exact decision = %#v", got)
	}
	if got := DecidePluginJobExecution(source, target, nil); got.Action != PluginJobCancel || got.Reason != PluginJobReasonIncompatible {
		t.Fatalf("incompatible decision = %#v", got)
	}
	migrations := []PluginJobMigration{
		{ID: "migration-z", From: source, To: target},
		{ID: "migration-a", From: source, To: target},
	}
	if got := DecidePluginJobExecution(source, target, migrations); got.Action != PluginJobMigrate || got.MigrationID != "migration-a" {
		t.Fatalf("migration decision = %#v", got)
	}
}

func TestDecidePluginJobUpgrade(t *testing.T) {
	source := pluginJobContractFixture("1.0.0", "digest-v1", "demo.payload", "1")
	target := pluginJobContractFixture("2.0.0", "digest-v2", "demo.payload", "2")

	if got := DecidePluginJobUpgrade(PluginJobUpgrade{Queued: target, Source: source, Target: target}); got.Action != PluginJobExecute {
		t.Fatalf("target decision = %#v", got)
	}
	if got := DecidePluginJobUpgrade(PluginJobUpgrade{Queued: source, Source: source, Target: target, SourceRuntimeAvailable: true}); got.Action != PluginJobDrain {
		t.Fatalf("drain decision = %#v", got)
	}
	if got := DecidePluginJobUpgrade(PluginJobUpgrade{
		Queued: source, Source: source, Target: target,
		Migrations: []PluginJobMigration{{ID: "payload-v2", From: source, To: target}},
	}); got.Action != PluginJobMigrate || got.MigrationID != "payload-v2" {
		t.Fatalf("migrate decision = %#v", got)
	}
	if got := DecidePluginJobUpgrade(PluginJobUpgrade{Queued: source, Source: source, Target: target}); got.Action != PluginJobCancel {
		t.Fatalf("cancel decision = %#v", got)
	}
}

func TestPluginJobDecisionIsDeterministic(t *testing.T) {
	source := pluginJobContractFixture("1.0.0", "digest-v1", "demo.payload", "1")
	target := pluginJobContractFixture("2.0.0", "digest-v2", "demo.payload", "2")
	migrations := []PluginJobMigration{
		{ID: "migration-z", From: source, To: target},
		{ID: "migration-a", From: source, To: target},
	}
	want := DecidePluginJobExecution(source, target, migrations)
	for range 100 {
		slices.Reverse(migrations)
		if got := DecidePluginJobExecution(source, target, migrations); got != want {
			t.Fatalf("nondeterministic decision: got %#v want %#v", got, want)
		}
	}
}

func TestSplitVersionedSchema(t *testing.T) {
	id, version, ok := SplitVersionedSchema("demo.payload@12")
	if !ok || id != "demo.payload" || version != "12" {
		t.Fatalf("split = %q %q %t", id, version, ok)
	}
	for _, invalid := range []string{"", "demo.payload", "@1", "demo.payload@"} {
		if _, _, ok := SplitVersionedSchema(invalid); ok {
			t.Fatalf("accepted invalid schema %q", invalid)
		}
	}
}

func pluginJobContractFixture(version, digest, schema, schemaVersion string) PluginJobContract {
	return PluginJobContract{
		ExtensionID: "demo.plugin", ExtensionVersion: version, ArtifactDigest: digest,
		JobName: "demo.sync", JobContract: "demo.job.sync@1",
		PayloadSchemaID: schema, PayloadSchemaVers: schemaVersion,
	}
}

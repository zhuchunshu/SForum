package jobs

import (
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type pluginPeriodicBundleStub struct {
	adds      int
	failOnAdd int
	removed   []string
}

func (b *pluginPeriodicBundleStub) AddSafely(*river.PeriodicJob) (rivertype.PeriodicJobHandle, error) {
	b.adds++
	if b.failOnAdd > 0 && b.adds == b.failOnAdd {
		return 0, errors.New("add failed")
	}
	return rivertype.PeriodicJobHandle(b.adds), nil
}

func (b *pluginPeriodicBundleStub) RemoveByID(id string) bool {
	b.removed = append(b.removed, id)
	return true
}

func TestPluginSchedulePeriodicPublisherBindsActiveAndRemovesOnDrain(t *testing.T) {
	registry := NewPluginScheduleAdmissionRegistry()
	runtime := exactPluginScheduleRuntime("instance-a", "1.0.0", "digest-a")
	if _, err := registry.PublishActive(runtime); err != nil {
		t.Fatal(err)
	}
	bundle := &pluginPeriodicBundleStub{}
	if err := registry.BindPeriodicPublisher(NewPluginSchedulePeriodicPublisher(bundle)); err != nil {
		t.Fatal(err)
	}
	if bundle.adds != 1 {
		t.Fatalf("periodic adds = %d", bundle.adds)
	}
	if _, err := registry.BeginDrain(runtime.Identity); err != nil {
		t.Fatal(err)
	}
	if len(bundle.removed) != 1 || bundle.removed[0] != "plugin_schedule.demo.plugin.schedule.sync" {
		t.Fatalf("removed = %#v", bundle.removed)
	}
	if _, err := registry.PublishActive(runtime); err != nil {
		t.Fatal(err)
	}
	if bundle.adds != 2 {
		t.Fatalf("rollback republish adds = %d", bundle.adds)
	}
}

func TestPluginSchedulePeriodicPublisherReplacesExactRuntimeAndRollsBackFailure(t *testing.T) {
	bundle := &pluginPeriodicBundleStub{}
	publisher := NewPluginSchedulePeriodicPublisher(bundle)
	oldRuntime := exactPluginScheduleRuntime("instance-a", "1.0.0", "digest-a")
	newRuntime := exactPluginScheduleRuntime("instance-b", "2.0.0", "digest-b")
	if err := publisher.Replace(nil, oldRuntime); err != nil {
		t.Fatal(err)
	}
	if err := publisher.Replace(&oldRuntime, newRuntime); err != nil {
		t.Fatal(err)
	}
	if bundle.adds != 2 || len(bundle.removed) != 1 {
		t.Fatalf("replace adds=%d removed=%#v", bundle.adds, bundle.removed)
	}

	failing := &pluginPeriodicBundleStub{failOnAdd: 2}
	publisher = NewPluginSchedulePeriodicPublisher(failing)
	if err := publisher.Replace(nil, oldRuntime); err != nil {
		t.Fatal(err)
	}
	if err := publisher.Replace(&oldRuntime, newRuntime); !errors.Is(err, ErrPluginScheduleInvalid) {
		t.Fatalf("replace failure = %v", err)
	}
	// Failed replacement removes the old entry, rejects the new entry, then
	// restores the previous exact runtime definition.
	if failing.adds != 3 || len(failing.removed) != 1 {
		t.Fatalf("rollback adds=%d removed=%#v", failing.adds, failing.removed)
	}
}

func TestPluginPeriodicScheduleParsesCronTimezoneAndInterval(t *testing.T) {
	schedule, err := pluginPeriodicSchedule("0 3 * * *", "Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	current := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	next := schedule.Next(current)
	if next.Location().String() != "Asia/Shanghai" || next.Hour() != 3 {
		t.Fatalf("next cron = %s", next)
	}
	if _, err := pluginPeriodicSchedule("@every 5m", "UTC"); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{"bad cron", "61 * * * *", "@every 100ms"} {
		if _, err := pluginPeriodicSchedule(invalid, "UTC"); err == nil {
			t.Fatalf("accepted invalid expression %q", invalid)
		}
	}
}

func exactPluginScheduleRuntime(instanceID, version, digest string) PluginScheduleRuntime {
	contract := PluginJobContract{
		ExtensionID: "demo.plugin", ExtensionVersion: version, ArtifactDigest: digest,
		JobName: "demo.sync", JobContract: "demo.plugin.job.sync@1",
		PayloadSchemaID: "demo.plugin.sync.payload", PayloadSchemaVersion: "1",
	}.Normalized()
	return PluginScheduleRuntime{
		Identity: PluginScheduleRuntimeIdentity{
			ExtensionID: "demo.plugin", ExtensionVersion: version,
			ArtifactDigest: digest, InstanceID: instanceID,
		},
		Schedules: []PluginScheduleDeclaration{{
			ScheduleID: "demo.plugin.schedule.sync", JobName: contract.JobName,
			JobContract: contract.JobContract, Cron: "0 3 * * *", Timezone: "UTC",
			Contract: contract, TrustGrantID: "grant-" + version,
		}},
	}
}

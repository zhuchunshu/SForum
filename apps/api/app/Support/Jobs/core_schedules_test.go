package jobs

import (
	"testing"
	"time"

	"github.com/riverqueue/river"
)

func TestCoreScheduleRegistryBuildsCorePeriodics(t *testing.T) {
	reg, err := NewCoreScheduleRegistry(map[string]river.PeriodicJobConstructor{
		ScheduleIdentityCleanupSessions: func() (river.JobArgs, *river.InsertOpts) {
			return stubArgs{kind: ScheduleIdentityCleanupSessions}, nil
		},
		ScheduleExtensionWebReleaseCleanup: func() (river.JobArgs, *river.InsertOpts) {
			return stubArgs{kind: ScheduleExtensionWebReleaseCleanup}, nil
		},
		ScheduleAttachmentsCleanupOrphans: func() (river.JobArgs, *river.InsertOpts) {
			return stubArgs{kind: ScheduleAttachmentsCleanupOrphans}, nil
		},
		ScheduleAuditCleanupEvents: func() (river.JobArgs, *river.InsertOpts) {
			return stubArgs{kind: ScheduleAuditCleanupEvents}, nil
		},
		ScheduleForumAutoLockIdle: func() (river.JobArgs, *river.InsertOpts) {
			return stubArgs{kind: ScheduleForumAutoLockIdle}, nil
		},
	})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if reg.Len() != 5 {
		t.Fatalf("expected 5 core schedules, got %d", reg.Len())
	}
	jobs, err := reg.BuildPeriodicJobs()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(jobs) != 5 {
		t.Fatalf("expected 5 river periodics, got %d", len(jobs))
	}

	// 元数据完整性：daily、owner 明确
	views := reg.Views()
	byID := map[string]ScheduleView{}
	for _, v := range views {
		byID[v.ID] = v
	}
	for _, id := range []string{
		ScheduleIdentityCleanupSessions,
		ScheduleExtensionWebReleaseCleanup,
		ScheduleAttachmentsCleanupOrphans,
		ScheduleAuditCleanupEvents,
		ScheduleForumAutoLockIdle,
	} {
		v, ok := byID[id]
		if !ok {
			t.Fatalf("missing schedule %s", id)
		}
		if !v.Enabled {
			t.Fatalf("%s should be enabled", id)
		}
		if v.IntervalSeconds != int64((24 * time.Hour) / time.Second) {
			t.Fatalf("%s interval=%d", id, v.IntervalSeconds)
		}
		if v.JobKind == "" || v.Owner == "" {
			t.Fatalf("%s incomplete: %+v", id, v)
		}
	}
	if byID[ScheduleIdentityCleanupSessions].Queue != QueueDefault {
		t.Fatalf("session cleanup queue=%s", byID[ScheduleIdentityCleanupSessions].Queue)
	}
	if byID[ScheduleAttachmentsCleanupOrphans].Queue != QueueMaintenance {
		t.Fatalf("orphan cleanup queue=%s", byID[ScheduleAttachmentsCleanupOrphans].Queue)
	}
}

func TestCoreScheduleRegistryWithoutConstructorsIsCatalogOnly(t *testing.T) {
	reg, err := NewCoreScheduleRegistry(nil)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	jobs, err := reg.BuildPeriodicJobs()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("catalog-only should not build periodics, got %d", len(jobs))
	}
	if len(reg.Views()) != 5 {
		t.Fatalf("views=%d", len(reg.Views()))
	}
}

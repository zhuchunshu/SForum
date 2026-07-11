package jobs

import (
	"testing"
	"time"

	"github.com/riverqueue/river"
)

type stubArgs struct {
	kind string
}

func (a stubArgs) Kind() string { return a.kind }

func TestScheduleRegistryRejectsInvalidAndDuplicate(t *testing.T) {
	r := NewScheduleRegistry()
	if err := r.Register(ScheduleDefinition{}); err == nil {
		t.Fatal("expected empty id error")
	}
	if err := r.Register(ScheduleDefinition{ID: "a"}); err == nil {
		t.Fatal("expected missing kind error")
	}
	if err := r.Register(ScheduleDefinition{ID: "a", JobKind: "a.kind"}); err == nil {
		t.Fatal("expected missing interval/cron error")
	}
	def := ScheduleDefinition{
		ID:       "a",
		JobKind:  "a.kind",
		Interval: time.Hour,
		Enabled:  true,
	}
	if err := r.Register(def); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := r.Register(def); err == nil {
		t.Fatal("expected duplicate id error")
	}
	if r.Len() != 1 {
		t.Fatalf("len=%d", r.Len())
	}
}

func TestScheduleRegistryBuildPeriodicJobsSkipsDisabledAndNilConstructor(t *testing.T) {
	r := NewScheduleRegistry()
	mustRegister := func(def ScheduleDefinition) {
		t.Helper()
		if err := r.Register(def); err != nil {
			t.Fatalf("register %s: %v", def.ID, err)
		}
	}
	mustRegister(ScheduleDefinition{
		ID:       "enabled",
		JobKind:  "demo.enabled",
		Queue:    QueueMaintenance,
		Interval: 15 * time.Minute,
		Owner:    "demo",
		Enabled:  true,
		Constructor: func() (river.JobArgs, *river.InsertOpts) {
			return stubArgs{kind: "demo.enabled"}, nil
		},
	})
	mustRegister(ScheduleDefinition{
		ID:       "disabled",
		JobKind:  "demo.disabled",
		Interval: time.Hour,
		Enabled:  false,
		Constructor: func() (river.JobArgs, *river.InsertOpts) {
			return stubArgs{kind: "demo.disabled"}, nil
		},
	})
	mustRegister(ScheduleDefinition{
		ID:       "catalog-only",
		JobKind:  "demo.catalog",
		Interval: time.Hour,
		Enabled:  true,
		// Constructor nil：仅出现在 catalog，不进入 River
	})

	jobs, err := r.BuildPeriodicJobs()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 periodic job, got %d", len(jobs))
	}

	views := r.Views()
	if len(views) != 3 {
		t.Fatalf("expected 3 views, got %d", len(views))
	}
	if views[0].ID != "enabled" || views[0].IntervalSeconds != 900 {
		t.Fatalf("unexpected first view: %+v", views[0])
	}
	if views[1].Enabled {
		t.Fatal("disabled schedule should remain visible but enabled=false")
	}
}

func TestScheduleRegistryBuildRejectsCronInF1(t *testing.T) {
	r := NewScheduleRegistry()
	if err := r.Register(ScheduleDefinition{
		ID:      "cronny",
		JobKind: "demo.cron",
		Cron:    "0 * * * *",
		Enabled: true,
		Constructor: func() (river.JobArgs, *river.InsertOpts) {
			return stubArgs{kind: "demo.cron"}, nil
		},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	_, err := r.BuildPeriodicJobs()
	if err == nil {
		t.Fatal("expected cron unsupported error")
	}
	if got := err.Error(); got == "" {
		t.Fatal("expected non-empty error")
	}
}

func TestScheduleConstructorFillsQueue(t *testing.T) {
	r := NewScheduleRegistry()
	if err := r.Register(ScheduleDefinition{
		ID:       "q",
		JobKind:  "demo.q",
		Queue:    QueueMaintenance,
		Interval: time.Hour,
		Enabled:  true,
		Constructor: func() (river.JobArgs, *river.InsertOpts) {
			return stubArgs{kind: "demo.q"}, nil
		},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	jobs, err := r.BuildPeriodicJobs()
	if err != nil || len(jobs) != 1 {
		t.Fatalf("build: jobs=%d err=%v", len(jobs), err)
	}
	// River 不导出 constructor；通过 wrap 行为用公开 View + 间接注册成功断言即可。
	// 额外验证 wrapConstructor 本身。
	def := r.Definitions()[0]
	args, opts := def.wrapConstructor()()
	if args.Kind() != "demo.q" {
		t.Fatalf("kind=%s", args.Kind())
	}
	if opts == nil || opts.Queue != QueueMaintenance {
		t.Fatalf("expected queue filled, opts=%+v", opts)
	}
}

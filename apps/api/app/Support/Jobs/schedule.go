package jobs

import (
	"fmt"
	"sort"
	"time"

	"github.com/riverqueue/river"
)

// ScheduleDefinition 描述一条平台拥有的周期任务目录项。
// River 负责实际触发；本结构是 SForum 的 schedule catalog，供 bootstrap 注册与 admin 只读展示。
// 插件不得自行启动 goroutine cron；后续插件 schedule 也须经本 registry 声明。
type ScheduleDefinition struct {
	// ID 稳定唯一键，例如 "identity.cleanup_sessions"。
	ID string
	// JobKind 对应 River job kind（通常与 Args.Kind() 一致）。
	JobKind string
	// Queue 期望队列名；BuildPeriodicJobs 会写入 InsertOpts（若 Constructor 未自带 opts）。
	Queue string
	// Interval 固定间隔；与 Cron 二选一，F1 仅要求 interval。
	Interval time.Duration
	// Cron 可选 cron 表达式；F1 未实现解析，非空且 Interval==0 时 Build 报错。
	Cron string
	// Owner 归属模块或扩展 id，例如 "identity" / "attachments" / "extensions"。
	Owner string
	// Enabled 为 false 时不进入 River PeriodicJobs，但仍出现在 catalog 列表中。
	Enabled bool
	// Description 运维可读说明（中文优先，与产品文案一致即可）。
	Description string
	// RunOnStart 是否在 worker 启动时立即插入一次（透传 River PeriodicJobOpts）。
	RunOnStart bool
	// Constructor 触发时构造 job args；不得阻塞。仅 worker 进程需要非 nil。
	Constructor river.PeriodicJobConstructor
}

// ScheduleRegistry 是进程内 schedule catalog。
// 新维护任务应 Register 一条定义，而不是在 bootstrap 里散落 NewPeriodicJob。
type ScheduleRegistry struct {
	byID  map[string]ScheduleDefinition
	order []string
}

func NewScheduleRegistry() *ScheduleRegistry {
	return &ScheduleRegistry{byID: make(map[string]ScheduleDefinition)}
}

// Register 登记一条 schedule。ID 必填且不可重复；JobKind 必填。
func (r *ScheduleRegistry) Register(def ScheduleDefinition) error {
	if r == nil {
		return fmt.Errorf("schedule registry is nil")
	}
	if def.ID == "" {
		return fmt.Errorf("schedule id is required")
	}
	if def.JobKind == "" {
		return fmt.Errorf("schedule %q: job kind is required", def.ID)
	}
	if def.Interval <= 0 && def.Cron == "" {
		return fmt.Errorf("schedule %q: interval or cron is required", def.ID)
	}
	if _, exists := r.byID[def.ID]; exists {
		return fmt.Errorf("schedule %q already registered", def.ID)
	}
	r.byID[def.ID] = def
	r.order = append(r.order, def.ID)
	return nil
}

// Len 返回已登记条目数（含 disabled）。
func (r *ScheduleRegistry) Len() int {
	if r == nil {
		return 0
	}
	return len(r.order)
}

// Definitions 返回按登记顺序的副本，供 admin / 测试只读遍历。
func (r *ScheduleRegistry) Definitions() []ScheduleDefinition {
	if r == nil || len(r.order) == 0 {
		return nil
	}
	out := make([]ScheduleDefinition, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.byID[id])
	}
	return out
}

// SortedDefinitions 返回按 ID 排序的副本，便于稳定测试与 OpenAPI 示例。
func (r *ScheduleRegistry) SortedDefinitions() []ScheduleDefinition {
	out := r.Definitions()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// BuildPeriodicJobs 将 enabled 且带 Constructor 的定义转为 River PeriodicJob。
// disabled 或 Constructor==nil 的条目跳过（catalog-only 进程可用）。
func (r *ScheduleRegistry) BuildPeriodicJobs() ([]*river.PeriodicJob, error) {
	if r == nil {
		return nil, nil
	}
	var jobs []*river.PeriodicJob
	for _, id := range r.order {
		def := r.byID[id]
		if !def.Enabled || def.Constructor == nil {
			continue
		}
		schedule, err := def.periodicSchedule()
		if err != nil {
			return nil, err
		}
		constructor := def.wrapConstructor()
		opts := &river.PeriodicJobOpts{
			ID:         def.ID,
			RunOnStart: def.RunOnStart,
		}
		jobs = append(jobs, river.NewPeriodicJob(schedule, constructor, opts))
	}
	return jobs, nil
}

func (def ScheduleDefinition) periodicSchedule() (river.PeriodicSchedule, error) {
	if def.Interval > 0 {
		return river.PeriodicInterval(def.Interval), nil
	}
	if def.Cron != "" {
		// F1 仅迁移 interval 任务；cron 解析留给后续 wave，避免半吊子第三方 cron 绑定。
		return nil, fmt.Errorf("schedule %q: cron schedules are not supported in F1", def.ID)
	}
	return nil, fmt.Errorf("schedule %q: no schedule configured", def.ID)
}

// wrapConstructor 在 Constructor 未提供 InsertOpts 时，用定义上的 Queue 补齐。
func (def ScheduleDefinition) wrapConstructor() river.PeriodicJobConstructor {
	inner := def.Constructor
	queue := def.Queue
	return func() (river.JobArgs, *river.InsertOpts) {
		args, opts := inner()
		if args == nil {
			return nil, nil
		}
		if opts == nil && queue != "" {
			opts = &river.InsertOpts{Queue: queue}
		} else if opts != nil && opts.Queue == "" && queue != "" {
			// 复制避免修改调用方可能复用的 opts 指针。
			copied := *opts
			copied.Queue = queue
			opts = &copied
		}
		return args, opts
	}
}

// ScheduleView 是 admin / API 使用的只读投影（不含 Constructor）。
type ScheduleView struct {
	ID               string `json:"id"`
	JobKind          string `json:"jobKind"`
	Queue            string `json:"queue"`
	IntervalSeconds  int64  `json:"intervalSeconds,omitempty"`
	Cron             string `json:"cron,omitempty"`
	Owner            string `json:"owner"`
	Enabled          bool   `json:"enabled"`
	Description      string `json:"description"`
	RunOnStart       bool   `json:"runOnStart"`
}

// Views 将 registry 投影为稳定 JSON 视图（按登记顺序）。
func (r *ScheduleRegistry) Views() []ScheduleView {
	defs := r.Definitions()
	if len(defs) == 0 {
		return []ScheduleView{}
	}
	out := make([]ScheduleView, 0, len(defs))
	for _, def := range defs {
		out = append(out, def.View())
	}
	return out
}

func (def ScheduleDefinition) View() ScheduleView {
	view := ScheduleView{
		ID:          def.ID,
		JobKind:     def.JobKind,
		Queue:       def.Queue,
		Cron:        def.Cron,
		Owner:       def.Owner,
		Enabled:     def.Enabled,
		Description: def.Description,
		RunOnStart:  def.RunOnStart,
	}
	if def.Interval > 0 {
		view.IntervalSeconds = int64(def.Interval / time.Second)
	}
	return view
}

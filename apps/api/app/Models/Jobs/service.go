package jobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

var ErrNotFound = errors.New("jobs: not found")

type Service struct {
	pool   *pgxpool.Pool
	client *supportjobs.Client
}

func NewService(pool *pgxpool.Pool, client *supportjobs.Client) *Service {
	return &Service{pool: pool, client: client}
}

func (s *Service) Overview(ctx context.Context, actor identity.Actor) (Overview, error) {
	if !actor.Can(identity.PermissionJobsView) {
		return Overview{}, identity.ErrPermissionDenied
	}
	rows, err := s.pool.Query(ctx, `SELECT state::text, count(*) FROM river_job GROUP BY state`)
	if err != nil {
		return Overview{}, err
	}
	defer rows.Close()
	result := Overview{Counts: map[string]int64{}}
	for rows.Next() {
		var state string
		var count int64
		if err := rows.Scan(&state, &count); err != nil {
			return Overview{}, err
		}
		result.Counts[state] = count
	}
	queues, err := s.queues(ctx)
	if err != nil {
		return Overview{}, err
	}
	result.Queues = queues
	return result, rows.Err()
}

func (s *Service) List(ctx context.Context, actor identity.Actor, input ListInput) ([]Job, error) {
	if !actor.Can(identity.PermissionJobsView) {
		return nil, identity.ErrPermissionDenied
	}
	limit := input.Limit
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	params := river.NewJobListParams().First(limit).OrderBy(river.JobListOrderByID, river.SortOrderDesc)
	if input.Queue != "" {
		params = params.Queues(input.Queue)
	}
	if input.Kind != "" {
		params = params.Kinds(input.Kind)
	}
	if input.State != "" {
		params = params.States(rivertype.JobState(input.State))
	}
	result, err := s.client.JobList(ctx, params)
	if err != nil {
		return nil, err
	}
	items := make([]Job, len(result.Jobs))
	for i, row := range result.Jobs {
		items[i] = mapJob(row)
	}
	return items, nil
}

func (s *Service) Detail(ctx context.Context, actor identity.Actor, id int64) (Job, error) {
	if !actor.Can(identity.PermissionJobsView) {
		return Job{}, identity.ErrPermissionDenied
	}
	row, err := s.client.JobGet(ctx, id)
	if errors.Is(err, rivertype.ErrNotFound) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, err
	}
	return mapJob(row), nil
}

func (s *Service) Retry(ctx context.Context, actor identity.Actor, id int64) (Job, error) {
	if !actor.Can(identity.PermissionJobsManage) {
		return Job{}, identity.ErrPermissionDenied
	}
	row, err := s.client.JobRetry(ctx, id)
	if errors.Is(err, rivertype.ErrNotFound) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, err
	}
	return mapJob(row), nil
}

func (s *Service) Cancel(ctx context.Context, actor identity.Actor, id int64) (Job, error) {
	if !actor.Can(identity.PermissionJobsManage) {
		return Job{}, identity.ErrPermissionDenied
	}
	row, err := s.client.JobCancel(ctx, id)
	if errors.Is(err, rivertype.ErrNotFound) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, err
	}
	return mapJob(row), nil
}

func (s *Service) SetQueuePaused(ctx context.Context, actor identity.Actor, name string, paused bool) error {
	if !actor.Can(identity.PermissionJobsManage) {
		return identity.ErrPermissionDenied
	}
	if name == "" {
		return fmt.Errorf("queue name is required")
	}
	if paused {
		return s.client.QueuePause(ctx, name, nil)
	}
	return s.client.QueueResume(ctx, name, nil)
}

// Schedules 返回宿主 schedule catalog（只读）。F1 使用内置 CoreScheduleDefinitions；
// 插件声明的 schedules 留到 F2 能力模型之后。
func (s *Service) Schedules(ctx context.Context, actor identity.Actor) ([]Schedule, error) {
	if !actor.Can(identity.PermissionJobsView) {
		return nil, identity.ErrPermissionDenied
	}
	_ = ctx
	defs := supportjobs.CoreScheduleDefinitions()
	out := make([]Schedule, 0, len(defs))
	for _, def := range defs {
		view := def.View()
		out = append(out, Schedule{
			ID:              view.ID,
			JobKind:         view.JobKind,
			Queue:           view.Queue,
			IntervalSeconds: view.IntervalSeconds,
			Cron:            view.Cron,
			Owner:           view.Owner,
			Enabled:         view.Enabled,
			Description:     view.Description,
			RunOnStart:      view.RunOnStart,
		})
	}
	return out, nil
}

func (s *Service) queues(ctx context.Context) ([]Queue, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT q.name, q.paused_at, q.updated_at,
		 count(j.id) FILTER (WHERE j.state IN ('available','scheduled','retryable')),
		 count(j.id) FILTER (WHERE j.state = 'running'),
		 count(j.id) FILTER (WHERE j.state = 'discarded')
		FROM river_queue q LEFT JOIN river_job j ON j.queue = q.name
		GROUP BY q.name, q.paused_at, q.updated_at ORDER BY q.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Queue{}
	for rows.Next() {
		var item Queue
		if err := rows.Scan(&item.Name, &item.PausedAt, &item.UpdatedAt, &item.Available, &item.Running, &item.Failed); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func mapJob(row *rivertype.JobRow) Job {
	return Job{ID: row.ID, Kind: row.Kind, Queue: row.Queue, State: string(row.State), Attempt: row.Attempt, MaxAttempts: row.MaxAttempts, Priority: row.Priority, Args: append([]byte(nil), row.EncodedArgs...), Metadata: append([]byte(nil), row.Metadata...), Tags: row.Tags, Errors: row.Errors, AttemptedBy: row.AttemptedBy, CreatedAt: row.CreatedAt, ScheduledAt: row.ScheduledAt, AttemptedAt: row.AttemptedAt, FinalizedAt: row.FinalizedAt}
}

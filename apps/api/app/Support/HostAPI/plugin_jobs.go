package hostapi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/riverqueue/river"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

// PluginJobKind 是 River 上的宿主统一 job kind；真实插件 kind 放在 payload 内。
const PluginJobKind = "extension.plugin_job"

// PluginJobArgs 入队载荷：仅允许插件 manifest 声明过的 JobName。
type PluginJobArgs struct {
	ExtensionID string         `json:"extensionId"`
	JobName     string         `json:"kind"`
	Payload     map[string]any `json:"payload,omitempty"`
	EnqueuedAt  time.Time      `json:"enqueuedAt"`
}

// Kind 返回 River 注册用的固定 kind（非插件声明的 JobName）。
func (PluginJobArgs) Kind() string { return PluginJobKind }

// RiverJobEnqueuer 把插件 job 写入 River default 队列。
type RiverJobEnqueuer struct {
	Dispatcher *supportjobs.Dispatcher
}

func (e *RiverJobEnqueuer) EnqueuePluginJob(ctx context.Context, extensionID, kind string, payload map[string]any) error {
	if e == nil || e.Dispatcher == nil {
		return fmt.Errorf("%w: job dispatcher missing", ErrUnavailable)
	}
	args := PluginJobArgs{
		ExtensionID: extensionID,
		JobName:     kind,
		Payload:     payload,
		EnqueuedAt:  time.Now().UTC(),
	}
	_, err := e.Dispatcher.Enqueue(ctx, args, supportjobs.EnqueueOptions{
		Queue: supportjobs.QueueDefault,
	})
	return err
}

// PluginJobWorker 是 F2 最小 worker：校验载荷后成功结束。
// 后续波次可转发到插件 RPC InvokeJob。
type PluginJobWorker struct {
	river.WorkerDefaults[PluginJobArgs]
}

func (w *PluginJobWorker) Work(_ context.Context, job *river.Job[PluginJobArgs]) error {
	if job.Args.ExtensionID == "" || job.Args.JobName == "" {
		return fmt.Errorf("plugin job missing extensionId or kind")
	}
	// 保留 payload 编码校验，避免坏 JSON 静默入队。
	if job.Args.Payload != nil {
		if _, err := json.Marshal(job.Args.Payload); err != nil {
			return err
		}
	}
	return nil
}

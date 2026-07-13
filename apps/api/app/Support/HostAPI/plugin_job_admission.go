package hostapi

import (
	"context"
	"errors"
	"strings"
)

var (
	// ErrPluginJobEnqueueStale 表示调用者已不是当前活动的 exact runtime。
	ErrPluginJobEnqueueStale = errors.New("hostapi: plugin job enqueue runtime is stale")
	// ErrPluginJobEnqueueDraining 表示 runtime 已关闭新的作业入队入口。
	ErrPluginJobEnqueueDraining = errors.New("hostapi: plugin job enqueue runtime is draining")
	// ErrPluginJobEnqueueUnavailable 表示宿主未能执行 runtime admission。
	ErrPluginJobEnqueueUnavailable = errors.New("hostapi: plugin job enqueue admission is unavailable")
)

// PluginJobEnqueueIdentity 是协议认证后的 exact runtime 身份。
// HostAPI 仅拥有最小契约，避免反向依赖 Extensions runtime 实现。
type PluginJobEnqueueIdentity struct {
	ExtensionID      string
	ExtensionVersion string
	ArtifactDigest   string
	InstanceID       string
}

func (i PluginJobEnqueueIdentity) valid() bool {
	return strings.TrimSpace(i.ExtensionID) != "" &&
		strings.TrimSpace(i.ExtensionVersion) != "" &&
		strings.TrimSpace(i.ArtifactDigest) != "" &&
		strings.TrimSpace(i.InstanceID) != ""
}

// PluginJobEnqueueLease 保证 admission 与最终 River INSERT 处于同一 drain 边界。
type PluginJobEnqueueLease interface {
	Context() context.Context
	Release()
}

// PluginJobEnqueueLeaseFailure 允许生产 adapter 将 runtime 自身的强制
// drain 与普通调用者取消区分开；测试或第三方实现无需实现该可选接口。
type PluginJobEnqueueLeaseFailure interface {
	PluginJobEnqueueFailure() error
}

type PluginJobEnqueueAdmission interface {
	AcquirePluginJobEnqueue(context.Context, PluginJobEnqueueIdentity) (PluginJobEnqueueLease, error)
}

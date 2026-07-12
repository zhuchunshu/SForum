package extensionsruntime

import (
	"context"
	"errors"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

// StorageProviderInvoker 是 ProtocolStarter / 测试替身对存储 RPC 的宿主入口。
// Manager 在闸门/熔断之后调用此接口。
type StorageProviderInvoker interface {
	StoragePutBegin(ctx context.Context, extensionID string, request StoragePutBeginRequest) (StorageSessionResponse, error)
	StoragePutChunk(ctx context.Context, extensionID string, request StoragePutChunkRequest) (StorageResult, error)
	StorageOpen(ctx context.Context, extensionID string, request StorageOpenRequest) (StorageSessionResponse, error)
	StorageGetChunk(ctx context.Context, extensionID string, request StorageGetChunkRequest) (StorageGetChunkResponse, error)
	StorageClose(ctx context.Context, extensionID string, request StorageCloseRequest) (StorageResult, error)
	StorageDelete(ctx context.Context, extensionID string, request StorageObjectRequest) (StorageResult, error)
	StorageStat(ctx context.Context, extensionID string, request StorageStatRequest) (StorageStatResponse, error)
	StorageExists(ctx context.Context, extensionID string, request StorageExistsRequest) (StorageExistsResponse, error)
	StoragePublicURL(ctx context.Context, extensionID string, request StoragePublicURLRequest) (StorageURLResponse, error)
	StorageSignedURL(ctx context.Context, extensionID string, request StorageSignedURLRequest) (StorageURLResponse, error)
	StorageProbe(ctx context.Context, extensionID string, request StorageProbeRequest) (StorageProbeResponse, error)
}

// StorageRuntime 是 Attachments / PluginStorageAdapter 依赖的最小 runtime 面。
// Manager 实现本接口；bootstrap 把 extensionRuntime 注入附件服务。
type StorageRuntime interface {
	StoragePutBegin(ctx context.Context, extensionID string, request StoragePutBeginRequest) (StorageSessionResponse, error)
	StoragePutChunk(ctx context.Context, extensionID string, request StoragePutChunkRequest) (StorageResult, error)
	StorageOpen(ctx context.Context, extensionID string, request StorageOpenRequest) (StorageSessionResponse, error)
	StorageGetChunk(ctx context.Context, extensionID string, request StorageGetChunkRequest) (StorageGetChunkResponse, error)
	StorageClose(ctx context.Context, extensionID string, request StorageCloseRequest) (StorageResult, error)
	StorageDelete(ctx context.Context, extensionID string, request StorageObjectRequest) (StorageResult, error)
	StorageStat(ctx context.Context, extensionID string, request StorageStatRequest) (StorageStatResponse, error)
	StorageExists(ctx context.Context, extensionID string, request StorageExistsRequest) (StorageExistsResponse, error)
	StoragePublicURL(ctx context.Context, extensionID string, request StoragePublicURLRequest) (StorageURLResponse, error)
	StorageSignedURL(ctx context.Context, extensionID string, request StorageSignedURLRequest) (StorageURLResponse, error)
	StorageProbe(ctx context.Context, extensionID string, request StorageProbeRequest) (StorageProbeResponse, error)
}

// storageCall 在运行中的扩展上执行一次存储 RPC，并走 F2.3 闸门/熔断。
// success 由调用方根据响应 OK（及传输 err）判定。
func (m *Manager) storageCall(
	ctx context.Context,
	extensionID string,
	fn func(ctx context.Context, invoker StorageProviderInvoker) (ok bool, reason string, err error),
) error {
	invoker, ok := m.starter.(StorageProviderInvoker)
	if !ok {
		return extensions.ErrRuntimeUnavailable
	}
	m.mu.RLock()
	_, running := m.running[extensionID]
	m.mu.RUnlock()
	if !running {
		return extensions.ErrRuntimeUnavailable
	}

	timeout := m.resilience.cfg.DefaultStorageTimeout
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	release, rejected := m.resilience.tryEnter(ctx, extensionID)
	if rejected != "" {
		return storageRejectedError(rejected)
	}
	success, reason, err := fn(ctx, invoker)
	if err != nil {
		reason = "extension.storage_failed"
		success = false
	}
	if ctx.Err() != nil && !success {
		reason = "extension.hook_timeout"
	}
	release(success, reason)
	return err
}

func storageRejectedError(reason string) error {
	switch reason {
	case "extension.circuit_open":
		return ErrStorageCircuitOpen
	case "extension.hook_timeout":
		return ErrStorageTimeout
	default:
		return extensions.ErrRuntimeUnavailable
	}
}

// 宿主侧可识别的存储 RPC 拒绝（PluginStorageAdapter 映射为 fail-closed）。
var (
	ErrStorageCircuitOpen = errors.New("extension.circuit_open")
	ErrStorageTimeout     = errors.New("extension.hook_timeout")
)

// StorageRPCError 把插件返回的 !OK 转成带 Reason 的错误，供 Adapter 使用。
type StorageRPCError struct {
	Reason  string
	Message string
}

func (e *StorageRPCError) Error() string {
	if e == nil {
		return "storage rpc failed"
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Reason != "" {
		return e.Reason
	}
	return "storage rpc failed"
}

func storageRPCErr(reason, message string) error {
	if reason == "" {
		reason = "extension.storage_failed"
	}
	return &StorageRPCError{Reason: reason, Message: message}
}

func (m *Manager) StoragePutBegin(ctx context.Context, extensionID string, request StoragePutBeginRequest) (StorageSessionResponse, error) {
	var response StorageSessionResponse
	err := m.storageCall(ctx, extensionID, func(ctx context.Context, invoker StorageProviderInvoker) (bool, string, error) {
		resp, err := invoker.StoragePutBegin(ctx, extensionID, request)
		response = resp
		if err != nil {
			return false, "", err
		}
		return resp.OK, resp.Reason, nil
	})
	return response, err
}

func (m *Manager) StoragePutChunk(ctx context.Context, extensionID string, request StoragePutChunkRequest) (StorageResult, error) {
	var response StorageResult
	err := m.storageCall(ctx, extensionID, func(ctx context.Context, invoker StorageProviderInvoker) (bool, string, error) {
		resp, err := invoker.StoragePutChunk(ctx, extensionID, request)
		response = resp
		if err != nil {
			return false, "", err
		}
		return resp.OK, resp.Reason, nil
	})
	return response, err
}

func (m *Manager) StorageOpen(ctx context.Context, extensionID string, request StorageOpenRequest) (StorageSessionResponse, error) {
	var response StorageSessionResponse
	err := m.storageCall(ctx, extensionID, func(ctx context.Context, invoker StorageProviderInvoker) (bool, string, error) {
		resp, err := invoker.StorageOpen(ctx, extensionID, request)
		response = resp
		if err != nil {
			return false, "", err
		}
		return resp.OK, resp.Reason, nil
	})
	return response, err
}

func (m *Manager) StorageGetChunk(ctx context.Context, extensionID string, request StorageGetChunkRequest) (StorageGetChunkResponse, error) {
	var response StorageGetChunkResponse
	err := m.storageCall(ctx, extensionID, func(ctx context.Context, invoker StorageProviderInvoker) (bool, string, error) {
		resp, err := invoker.StorageGetChunk(ctx, extensionID, request)
		response = resp
		if err != nil {
			return false, "", err
		}
		return resp.OK, resp.Reason, nil
	})
	return response, err
}

func (m *Manager) StorageClose(ctx context.Context, extensionID string, request StorageCloseRequest) (StorageResult, error) {
	// Close 失败不计入熔断（尽力清理会话）。
	invoker, ok := m.starter.(StorageProviderInvoker)
	if !ok {
		return StorageResult{}, extensions.ErrRuntimeUnavailable
	}
	m.mu.RLock()
	_, running := m.running[extensionID]
	m.mu.RUnlock()
	if !running {
		return StorageResult{}, extensions.ErrRuntimeUnavailable
	}
	return invoker.StorageClose(ctx, extensionID, request)
}

func (m *Manager) StorageDelete(ctx context.Context, extensionID string, request StorageObjectRequest) (StorageResult, error) {
	var response StorageResult
	err := m.storageCall(ctx, extensionID, func(ctx context.Context, invoker StorageProviderInvoker) (bool, string, error) {
		resp, err := invoker.StorageDelete(ctx, extensionID, request)
		response = resp
		if err != nil {
			return false, "", err
		}
		return resp.OK, resp.Reason, nil
	})
	return response, err
}

func (m *Manager) StorageStat(ctx context.Context, extensionID string, request StorageStatRequest) (StorageStatResponse, error) {
	var response StorageStatResponse
	err := m.storageCall(ctx, extensionID, func(ctx context.Context, invoker StorageProviderInvoker) (bool, string, error) {
		resp, err := invoker.StorageStat(ctx, extensionID, request)
		response = resp
		if err != nil {
			return false, "", err
		}
		return resp.OK, resp.Reason, nil
	})
	return response, err
}

func (m *Manager) StorageExists(ctx context.Context, extensionID string, request StorageExistsRequest) (StorageExistsResponse, error) {
	var response StorageExistsResponse
	err := m.storageCall(ctx, extensionID, func(ctx context.Context, invoker StorageProviderInvoker) (bool, string, error) {
		resp, err := invoker.StorageExists(ctx, extensionID, request)
		response = resp
		if err != nil {
			return false, "", err
		}
		return resp.OK, resp.Reason, nil
	})
	return response, err
}

func (m *Manager) StoragePublicURL(ctx context.Context, extensionID string, request StoragePublicURLRequest) (StorageURLResponse, error) {
	var response StorageURLResponse
	err := m.storageCall(ctx, extensionID, func(ctx context.Context, invoker StorageProviderInvoker) (bool, string, error) {
		resp, err := invoker.StoragePublicURL(ctx, extensionID, request)
		response = resp
		if err != nil {
			return false, "", err
		}
		return resp.OK, resp.Reason, nil
	})
	return response, err
}

func (m *Manager) StorageSignedURL(ctx context.Context, extensionID string, request StorageSignedURLRequest) (StorageURLResponse, error) {
	var response StorageURLResponse
	err := m.storageCall(ctx, extensionID, func(ctx context.Context, invoker StorageProviderInvoker) (bool, string, error) {
		resp, err := invoker.StorageSignedURL(ctx, extensionID, request)
		response = resp
		if err != nil {
			return false, "", err
		}
		return resp.OK, resp.Reason, nil
	})
	return response, err
}

func (m *Manager) StorageProbe(ctx context.Context, extensionID string, request StorageProbeRequest) (StorageProbeResponse, error) {
	var response StorageProbeResponse
	err := m.storageCall(ctx, extensionID, func(ctx context.Context, invoker StorageProviderInvoker) (bool, string, error) {
		resp, err := invoker.StorageProbe(ctx, extensionID, request)
		response = resp
		if err != nil {
			return false, "", err
		}
		return resp.OK, resp.Reason, nil
	})
	return response, err
}

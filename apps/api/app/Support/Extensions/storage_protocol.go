package extensionsruntime

// 附件存储插件 RPC 契约（E6.2）。
// 与 mail 一样走 go-plugin Protocol V2；大对象用分块，默认 chunk 1 MiB（见 Support/Storage）。
// 业务层只见 storage.Adapter；本文件定义跨进程载荷。

// StorageResult 是无载荷写操作的通用结果（PutChunk / Close / Delete）。
type StorageResult struct {
	OK      bool
	Reason  string
	Message string
}

// StoragePutBeginRequest 开启一次 Put 会话。
type StoragePutBeginRequest struct {
	InstanceID  string
	Key         string
	ContentType string
	// Size 为对象总字节；未知时为 0。宿主在流结束前仍以 Final 标记结束。
	Size int64
}

// StorageSessionResponse 返回会话 id（PutBegin / Open）。
type StorageSessionResponse struct {
	OK          bool
	SessionID   string
	Size        int64
	ContentType string
	Reason      string
	Message     string
}

// StoragePutChunkRequest 写入一块数据；Final=true 表示对象写完并提交。
type StoragePutChunkRequest struct {
	SessionID string
	Data      []byte
	Final     bool
}

// StorageOpenRequest 按 key 打开对象读会话。
type StorageOpenRequest struct {
	InstanceID string
	Key        string
}

// StorageGetChunkRequest 从读会话拉取下一块。
type StorageGetChunkRequest struct {
	SessionID string
	// MaxBytes 宿主希望的最大块；插件可返回更短。0 表示由插件默认。
	MaxBytes int
}

// StorageGetChunkResponse 读块结果；EOF=true 表示无更多数据。
type StorageGetChunkResponse struct {
	OK      bool
	Data    []byte
	EOF     bool
	Reason  string
	Message string
}

// StorageCloseRequest 关闭读/写会话（Abort 未完成 Put，或释放 Open）。
type StorageCloseRequest struct {
	SessionID string
}

// StorageObjectRequest 仅含对象 key 的操作（Delete）。
type StorageObjectRequest struct {
	InstanceID string
	Key        string
}

// StorageStatRequest / StorageStatResponse 元数据查询。
type StorageStatRequest struct {
	InstanceID string
	Key        string
}

type StorageStatResponse struct {
	OK           bool
	Exists       bool
	Size         int64
	ContentType  string
	ModifiedUnix int64
	Reason       string
	Message      string
}

// StorageExistsRequest / StorageExistsResponse 存在性查询。
type StorageExistsRequest struct {
	InstanceID string
	Key        string
}

type StorageExistsResponse struct {
	OK      bool
	Exists  bool
	Reason  string
	Message string
}

// StoragePublicURLRequest / StorageSignedURLRequest / StorageURLResponse URL 辅助。
type StoragePublicURLRequest struct {
	InstanceID string
	Key        string
}

type StorageSignedURLRequest struct {
	InstanceID string
	Key        string
	TTLSeconds int64
}

type StorageURLResponse struct {
	OK      bool
	URL     string
	Reason  string
	Message string
}

// StorageProbeRequest / StorageProbeResponse 管理端「测试连接」。
type StorageProbeRequest struct {
	InstanceID string
}

type StorageProbeResponse struct {
	OK      bool
	Reason  string
	Message string
}

// StorageConfigureInstanceRequest installs or replaces one Host-owned provider
// instance in the running plugin. Settings may contain resolved secrets and
// must never be logged or included in telemetry.
type StorageConfigureInstanceRequest struct {
	InstanceID string
	Settings   map[string]string
}

type StorageRemoveInstanceRequest struct {
	InstanceID string
}

// StorageProbeConfigRequest validates draft configuration without changing
// the active in-process instance map.
type StorageProbeConfigRequest struct {
	Settings map[string]string
}

type StorageInstanceProtocol interface {
	StorageConfigureInstance(StorageConfigureInstanceRequest) (StorageResult, error)
	StorageRemoveInstance(StorageRemoveInstanceRequest) (StorageResult, error)
	StorageProbeConfig(StorageProbeConfigRequest) (StorageProbeResponse, error)
}

// storageNotImplemented 统一「本插件未实现存储槽」结果。
func storageNotImplementedResult() StorageResult {
	return StorageResult{
		Reason:  "plugin.storage_not_implemented",
		Message: "this plugin does not implement attachment.storage.provider",
	}
}

func storageNotImplementedSession() StorageSessionResponse {
	return StorageSessionResponse{
		Reason:  "plugin.storage_not_implemented",
		Message: "this plugin does not implement attachment.storage.provider",
	}
}

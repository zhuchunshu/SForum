package extensionsruntime

// ProtocolNoop 提供 PluginProtocol 全部方法的安全默认实现。
// 插件可嵌入后只覆盖需要的 RPC（与 sdk/plugin.Noop 一致）。
type ProtocolNoop struct{}

func (ProtocolNoop) Health() (PluginHealth, error) {
	return PluginHealth{OK: true}, nil
}

func (ProtocolNoop) RouteTarget() (PluginRouteTarget, error) {
	return PluginRouteTarget{}, nil
}

func (ProtocolNoop) InvokeHook(PluginHookRequest) (PluginHookResponse, error) {
	return PluginHookResponse{OK: true}, nil
}

func (ProtocolNoop) SendMail(MailProviderRequest) (MailProviderResponse, error) {
	return MailProviderResponse{
		OK:      false,
		Reason:  "plugin.mail_not_implemented",
		Message: "this plugin does not implement mail.provider",
	}, nil
}

func (ProtocolNoop) StoragePutBegin(StoragePutBeginRequest) (StorageSessionResponse, error) {
	return storageNotImplementedSession(), nil
}

func (ProtocolNoop) StoragePutChunk(StoragePutChunkRequest) (StorageResult, error) {
	return storageNotImplementedResult(), nil
}

func (ProtocolNoop) StorageOpen(StorageOpenRequest) (StorageSessionResponse, error) {
	return storageNotImplementedSession(), nil
}

func (ProtocolNoop) StorageGetChunk(StorageGetChunkRequest) (StorageGetChunkResponse, error) {
	r := storageNotImplementedResult()
	return StorageGetChunkResponse{Reason: r.Reason, Message: r.Message}, nil
}

func (ProtocolNoop) StorageClose(StorageCloseRequest) (StorageResult, error) {
	// Close 在未实现存储时仍视为成功，避免宿主读路径泄漏会话时二次报错。
	return StorageResult{OK: true}, nil
}

func (ProtocolNoop) StorageDelete(StorageObjectRequest) (StorageResult, error) {
	return storageNotImplementedResult(), nil
}

func (ProtocolNoop) StorageStat(StorageStatRequest) (StorageStatResponse, error) {
	r := storageNotImplementedResult()
	return StorageStatResponse{Reason: r.Reason, Message: r.Message}, nil
}

func (ProtocolNoop) StorageExists(StorageExistsRequest) (StorageExistsResponse, error) {
	r := storageNotImplementedResult()
	return StorageExistsResponse{Reason: r.Reason, Message: r.Message}, nil
}

func (ProtocolNoop) StoragePublicURL(StoragePublicURLRequest) (StorageURLResponse, error) {
	r := storageNotImplementedResult()
	return StorageURLResponse{Reason: r.Reason, Message: r.Message}, nil
}

func (ProtocolNoop) StorageSignedURL(StorageSignedURLRequest) (StorageURLResponse, error) {
	r := storageNotImplementedResult()
	return StorageURLResponse{Reason: r.Reason, Message: r.Message}, nil
}

func (ProtocolNoop) StorageProbe(StorageProbeRequest) (StorageProbeResponse, error) {
	r := storageNotImplementedResult()
	return StorageProbeResponse{Reason: r.Reason, Message: r.Message}, nil
}

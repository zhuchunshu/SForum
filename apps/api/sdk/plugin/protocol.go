package plugin

import (
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

// 协议类型别名：插件实现 Protocol，经 Serve 暴露给宿主。

type (
	// Protocol 是 go-plugin 后端插件必须实现的 RPC 面。
	Protocol = extensionsruntime.PluginProtocol

	// Health 是 Health() RPC 的返回值。
	Health = extensionsruntime.PluginHealth

	// RouteTarget 是 RouteTarget() RPC 的返回值。
	// 纯 provider 插件可返回空 BaseURL（或 "none"/"disabled"）。
	RouteTarget = extensionsruntime.PluginRouteTarget

	// HookRequest / HookResponse 是事件/filter 钩子载荷。
	HookRequest  = extensionsruntime.PluginHookRequest
	HookResponse = extensionsruntime.PluginHookResponse

	// MailRequest / MailResponse 是 mail.provider 投递载荷。
	MailRequest  = extensionsruntime.MailProviderRequest
	MailResponse = extensionsruntime.MailProviderResponse
)

// Serve 以 HashiCorp go-plugin 协议运行插件进程（阻塞）。
// 通常在 main 中调用：plugin.Serve(myPlugin{})。
func Serve(impl Protocol) {
	extensionsruntime.ServeProtocolPlugin(impl)
}

// Noop 提供全部 RPC 的默认实现，便于只覆盖需要的方法。
// 嵌入后可只实现 Health / InvokeHook / SendMail 中的子集。
type Noop struct{}

func (Noop) Health() (Health, error) {
	return Health{OK: true}, nil
}

func (Noop) RouteTarget() (RouteTarget, error) {
	// 默认不暴露可代理 HTTP 路由。
	return RouteTarget{}, nil
}

func (Noop) InvokeHook(HookRequest) (HookResponse, error) {
	return HookResponse{OK: true}, nil
}

func (Noop) SendMail(MailRequest) (MailResponse, error) {
	return MailResponse{
		OK:      false,
		Reason:  "plugin.mail_not_implemented",
		Message: "this plugin does not implement mail.provider",
	}, nil
}

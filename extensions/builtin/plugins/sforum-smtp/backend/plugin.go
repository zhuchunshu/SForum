package main

import (
	"os"
	"strconv"

	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

// smtpPlugin 仅实现 mail.provider；存储 RPC 走 ProtocolNoop 默认拒绝。
type smtpPlugin struct {
	extensionsruntime.ProtocolNoop
}

func (smtpPlugin) Health() (extensionsruntime.PluginHealth, error) {
	return extensionsruntime.PluginHealth{OK: true}, nil
}
func (smtpPlugin) RouteTarget() (extensionsruntime.PluginRouteTarget, error) {
	// SMTP 仅通过 RPC 提供 mail.provider，不暴露 HTTP 路由。
	return extensionsruntime.PluginRouteTarget{}, nil
}
func (smtpPlugin) InvokeHook(extensionsruntime.PluginHookRequest) (extensionsruntime.PluginHookResponse, error) {
	return extensionsruntime.PluginHookResponse{OK: true}, nil
}
func (smtpPlugin) ProviderProbe(request extensionsruntime.ProviderProbeRequest) (extensionsruntime.ProviderProbeResponse, error) {
	if request.Slot != "mail.provider" {
		return extensionsruntime.ProviderProbeResponse{Reason: "provider.slot_unsupported", Message: "unsupported provider slot"}, nil
	}
	port, _ := strconv.Atoi(os.Getenv("SFORUM_SETTING_PORT"))
	config := smtpConfig{Host: os.Getenv("SFORUM_SETTING_HOST"), Port: port, Encryption: os.Getenv("SFORUM_SETTING_ENCRYPTION"), Username: os.Getenv("SFORUM_SETTING_USERNAME"), Password: os.Getenv("SFORUM_SETTING_PASSWORD"), FromAddress: os.Getenv("SFORUM_SETTING_FROM_ADDRESS"), FromName: os.Getenv("SFORUM_SETTING_FROM_NAME")}
	if err := probeSMTP(config); err != nil {
		return extensionsruntime.ProviderProbeResponse{Reason: err.reason, Message: err.message}, nil
	}
	return extensionsruntime.ProviderProbeResponse{OK: true, Reason: "smtp.connection_ok", Message: "SMTP connection and authentication succeeded."}, nil
}
func (smtpPlugin) SendMail(request extensionsruntime.MailProviderRequest) (extensionsruntime.MailProviderResponse, error) {
	port, _ := strconv.Atoi(os.Getenv("SFORUM_SETTING_PORT"))
	config := smtpConfig{Host: os.Getenv("SFORUM_SETTING_HOST"), Port: port, Encryption: os.Getenv("SFORUM_SETTING_ENCRYPTION"), Username: os.Getenv("SFORUM_SETTING_USERNAME"), Password: os.Getenv("SFORUM_SETTING_PASSWORD"), FromAddress: os.Getenv("SFORUM_SETTING_FROM_ADDRESS"), FromName: os.Getenv("SFORUM_SETTING_FROM_NAME")}
	if err := sendSMTP(config, request); err != nil {
		return extensionsruntime.MailProviderResponse{Classification: err.classification, Reason: err.reason, Message: err.message}, nil
	}
	return extensionsruntime.MailProviderResponse{OK: true, Reason: "smtp.sent"}, nil
}

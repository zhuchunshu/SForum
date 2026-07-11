package main

import (
	"os"
	"strconv"

	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

type smtpPlugin struct{}

func (smtpPlugin) Health() (extensionsruntime.PluginHealth, error) {
	return extensionsruntime.PluginHealth{OK: true}, nil
}
func (smtpPlugin) RouteTarget() (extensionsruntime.PluginRouteTarget, error) {
	return extensionsruntime.PluginRouteTarget{BaseURL: "disabled"}, nil
}
func (smtpPlugin) InvokeHook(extensionsruntime.PluginHookRequest) (extensionsruntime.PluginHookResponse, error) {
	return extensionsruntime.PluginHookResponse{OK: true}, nil
}
func (smtpPlugin) SendMail(request extensionsruntime.MailProviderRequest) (extensionsruntime.MailProviderResponse, error) {
	port, _ := strconv.Atoi(os.Getenv("SFORUM_SETTING_PORT"))
	config := smtpConfig{Host: os.Getenv("SFORUM_SETTING_HOST"), Port: port, Encryption: os.Getenv("SFORUM_SETTING_ENCRYPTION"), Username: os.Getenv("SFORUM_SETTING_USERNAME"), Password: os.Getenv("SFORUM_SETTING_PASSWORD"), FromAddress: os.Getenv("SFORUM_SETTING_FROM_ADDRESS"), FromName: os.Getenv("SFORUM_SETTING_FROM_NAME")}
	if err := sendSMTP(config, request); err != nil {
		return extensionsruntime.MailProviderResponse{Classification: err.classification, Reason: err.reason, Message: err.message}, nil
	}
	return extensionsruntime.MailProviderResponse{OK: true, Reason: "smtp.sent"}, nil
}

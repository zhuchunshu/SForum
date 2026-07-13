package extensionsruntime

import (
	"context"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

type fixedPluginSettings map[string]string

func (settings fixedPluginSettings) ListSettings(context.Context, string) (map[string]string, error) {
	copy := make(map[string]string, len(settings))
	for key, value := range settings {
		copy[key] = value
	}
	return copy, nil
}

// ProbeSettingsAction starts an isolated short-lived RPC runtime. It receives no Host API
// token and is never registered in Manager routes, hooks, jobs, schedules, or providers.
func (m *Manager) ProbeSettingsAction(ctx context.Context, extension extensions.Extension, providerSlot string, values map[string]string) (extensions.SettingsActionProbeResult, error) {
	starter := NewProtocolStarter(ProtocolStarterConfig{Settings: fixedPluginSettings(values)})
	if _, err := starter.Start(ctx, extension); err != nil {
		return extensions.SettingsActionProbeResult{}, err
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = starter.Stop(stopCtx, extension)
	}()
	if providerSlot == "attachment.storage.provider" {
		response, err := starter.StorageProbe(ctx, extension.ID, StorageProbeRequest{})
		return extensions.SettingsActionProbeResult{OK: response.OK, Reason: response.Reason, Message: response.Message}, err
	}
	response, err := starter.ProviderProbe(ctx, extension.ID, ProviderProbeRequest{Slot: providerSlot})
	return extensions.SettingsActionProbeResult{OK: response.OK, Reason: response.Reason, Message: response.Message, Details: response.Details, Suggestions: response.Suggestions}, err
}

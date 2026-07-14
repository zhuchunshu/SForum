package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strings"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

var apiThemeRuntimeHostname = os.Hostname

func newAPIThemeRuntimeWatcher(
	store *extensions.PostgresStore,
	service *extensions.Service,
	logger *slog.Logger,
) (*extensions.ThemeRuntimeWatcher, error) {
	hostname, err := apiThemeRuntimeHostname()
	if err != nil {
		return nil, fmt.Errorf("resolve theme runtime node hostname: %w", err)
	}
	nodeID, err := normalizeThemeRuntimeNodeID(hostname)
	if err != nil {
		return nil, err
	}
	report := func(err error) {
		if err != nil && logger != nil {
			logger.Warn("theme runtime watcher degraded", "error", err)
		}
	}
	notifications := extensions.NewPostgresThemeRuntimeNotifications(
		store, extensions.DefaultThemeRuntimeReconnectDelay, report,
	)
	return extensions.NewThemeRuntimeWatcher(store, service, notifications, extensions.ThemeRuntimeWatcherConfig{
		Identity: extensions.ThemeRuntimeNodeIdentity{
			NodeID: nodeID,
			BootID: extensions.NewActivationBootID(),
		},
		OnError: report,
	})
}

func normalizeThemeRuntimeNodeID(hostname string) (string, error) {
	nodeID := strings.TrimSpace(hostname)
	if nodeID == "" {
		return "", extensions.ErrThemeRuntimeNodeInvalid
	}
	if len([]byte(nodeID)) > 128 {
		digest := sha256.Sum256([]byte(nodeID))
		nodeID = "host-" + hex.EncodeToString(digest[:])
	}
	return nodeID, nil
}

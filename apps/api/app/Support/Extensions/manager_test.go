package extensionsruntime

import (
	"context"
	"errors"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestManagerTracksStartStopStatusAndRouteTargets(t *testing.T) {
	manager := NewManager(ManagerConfig{})
	extension := runtimeExtension("demo.plugin")

	if err := manager.Start(context.Background(), extension); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	status := manager.Status(context.Background(), extension)
	if status.State != extensions.RuntimeRunning || status.RouteCount != 1 {
		t.Fatalf("unexpected running status: %#v", status)
	}
	target, ok := manager.RouteTarget("demo.plugin")
	if !ok || target.BaseURL == "" {
		t.Fatalf("expected route target, got %#v ok=%v", target, ok)
	}

	if err := manager.Stop(context.Background(), extension); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	status = manager.Status(context.Background(), extension)
	if status.State != extensions.RuntimeStopped {
		t.Fatalf("expected stopped status, got %#v", status)
	}
}

func TestManagerRecordsStartFailure(t *testing.T) {
	manager := NewManager(ManagerConfig{Starter: fakeStarter{err: errors.New("start failed")}})
	extension := runtimeExtension("broken.plugin")
	err := manager.Start(context.Background(), extension)
	if err == nil {
		t.Fatal("expected start failure")
	}
	status := manager.Status(context.Background(), extension)
	if status.State != extensions.RuntimeFailed || status.LastError == "" {
		t.Fatalf("expected failed status, got %#v", status)
	}
}

func runtimeExtension(id string) extensions.Extension {
	return extensions.Extension{
		ID:     id,
		Type:   extensions.TypePlugin,
		Status: extensions.StatusEnabled,
		Manifest: extensions.Manifest{
			ID:            id,
			Name:          "Demo Plugin",
			Version:       "1.0.0",
			Type:          extensions.TypePlugin,
			SForumVersion: "^1.0.0",
			Backend:       extensions.ManifestBackend{Entry: "backend/plugin", RPC: "hashicorp-go-plugin", ProtocolVersion: 1},
			Routes:        []extensions.ManifestRoute{{Path: "/hello", Methods: []string{"GET"}, Access: extensions.RouteAccessPublic}},
		},
	}
}

type fakeStarter struct {
	err error
}

func (s fakeStarter) Start(context.Context, extensions.Extension) (RouteTarget, error) {
	if s.err != nil {
		return RouteTarget{}, s.err
	}
	return RouteTarget{BaseURL: "http://127.0.0.1:43210"}, nil
}

func (s fakeStarter) Stop(context.Context, extensions.Extension) error {
	return nil
}

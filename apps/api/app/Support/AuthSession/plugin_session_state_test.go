package authsession

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

// recordingInvalidator 记录被要求擦除的 sid，并可注入失败。
type recordingInvalidator struct {
	mu      sync.Mutex
	sids    []string
	failure error
}

func (r *recordingInvalidator) InvalidateSessionBySID(_ context.Context, sid string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sids = append(r.sids, sid)
	if r.failure != nil {
		return 0, r.failure
	}
	return 1, nil
}

func (r *recordingInvalidator) recorded() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.sids...)
}

func newAuthSessionTestManager(invalidator PluginSessionStateInvalidator) (*Manager, *recordingInvalidator) {
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{HashSecret: "test-secret"})
	recorder, _ := invalidator.(*recordingInvalidator)
	if invalidator != nil {
		manager = manager.WithPluginSessionStateInvalidator(invalidator)
	}
	return manager, recorder
}

// 登出必须擦除被登出会话的插件状态，且用的是该会话的 sid。
func TestDestroyErasesPluginSessionStateForLoggedOutSID(t *testing.T) {
	recorder := &recordingInvalidator{}
	manager, _ := newAuthSessionTestManager(recorder)

	var issued string
	app := fiber.New()
	app.Post("/logout", func(c fiber.Ctx) error {
		info, err := manager.Start(c, 42)
		if err != nil {
			return err
		}
		issued = info.SID
		return manager.Destroy(c)
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/logout", nil))
	if err != nil {
		t.Fatalf("logout request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if issued == "" {
		t.Fatal("expected an issued sid")
	}
	recorded := recorder.recorded()
	if len(recorded) != 1 || recorded[0] != issued {
		t.Fatalf("expected exactly the logged-out sid %q to be erased, got %#v", issued, recorded)
	}
}

// 登录替换旧会话时，必须擦除**被替换掉的**旧 sid（而不是新 sid），
// 否则匿名会话的插件状态会静默变成登录态的插件状态（S4/S5 禁止的无迁移传递）。
func TestStartErasesReplacedSIDSessionState(t *testing.T) {
	recorder := &recordingInvalidator{}
	manager, _ := newAuthSessionTestManager(recorder)

	var first, second string
	app := fiber.New()
	app.Post("/login-twice", func(c fiber.Ctx) error {
		firstInfo, err := manager.Start(c, 42)
		if err != nil {
			return err
		}
		first = firstInfo.SID
		secondInfo, err := manager.Start(c, 42)
		if err != nil {
			return err
		}
		second = secondInfo.SID
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login-twice", nil))
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if first == "" || second == "" || first == second {
		t.Fatalf("expected two distinct sids, got first=%q second=%q", first, second)
	}
	recorded := recorder.recorded()
	if len(recorded) != 1 || recorded[0] != first {
		t.Fatalf("expected the replaced sid %q to be erased exactly once, got %#v", first, recorded)
	}
}

// 擦除失败不得让登出失败：那会把后台卫生问题升级成用户可见故障。
func TestDestroySucceedsWhenPluginStateErasureFails(t *testing.T) {
	recorder := &recordingInvalidator{failure: errors.New("backend down")}
	manager, _ := newAuthSessionTestManager(recorder)

	app := fiber.New()
	app.Post("/logout", func(c fiber.Ctx) error {
		if _, err := manager.Start(c, 42); err != nil {
			return err
		}
		return manager.Destroy(c)
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/logout", nil))
	if err != nil {
		t.Fatalf("logout request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected logout to succeed despite the erasure failure, got %d", resp.StatusCode)
	}
	if len(recorder.recorded()) != 1 {
		t.Fatalf("expected the erasure to still have been attempted, got %#v", recorder.recorded())
	}
}

// 未注入钩子时（绝大多数部署与其他包的测试）登出必须照常工作。
func TestDestroyWithoutInvalidatorStillSucceeds(t *testing.T) {
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{HashSecret: "test-secret"})
	app := fiber.New()
	app.Post("/logout", func(c fiber.Ctx) error {
		if _, err := manager.Start(c, 42); err != nil {
			return err
		}
		return manager.Destroy(c)
	})
	resp, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/logout", nil))
	if err != nil {
		t.Fatalf("logout request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 without an invalidator, got %d", resp.StatusCode)
	}
}

// 装配层可能注入一个「结构上非 nil 但内部是空接口」的值；dropPluginSessionState
// 必须把 panic 吃掉而不是把登出带崩。
func TestDropPluginSessionStateRecoversFromPanic(t *testing.T) {
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{HashSecret: "test-secret"}).
		WithPluginSessionStateInvalidator(panickingInvalidator{})
	manager.dropPluginSessionState(context.Background(), "sid")
}

type panickingInvalidator struct{}

func (panickingInvalidator) InvalidateSessionBySID(context.Context, string) (int, error) {
	panic("invalidator exploded")
}

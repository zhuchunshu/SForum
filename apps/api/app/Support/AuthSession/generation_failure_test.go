package authsession

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func TestCurrentUserIDRejectsPayloadFromChangedGeneration(t *testing.T) {
	var (
		manager       *Manager
		request       fiber.Ctx
		switchSession bool
	)
	manager = NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{
		HashSecret: "test-secret",
		TokenVersion: func(ctx context.Context, userID int64) (int64, error) {
			if switchSession && userID == 42 {
				switchSession = false
				if err := manager.Destroy(request); err != nil {
					return 0, err
				}
				pending, err := manager.BeginWithContext(request, ctx, 84)
				if err != nil {
					return 0, err
				}
				if err := pending.SaveContext(ctx); err != nil {
					return 0, err
				}
			}
			return 0, nil
		},
	})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		_, err := manager.Start(c, 42)
		return err
	})
	app.Get("/switch", func(c fiber.Ctx) error {
		request = c
		switchSession = true
		userID, ok, err := manager.CurrentUserID(c)
		if err != nil || ok || userID != 0 {
			t.Fatalf("stale lookup user=%d ok=%t err=%v", userID, ok, err)
		}
		userID, ok, err = manager.CurrentUserIDWithoutRenewal(c)
		if err != nil || !ok || userID != 84 {
			t.Fatalf("replacement lookup user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	loginResponse, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer loginResponse.Body.Close()
	cookie := loginResponse.Cookies()[0]
	switchRequest := httptest.NewRequest(fiber.MethodGet, "/switch", nil)
	switchRequest.AddCookie(cookie)
	switchResponse, err := app.Test(switchRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer switchResponse.Body.Close()
	if switchResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("status=%d", switchResponse.StatusCode)
	}
}

func TestBeginResetPanicInvalidatesClaimAndAllowsFreshBegin(t *testing.T) {
	panicValue := &struct{ label string }{label: "reset storage panic"}
	storage := newCommitUnknownSessionStorage()
	storage.deletePanics[1] = panicValue
	manager := NewManager(session.NewStore(session.Config{
		Storage: storage, IdleTimeout: time.Hour,
	}), Config{HashSecret: "test-secret"})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			_, _ = manager.Begin(c, 42)
		}()
		if recovered != panicValue {
			t.Fatalf("recovered=%#v", recovered)
		}
		pending, err := manager.Begin(c, 42)
		if err != nil {
			return err
		}
		if err := pending.SaveContext(c.Context()); err != nil {
			return err
		}
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil || !ok || userID != 42 {
			t.Fatalf("fresh Begin user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK || len(response.Cookies()) != 1 {
		t.Fatalf("status=%d cookies=%#v", response.StatusCode, response.Cookies())
	}
}

func TestPendingCleanupPanicsPreserveOriginalAndRevokeDirectory(t *testing.T) {
	originalPanic := &struct{ label string }{label: "directory mutation panic"}
	cleanupPanic := &struct{ label string }{label: "cleanup storage panic"}
	storage := newCommitUnknownSessionStorage()
	storage.deletePrePanics[2] = cleanupPanic
	storage.deletePrePanics[3] = cleanupPanic
	storage.setPrePanics[2] = cleanupPanic
	directory := &panicCreateSessionStore{panicValue: originalPanic}
	manager := NewManager(session.NewStore(session.Config{
		Storage: storage, IdleTimeout: time.Hour,
	}), Config{HashSecret: "test-secret", SessionStore: directory})
	var issuedID string
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		pending, err := manager.Begin(c, 42)
		if err != nil {
			return err
		}
		issuedID = pending.Info().ID
		pending.SetDeviceInfo(SessionRecordInput{UserID: 999, DeviceName: "panic"})
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			_ = pending.SaveContext(c.Context())
		}()
		if recovered != originalPanic {
			t.Fatalf("recovered=%#v", recovered)
		}
		return c.SendStatus(fiber.StatusServiceUnavailable)
	})
	app.Get("/session", func(c fiber.Ctx) error {
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil || ok || userID != 0 {
			t.Fatalf("directory replay user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	directory.mu.Lock()
	createCalls, revokeCalls, revoked := directory.createCalls, directory.revokeCalls, directory.revoked
	directory.mu.Unlock()
	if response.StatusCode != fiber.StatusServiceUnavailable || createCalls != 1 || revokeCalls != 1 || !revoked {
		t.Fatalf(
			"status=%d directory create=%d revoke=%d revoked=%t",
			response.StatusCode, createCalls, revokeCalls, revoked,
		)
	}
	storage.mu.Lock()
	credential := append([]byte(nil), storage.values[issuedID]...)
	storage.mu.Unlock()
	if len(credential) == 0 {
		t.Fatalf("credential %q did not survive pre-commit cleanup panics", issuedID)
	}

	withoutDirectory := NewManager(session.NewStore(session.Config{
		Storage: storage, IdleTimeout: time.Hour,
	}), Config{HashSecret: "test-secret"})
	controlApp := fiber.New()
	controlApp.Get("/session", func(c fiber.Ctx) error {
		userID, ok, err := withoutDirectory.CurrentUserIDWithoutRenewal(c)
		if err != nil || !ok || userID != 42 {
			t.Fatalf("control replay user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	controlReplay := httptest.NewRequest(fiber.MethodGet, "/session", nil)
	controlReplay.AddCookie(&http.Cookie{Name: manager.store.Extractor.Key, Value: issuedID})
	controlResponse, err := controlApp.Test(controlReplay)
	if err != nil {
		t.Fatal(err)
	}
	controlResponse.Body.Close()
	if controlResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("control replay status=%d", controlResponse.StatusCode)
	}

	replay := httptest.NewRequest(fiber.MethodGet, "/session", nil)
	replay.AddCookie(&http.Cookie{Name: manager.store.Extractor.Key, Value: issuedID})
	replayResponse, err := app.Test(replay)
	if err != nil {
		t.Fatal(err)
	}
	defer replayResponse.Body.Close()
	if replayResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("directory replay status=%d", replayResponse.StatusCode)
	}
}

func TestDestroyCleanupPreCommitPanicsPreserveOriginalAndFailClosed(t *testing.T) {
	originalPanic := &struct{ label string }{label: "destroy storage panic"}
	cleanupPanic := &struct{ label string }{label: "cleanup storage panic"}
	storage := newCommitUnknownSessionStorage()
	directory := newFakeSessionStore()
	manager := NewManager(session.NewStore(session.Config{
		Storage: storage, IdleTimeout: time.Hour,
	}), Config{HashSecret: "test-secret", SessionStore: directory})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		_, err := manager.Start(c, 42)
		return err
	})
	app.Post("/logout", func(c fiber.Ctx) error {
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			_ = manager.Destroy(c)
		}()
		if recovered != originalPanic {
			t.Fatalf("recovered=%#v", recovered)
		}
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil || ok || userID != 0 {
			t.Fatalf("destroy panic request user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusServiceUnavailable)
	})
	app.Get("/session", func(c fiber.Ctx) error {
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil || ok || userID != 0 {
			t.Fatalf("replayed destroyed credential user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})

	loginResponse, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer loginResponse.Body.Close()
	if loginResponse.StatusCode != fiber.StatusOK || len(loginResponse.Cookies()) != 1 {
		t.Fatalf("login status=%d cookies=%#v", loginResponse.StatusCode, loginResponse.Cookies())
	}
	oldCookie := loginResponse.Cookies()[0]
	storage.mu.Lock()
	credentialBefore := append([]byte(nil), storage.values[oldCookie.Value]...)
	firstDestroyCall := storage.deleteCalls + 1
	firstCleanupSetCall := storage.setCalls + 1
	storage.deletePrePanics[firstDestroyCall] = originalPanic
	storage.deletePrePanics[firstDestroyCall+1] = cleanupPanic
	storage.deletePrePanics[firstDestroyCall+2] = cleanupPanic
	storage.setPrePanics[firstCleanupSetCall] = cleanupPanic
	storage.mu.Unlock()
	if len(credentialBefore) == 0 {
		t.Fatal("login credential was not persisted")
	}

	logoutRequest := httptest.NewRequest(fiber.MethodPost, "/logout", nil)
	logoutRequest.AddCookie(oldCookie)
	logoutResponse, err := app.Test(logoutRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer logoutResponse.Body.Close()
	if logoutResponse.StatusCode != fiber.StatusServiceUnavailable || len(logoutResponse.Cookies()) != 1 ||
		logoutResponse.Cookies()[0].Value != "" || logoutResponse.Cookies()[0].MaxAge >= 0 {
		t.Fatalf("logout status=%d cookies=%#v", logoutResponse.StatusCode, logoutResponse.Cookies())
	}

	directory.mu.Lock()
	created := append([]SessionRecordInput(nil), directory.created...)
	revocations := append([]fakeRevoke(nil), directory.revokeCalls...)
	directory.mu.Unlock()
	if len(created) != 1 || len(revocations) != 1 || revocations[0].userID != 42 ||
		revocations[0].sid != created[0].SID || revocations[0].reason != "logout" {
		t.Fatalf("created=%#v revocations=%#v", created, revocations)
	}
	storage.mu.Lock()
	credentialAfter := append([]byte(nil), storage.values[oldCookie.Value]...)
	storage.mu.Unlock()
	if string(credentialAfter) != string(credentialBefore) {
		t.Fatalf("credential changed despite pre-commit cleanup panics")
	}

	replay := httptest.NewRequest(fiber.MethodGet, "/session", nil)
	replay.AddCookie(&http.Cookie{Name: oldCookie.Name, Value: oldCookie.Value})
	replayResponse, err := app.Test(replay)
	if err != nil {
		t.Fatal(err)
	}
	defer replayResponse.Body.Close()
	if replayResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("replay status=%d", replayResponse.StatusCode)
	}
}

func TestPendingDeviceInfoCannotChangeAuthorityUser(t *testing.T) {
	directory := newFakeSessionStore()
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{
		HashSecret: "test-secret", SessionStore: directory,
	})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		pending, err := manager.Begin(c, 42)
		if err != nil {
			return err
		}
		pending.SetDeviceInfo(SessionRecordInput{UserID: 999, DeviceName: "spoofed"})
		return pending.SaveContext(c.Context())
	})
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	directory.mu.Lock()
	created := append([]SessionRecordInput(nil), directory.created...)
	directory.mu.Unlock()
	if response.StatusCode != fiber.StatusOK || len(created) != 1 || created[0].UserID != 42 {
		t.Fatalf("status=%d created=%#v", response.StatusCode, created)
	}
}

func TestStartCreatesMinimalDirectoryRecord(t *testing.T) {
	directory := newFakeSessionStore()
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{
		HashSecret: "test-secret", SessionStore: directory,
	})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		_, err := manager.Start(c, 42)
		return err
	})
	app.Get("/session", func(c fiber.Ctx) error {
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil || !ok || userID != 42 {
			t.Fatalf("session user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	loginResponse, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer loginResponse.Body.Close()
	if len(loginResponse.Cookies()) != 1 {
		t.Fatalf("cookies=%#v", loginResponse.Cookies())
	}
	request := httptest.NewRequest(fiber.MethodGet, "/session", nil)
	request.AddCookie(loginResponse.Cookies()[0])
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
}

func TestBeginSaveCreatesMinimalDirectoryRecord(t *testing.T) {
	directory := newFakeSessionStore()
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{
		HashSecret: "test-secret", SessionStore: directory,
	})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		pending, err := manager.Begin(c, 42)
		if err != nil {
			return err
		}
		return pending.Save()
	})
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	directory.mu.Lock()
	created := append([]SessionRecordInput(nil), directory.created...)
	directory.mu.Unlock()
	if response.StatusCode != fiber.StatusOK || len(created) != 1 || created[0].UserID != 42 || created[0].SID == "" {
		t.Fatalf("status=%d created=%#v", response.StatusCode, created)
	}
}

func TestAbandonedBeginRevokesPreviousDirectoryEntry(t *testing.T) {
	storage := newCommitUnknownSessionStorage()
	directory := newFakeSessionStore()
	manager := NewManager(session.NewStore(session.Config{
		Storage: storage, IdleTimeout: time.Hour,
	}), Config{HashSecret: "test-secret", SessionStore: directory})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		_, err := manager.Start(c, 42)
		return err
	})
	app.Post("/begin", func(c fiber.Ctx) error {
		if _, err := manager.Begin(c, 84); err != nil {
			return err
		}
		return c.SendStatus(fiber.StatusServiceUnavailable)
	})
	app.Get("/session", func(c fiber.Ctx) error {
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil || ok || userID != 0 {
			t.Fatalf("abandoned replacement user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	loginResponse, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer loginResponse.Body.Close()
	oldCookie := loginResponse.Cookies()[0]
	beginRequest := httptest.NewRequest(fiber.MethodPost, "/begin", nil)
	beginRequest.AddCookie(oldCookie)
	beginResponse, err := app.Test(beginRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer beginResponse.Body.Close()
	directory.mu.Lock()
	created := append([]SessionRecordInput(nil), directory.created...)
	revocations := append([]fakeRevoke(nil), directory.revokeCalls...)
	directory.mu.Unlock()
	if beginResponse.StatusCode != fiber.StatusServiceUnavailable || len(created) != 1 || len(revocations) != 1 ||
		revocations[0].userID != 42 || revocations[0].sid != created[0].SID ||
		revocations[0].reason != sessionReplacedReason {
		t.Fatalf("status=%d created=%#v revocations=%#v", beginResponse.StatusCode, created, revocations)
	}
	replay := httptest.NewRequest(fiber.MethodGet, "/session", nil)
	replay.AddCookie(oldCookie)
	replayResponse, err := app.Test(replay)
	if err != nil {
		t.Fatal(err)
	}
	defer replayResponse.Body.Close()
	if replayResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("replay status=%d", replayResponse.StatusCode)
	}
}

func TestSaveUsesBeginAuthorityContext(t *testing.T) {
	type contextKey struct{}
	const marker = "begin-authority"
	directory := &saveContextSessionStore{key: contextKey{}, want: marker}
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{
		HashSecret: "test-secret", SessionStore: directory,
	})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		issueCtx := context.WithValue(c.Context(), contextKey{}, marker)
		pending, err := manager.BeginWithContext(c, issueCtx, 42)
		if err != nil {
			return err
		}
		pending.SetDeviceInfo(SessionRecordInput{UserID: 42})
		return pending.Save()
	})
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK || !directory.valid {
		t.Fatalf("status=%d context valid=%t", response.StatusCode, directory.valid)
	}
}

func TestSessionReplacementRevokesPreviousDirectoryEntry(t *testing.T) {
	for _, replacementUserID := range []int64{42, 84} {
		t.Run(fmt.Sprintf("user_%d", replacementUserID), func(t *testing.T) {
			directory := newFakeSessionStore()
			manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{
				HashSecret: "test-secret", SessionStore: directory,
			})
			app := fiber.New()
			app.Post("/login/first", func(c fiber.Ctx) error {
				_, err := manager.Start(c, 42)
				return err
			})
			app.Post("/login/replacement", func(c fiber.Ctx) error {
				_, err := manager.Start(c, replacementUserID)
				return err
			})
			firstResponse, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login/first", nil))
			if err != nil {
				t.Fatal(err)
			}
			defer firstResponse.Body.Close()
			firstCookie := firstResponse.Cookies()[0]
			replacementRequest := httptest.NewRequest(fiber.MethodPost, "/login/replacement", nil)
			replacementRequest.AddCookie(firstCookie)
			replacementResponse, err := app.Test(replacementRequest)
			if err != nil {
				t.Fatal(err)
			}
			defer replacementResponse.Body.Close()
			directory.mu.Lock()
			created := append([]SessionRecordInput(nil), directory.created...)
			revocations := append([]fakeRevoke(nil), directory.revokeCalls...)
			directory.mu.Unlock()
			if replacementResponse.StatusCode != fiber.StatusOK || len(created) != 2 || len(revocations) != 1 ||
				revocations[0].userID != 42 || revocations[0].sid != created[0].SID ||
				revocations[0].reason != sessionReplacedReason || created[1].UserID != replacementUserID {
				t.Fatalf("status=%d created=%#v revocations=%#v", replacementResponse.StatusCode, created, revocations)
			}
		})
	}
}

func TestBeginResetFailureScrubsPreviousCredentialAndDirectory(t *testing.T) {
	resetErr := errors.New("reset delete unavailable")
	storage := newCommitUnknownSessionStorage()
	storage.deleteErrors[2] = resetErr
	directory := newFakeSessionStore()
	manager := NewManager(session.NewStore(session.Config{
		Storage: storage, IdleTimeout: time.Hour,
	}), Config{HashSecret: "test-secret", SessionStore: directory})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		_, err := manager.Start(c, 42)
		return err
	})
	app.Post("/replace", func(c fiber.Ctx) error {
		if _, err := manager.Begin(c, 84); !errors.Is(err, resetErr) {
			t.Fatalf("replace error=%v", err)
		}
		return c.SendStatus(fiber.StatusServiceUnavailable)
	})
	app.Get("/session", func(c fiber.Ctx) error {
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil || ok || userID != 0 {
			t.Fatalf("replayed reset failure user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	loginResponse, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer loginResponse.Body.Close()
	oldCookie := loginResponse.Cookies()[0]
	replaceRequest := httptest.NewRequest(fiber.MethodPost, "/replace", nil)
	replaceRequest.AddCookie(oldCookie)
	replaceResponse, err := app.Test(replaceRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer replaceResponse.Body.Close()
	if replaceResponse.StatusCode != fiber.StatusServiceUnavailable || len(replaceResponse.Cookies()) != 1 ||
		replaceResponse.Cookies()[0].Value != "" || replaceResponse.Cookies()[0].MaxAge >= 0 {
		t.Fatalf("replace status=%d cookies=%#v", replaceResponse.StatusCode, replaceResponse.Cookies())
	}
	directory.mu.Lock()
	revocations := append([]fakeRevoke(nil), directory.revokeCalls...)
	directory.mu.Unlock()
	if len(revocations) != 1 || revocations[0].userID != 42 || revocations[0].reason != "issue_failed" {
		t.Fatalf("revocations=%#v", revocations)
	}
	storage.mu.Lock()
	_, credentialExists := storage.values[oldCookie.Value]
	storage.mu.Unlock()
	if credentialExists {
		t.Fatalf("old credential %q survived reset failure cleanup", oldCookie.Value)
	}
	replay := httptest.NewRequest(fiber.MethodGet, "/session", nil)
	replay.AddCookie(&http.Cookie{Name: oldCookie.Name, Value: oldCookie.Value})
	replayResponse, err := app.Test(replay)
	if err != nil {
		t.Fatal(err)
	}
	defer replayResponse.Body.Close()
	if replayResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("replay status=%d", replayResponse.StatusCode)
	}
}

type saveContextSessionStore struct {
	key   any
	want  any
	valid bool
}

func (s *saveContextSessionStore) CreateSession(ctx context.Context, _ SessionRecordInput) error {
	s.valid = ctx.Value(s.key) == s.want
	return nil
}

func (*saveContextSessionStore) IsSessionRevoked(context.Context, int64, string) (bool, error) {
	return false, nil
}

func (*saveContextSessionStore) TouchSessionLastSeen(context.Context, int64, string) error {
	return nil
}

func (*saveContextSessionStore) RevokeSession(context.Context, int64, string, string) error {
	return nil
}

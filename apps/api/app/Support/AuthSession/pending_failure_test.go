package authsession

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func TestPendingSaveFailureInvalidatesCommittedCredential(t *testing.T) {
	storage := newCommitUnknownSessionStorage()
	storage.setErrors[1] = errors.New("session set outcome unknown")
	storage.setErrors[2] = errors.New("empty scrub outcome unknown")
	storage.deleteErrors[2] = errors.New("session delete unavailable")
	storage.deleteErrors[3] = errors.New("session delete retry unavailable")
	store := session.NewStore(session.Config{
		Storage: storage, IdleTimeout: time.Hour,
		Extractor:  extractors.FromCookie("custom_session"),
		CookiePath: "/secure", CookieDomain: "example.test",
		CookieSameSite: fiber.CookieSameSiteNoneMode, CookieSecure: true, CookieHTTPOnly: true,
	})
	manager := NewManager(store, Config{HashSecret: "test-secret"})
	var issuedID string
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		pending, err := manager.Begin(c, 42)
		if err != nil {
			return err
		}
		issuedID = pending.Info().ID
		if err := pending.SaveContext(c.Context()); err == nil {
			t.Fatal("commit-unknown storage unexpectedly succeeded")
		}
		if userID, ok, err := manager.CurrentUserIDWithoutRenewal(c); err != nil || ok || userID != 0 {
			t.Fatalf("failed issue remained authenticated in request: user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusServiceUnavailable)
	})
	app.Get("/session", func(c fiber.Ctx) error {
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil {
			return err
		}
		if ok || userID != 0 {
			t.Fatalf("failed credential replay authenticated user=%d ok=%t", userID, ok)
		}
		return c.SendStatus(fiber.StatusOK)
	})

	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "https://example.test/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusServiceUnavailable || issuedID == "" {
		t.Fatalf("status=%d issuedID=%q", response.StatusCode, issuedID)
	}
	cookies := response.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("failure cookies=%#v", cookies)
	}
	expired := cookies[0]
	if expired.Name != "custom_session" || expired.Value != "" || expired.MaxAge >= 0 ||
		expired.Path != "/secure" || expired.Domain != "example.test" ||
		expired.SameSite != http.SameSiteNoneMode || !expired.Secure || !expired.HttpOnly {
		t.Fatalf("expired cookie lost production attributes: %#v", expired)
	}

	replay := httptest.NewRequest(fiber.MethodGet, "https://example.test/session", nil)
	replay.AddCookie(&http.Cookie{Name: "custom_session", Value: issuedID})
	replayResponse, err := app.Test(replay)
	if err != nil {
		t.Fatal(err)
	}
	defer replayResponse.Body.Close()
	if replayResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("replay status=%d", replayResponse.StatusCode)
	}
}

func TestBeginWithContextTokenVersionFailurePreservesExistingSession(t *testing.T) {
	versionErr := error(nil)
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{
		HashSecret: "test-secret",
		TokenVersion: func(context.Context, int64) (int64, error) {
			return 3, versionErr
		},
	})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		_, err := manager.Start(c, 42)
		return err
	})
	app.Post("/replace", func(c fiber.Ctx) error {
		if _, err := manager.BeginWithAuthorityVersion(c, c.Context(), 42, 3); !errors.Is(err, versionErr) {
			t.Fatalf("Begin error=%v want=%v", err, versionErr)
		}
		return c.SendStatus(fiber.StatusServiceUnavailable)
	})
	app.Get("/session", func(c fiber.Ctx) error {
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil || !ok || userID != 42 {
			t.Fatalf("preserved session user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})

	loginResponse, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer loginResponse.Body.Close()
	cookie := loginResponse.Cookies()[0]
	versionErr = errors.New("token version unavailable")
	replace := httptest.NewRequest(fiber.MethodPost, "/replace", nil)
	replace.AddCookie(cookie)
	replaceResponse, err := app.Test(replace)
	if err != nil {
		t.Fatal(err)
	}
	defer replaceResponse.Body.Close()
	if replaceResponse.StatusCode != fiber.StatusServiceUnavailable || len(replaceResponse.Cookies()) != 0 {
		t.Fatalf("replace status=%d cookies=%#v", replaceResponse.StatusCode, replaceResponse.Cookies())
	}
	versionErr = nil
	request := httptest.NewRequest(fiber.MethodGet, "/session", nil)
	request.AddCookie(cookie)
	sessionResponse, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer sessionResponse.Body.Close()
	if sessionResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("session status=%d", sessionResponse.StatusCode)
	}
}

func TestPendingSessionRemainsUnauthenticatedUntilSave(t *testing.T) {
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{HashSecret: "test-secret"})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		pending, err := manager.Begin(c, 42)
		if err != nil {
			return err
		}
		if userID, ok, err := manager.CurrentUserIDWithoutRenewal(c); err != nil || ok || userID != 0 {
			t.Fatalf("pending session authenticated user=%d ok=%t err=%v", userID, ok, err)
		}
		if err := pending.SaveContext(c.Context()); err != nil {
			return err
		}
		if userID, ok, err := manager.CurrentUserIDWithoutRenewal(c); err != nil || !ok || userID != 42 {
			t.Fatalf("saved session user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
}

func TestRenewalSaveFailureInvalidatesOldAndCommitUnknownCredentials(t *testing.T) {
	storage := newCommitUnknownSessionStorage()
	renewErr := errors.New("renewed session set outcome unknown")
	storage.setErrors[2] = renewErr
	storage.setErrors[3] = errors.New("renewed session scrub outcome unknown")
	storage.deleteErrors[3] = errors.New("renewed session delete unavailable")
	storage.deleteErrors[4] = errors.New("renewed session delete retry unavailable")
	store := session.NewStore(session.Config{
		Storage: storage, IdleTimeout: time.Hour,
		Extractor:  extractors.FromCookie("custom_session"),
		CookiePath: "/secure", CookieDomain: "example.test",
		CookieSameSite: fiber.CookieSameSiteNoneMode, CookieSecure: true, CookieHTTPOnly: true,
	})
	manager := NewManager(store, Config{HashSecret: "test-secret", RenewalInterval: time.Hour})
	now := time.Now().UTC()
	manager.now = func() time.Time { return now }
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		_, err := manager.Start(c, 42)
		return err
	})
	app.Get("/renew", func(c fiber.Ctx) error {
		for lookup := 0; lookup < 2; lookup++ {
			if userID, ok, err := manager.CurrentUserID(c); !errors.Is(err, renewErr) || ok || userID != 0 {
				t.Fatalf("renewal lookup %d result user=%d ok=%t err=%v", lookup, userID, ok, err)
			}
		}
		if userID, ok, err := manager.CurrentUserIDWithoutRenewal(c); err != nil || ok || userID != 0 {
			t.Fatalf("failed renewal remained authenticated in request: user=%d ok=%t err=%v", userID, ok, err)
		}
		return renewErr
	})
	app.Get("/session", func(c fiber.Ctx) error {
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil {
			return err
		}
		if ok || userID != 0 {
			t.Fatalf("failed renewal credential authenticated user=%d ok=%t", userID, ok)
		}
		return c.SendStatus(fiber.StatusOK)
	})

	loginResponse, err := app.Test(httptest.NewRequest(fiber.MethodPost, "https://example.test/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer loginResponse.Body.Close()
	if loginResponse.StatusCode != fiber.StatusOK || len(loginResponse.Cookies()) != 1 {
		t.Fatalf("login status=%d cookies=%#v", loginResponse.StatusCode, loginResponse.Cookies())
	}
	oldCookie := loginResponse.Cookies()[0]
	now = now.Add(2 * time.Hour)
	renewRequest := httptest.NewRequest(fiber.MethodGet, "https://example.test/renew", nil)
	renewRequest.AddCookie(oldCookie)
	renewResponse, err := app.Test(renewRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer renewResponse.Body.Close()
	if renewResponse.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("renew status=%d", renewResponse.StatusCode)
	}
	cookies := renewResponse.Cookies()
	if len(cookies) != 1 || cookies[0].Name != "custom_session" || cookies[0].Value != "" || cookies[0].MaxAge >= 0 ||
		cookies[0].Path != "/secure" || cookies[0].Domain != "example.test" ||
		cookies[0].SameSite != http.SameSiteNoneMode || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatalf("renew failure cookies=%#v", cookies)
	}

	writtenKeys := storage.writtenKeys()
	if len(writtenKeys) != 3 || writtenKeys[0] != oldCookie.Value ||
		writtenKeys[1] == oldCookie.Value || writtenKeys[2] != writtenKeys[1] {
		t.Fatalf("written session ids=%#v old=%q", writtenKeys, oldCookie.Value)
	}
	for _, credentialID := range []string{oldCookie.Value, writtenKeys[1]} {
		replay := httptest.NewRequest(fiber.MethodGet, "https://example.test/session", nil)
		replay.AddCookie(&http.Cookie{Name: "custom_session", Value: credentialID})
		replayResponse, replayErr := app.Test(replay)
		if replayErr != nil {
			t.Fatal(replayErr)
		}
		if replayResponse.StatusCode != fiber.StatusOK {
			replayResponse.Body.Close()
			t.Fatalf("credential %q replay status=%d", credentialID, replayResponse.StatusCode)
		}
		replayResponse.Body.Close()
	}
}

func TestRenewalRegenerateFailureInvalidatesCommitUnknownCredential(t *testing.T) {
	storage := newCommitUnknownSessionStorage()
	renewErr := errors.New("old session delete outcome unknown")
	storage.deleteErrors[2] = renewErr
	storage.deleteCommits[2] = true
	directory := newFakeSessionStore()
	store := session.NewStore(session.Config{
		Storage: storage, IdleTimeout: time.Hour,
		Extractor:  extractors.FromCookie("custom_session"),
		CookiePath: "/secure", CookieDomain: "example.test",
		CookieSameSite: fiber.CookieSameSiteNoneMode, CookieSecure: true, CookieHTTPOnly: true,
	})
	manager := NewManager(store, Config{
		HashSecret: "test-secret", RenewalInterval: time.Hour, SessionStore: directory,
	})
	now := time.Now().UTC()
	manager.now = func() time.Time { return now }
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		pending, err := manager.Begin(c, 42)
		if err != nil {
			return err
		}
		pending.SetDeviceInfo(SessionRecordInput{UserID: 42, DeviceName: "test"})
		return pending.SaveContext(c.Context())
	})
	app.Get("/renew", func(c fiber.Ctx) error {
		if userID, ok, err := manager.CurrentUserID(c); !errors.Is(err, renewErr) || ok || userID != 0 {
			t.Fatalf("renewal result user=%d ok=%t err=%v", userID, ok, err)
		}
		return renewErr
	})
	app.Get("/session", func(c fiber.Ctx) error {
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil || ok || userID != 0 {
			t.Fatalf("failed credential replay user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})

	loginResponse, err := app.Test(httptest.NewRequest(fiber.MethodPost, "https://example.test/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer loginResponse.Body.Close()
	oldCookie := loginResponse.Cookies()[0]
	now = now.Add(2 * time.Hour)
	renewRequest := httptest.NewRequest(fiber.MethodGet, "https://example.test/renew", nil)
	renewRequest.AddCookie(oldCookie)
	renewResponse, err := app.Test(renewRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer renewResponse.Body.Close()
	if renewResponse.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("renew status=%d", renewResponse.StatusCode)
	}
	cookies := renewResponse.Cookies()
	if len(cookies) != 1 || cookies[0].Name != "custom_session" || cookies[0].Value != "" || cookies[0].MaxAge >= 0 ||
		cookies[0].Path != "/secure" || cookies[0].Domain != "example.test" ||
		cookies[0].SameSite != http.SameSiteNoneMode || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatalf("renew failure cookies=%#v", cookies)
	}
	if keys := storage.writtenKeys(); len(keys) != 1 || keys[0] != oldCookie.Value {
		t.Fatalf("regenerate failure issued another credential: %#v", keys)
	}
	deleted := storage.deletedKeys()
	if len(deleted) < 3 || deleted[len(deleted)-2] != oldCookie.Value || deleted[len(deleted)-1] != oldCookie.Value {
		t.Fatalf("old credential cleanup deletes=%#v old=%q", deleted, oldCookie.Value)
	}
	directory.mu.Lock()
	revocations := append([]fakeRevoke(nil), directory.revokeCalls...)
	directory.mu.Unlock()
	if len(revocations) != 1 || revocations[0].userID != 42 || revocations[0].reason != "renew_failed" {
		t.Fatalf("renew failure revocations=%#v", revocations)
	}
	replay := httptest.NewRequest(fiber.MethodGet, "https://example.test/session", nil)
	replay.AddCookie(oldCookie)
	replayResponse, err := app.Test(replay)
	if err != nil {
		t.Fatal(err)
	}
	defer replayResponse.Body.Close()
	if replayResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("old credential replay status=%d", replayResponse.StatusCode)
	}
}

func TestPendingSaveContextIsConcurrentOneShot(t *testing.T) {
	storage := newCommitUnknownSessionStorage()
	directory := newFakeSessionStore()
	manager := NewManager(session.NewStore(session.Config{
		Storage: storage, IdleTimeout: time.Hour,
	}), Config{HashSecret: "test-secret", SessionStore: directory})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		pending, err := manager.Begin(c, 42)
		if err != nil {
			return err
		}
		pending.SetDeviceInfo(SessionRecordInput{UserID: 42, DeviceName: "test"})
		const callers = 8
		start := make(chan struct{})
		results := make(chan error, callers)
		for range callers {
			go func() {
				<-start
				results <- pending.SaveContext(c.Context())
			}()
		}
		close(start)
		succeeded := 0
		consumed := 0
		for range callers {
			switch result := <-results; {
			case result == nil:
				succeeded++
			case errors.Is(result, ErrPendingConsumed):
				consumed++
			default:
				t.Fatalf("unexpected pending result: %v", result)
			}
		}
		if succeeded != 1 || consumed != callers-1 {
			t.Fatalf("pending results success=%d consumed=%d", succeeded, consumed)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK || len(storage.writtenKeys()) != 1 {
		t.Fatalf("status=%d writes=%#v", response.StatusCode, storage.writtenKeys())
	}
	directory.mu.Lock()
	created := len(directory.created)
	directory.mu.Unlock()
	if created != 1 {
		t.Fatalf("directory creates=%d", created)
	}
}

func TestPendingSaveContextCancellationStopsBeforeDirectoryAndDetachesCleanup(t *testing.T) {
	type contextKey struct{}
	const marker = "issue-authority"
	probe := &contextSessionStoreProbe{key: contextKey{}, want: marker}
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{
		HashSecret:   "test-secret",
		SessionStore: probe,
		TokenVersion: func(ctx context.Context, _ int64) (int64, error) {
			if got := ctx.Value(contextKey{}); got != marker {
				t.Fatalf("token-version context marker=%v", got)
			}
			return 7, nil
		},
	})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		base := context.WithValue(c.Context(), contextKey{}, marker)
		issueCtx, cancel := context.WithCancel(base)
		defer cancel()
		pending, err := manager.BeginWithAuthorityVersion(c, issueCtx, 42, 7)
		if err != nil {
			return err
		}
		pending.SetDeviceInfo(SessionRecordInput{UserID: 42, DeviceName: "test"})
		cancel()
		if err := pending.SaveContext(issueCtx); !errors.Is(err, context.Canceled) {
			t.Fatalf("pending save error=%v", err)
		}
		return c.SendStatus(fiber.StatusServiceUnavailable)
	})
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusServiceUnavailable || probe.createCalls != 0 || probe.revokeCalls != 1 ||
		!probe.revokeValid || probe.revokeReason != "issue_failed" {
		t.Fatalf("status=%d probe=%#v", response.StatusCode, probe)
	}
}

func TestPendingOwnershipRejectsSecondBeginUntilSave(t *testing.T) {
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
		pending.SetDeviceInfo(SessionRecordInput{UserID: 42, DeviceName: "first"})
		if _, err := manager.Begin(c, 84); !errors.Is(err, ErrSessionAuthorityChanged) {
			t.Fatalf("second Begin error=%v", err)
		}
		if err := pending.SaveContext(c.Context()); err != nil {
			return err
		}
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil || !ok || userID != 42 {
			t.Fatalf("saved owner user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	directory.mu.Lock()
	created := append([]SessionRecordInput(nil), directory.created...)
	directory.mu.Unlock()
	if response.StatusCode != fiber.StatusOK || len(created) != 1 || created[0].UserID != 42 || created[0].DeviceName != "first" {
		t.Fatalf("status=%d created=%#v", response.StatusCode, created)
	}
}

func TestDestroyInvalidatesStalePending(t *testing.T) {
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
		pending.SetDeviceInfo(SessionRecordInput{UserID: 42, DeviceName: "stale"})
		if err := manager.Destroy(c); err != nil {
			return err
		}
		if err := pending.SaveContext(c.Context()); !errors.Is(err, ErrSessionAuthorityChanged) {
			t.Fatalf("stale Save error=%v", err)
		}
		if userID, ok, err := manager.CurrentUserIDWithoutRenewal(c); err != nil || ok || userID != 0 {
			t.Fatalf("destroyed pending user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	directory.mu.Lock()
	created := len(directory.created)
	directory.mu.Unlock()
	if response.StatusCode != fiber.StatusNoContent || created != 0 {
		t.Fatalf("status=%d directory creates=%d", response.StatusCode, created)
	}
}

func TestNewPendingAfterDestroyCannotBeOverwrittenByStalePending(t *testing.T) {
	directory := newFakeSessionStore()
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{
		HashSecret: "test-secret", SessionStore: directory,
	})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		stale, err := manager.Begin(c, 42)
		if err != nil {
			return err
		}
		stale.SetDeviceInfo(SessionRecordInput{UserID: 42, DeviceName: "stale"})
		if err := manager.Destroy(c); err != nil {
			return err
		}
		current, err := manager.Begin(c, 84)
		if err != nil {
			return err
		}
		current.SetDeviceInfo(SessionRecordInput{UserID: 84, DeviceName: "current"})
		if err := stale.SaveContext(c.Context()); !errors.Is(err, ErrSessionAuthorityChanged) {
			t.Fatalf("stale Save error=%v", err)
		}
		if err := current.SaveContext(c.Context()); err != nil {
			return err
		}
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil || !ok || userID != 84 {
			t.Fatalf("current pending user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	directory.mu.Lock()
	created := append([]SessionRecordInput(nil), directory.created...)
	directory.mu.Unlock()
	if response.StatusCode != fiber.StatusOK || len(created) != 1 || created[0].UserID != 84 || created[0].DeviceName != "current" {
		t.Fatalf("status=%d created=%#v", response.StatusCode, created)
	}
}

func TestBeginWithAuthorityVersionRequiresTokenVersionSource(t *testing.T) {
	manager := NewManager(session.NewStore(session.Config{IdleTimeout: time.Hour}), Config{HashSecret: "test-secret"})
	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		_, err := manager.Start(c, 42)
		return err
	})
	app.Post("/exact", func(c fiber.Ctx) error {
		if _, err := manager.BeginWithAuthorityVersion(c, c.Context(), 42, 7); !errors.Is(err, ErrSessionAuthorityChanged) {
			t.Fatalf("exact Begin error=%v", err)
		}
		return c.SendStatus(fiber.StatusServiceUnavailable)
	})
	app.Get("/session", func(c fiber.Ctx) error {
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil || !ok || userID != 42 {
			t.Fatalf("preserved session user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	loginResponse, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer loginResponse.Body.Close()
	cookie := loginResponse.Cookies()[0]
	exactRequest := httptest.NewRequest(fiber.MethodPost, "/exact", nil)
	exactRequest.AddCookie(cookie)
	exactResponse, err := app.Test(exactRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer exactResponse.Body.Close()
	if exactResponse.StatusCode != fiber.StatusServiceUnavailable || len(exactResponse.Cookies()) != 0 {
		t.Fatalf("exact status=%d cookies=%#v", exactResponse.StatusCode, exactResponse.Cookies())
	}
	replay := httptest.NewRequest(fiber.MethodGet, "/session", nil)
	replay.AddCookie(cookie)
	replayResponse, err := app.Test(replay)
	if err != nil {
		t.Fatal(err)
	}
	defer replayResponse.Body.Close()
	if replayResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("replay status=%d", replayResponse.StatusCode)
	}
}

func TestCredentialCleanupUsesIndependentBudgets(t *testing.T) {
	type contextKey struct{}
	const marker = "cleanup-budget"
	storage := &cleanupBudgetStorage{key: contextKey{}, want: marker}
	directory := &blockingCleanupSessionStore{key: contextKey{}, want: marker}
	manager := NewManager(session.NewStore(session.Config{
		Storage: storage, IdleTimeout: time.Hour,
	}), Config{HashSecret: "test-secret", SessionStore: directory})
	manager.cleanupTimeout = 20 * time.Millisecond
	app := fiber.New()
	app.Post("/cleanup", func(c fiber.Ctx) error {
		state, err := manager.requestSessionState(c)
		if err != nil {
			return err
		}
		state.session.Set(sessionUserIDKey, int64(42))
		state.session.Set(sessionSIDKey, "cleanup-sid")
		ctx, cancel := context.WithCancel(context.WithValue(c.Context(), contextKey{}, marker))
		cancel()
		manager.abortSessionCredential(
			ctx, c, state, state.session, state.session.ID(), 42, "cleanup-sid", "issue_failed",
		)
		return c.SendStatus(fiber.StatusServiceUnavailable)
	})
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/cleanup", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("status=%d", response.StatusCode)
	}
	directory.mu.Lock()
	revokeValid := directory.valid
	revokeCalls := directory.calls
	directory.mu.Unlock()
	storage.mu.Lock()
	deleteValid := append([]bool(nil), storage.deleteValid...)
	setValid := append([]bool(nil), storage.setValid...)
	storage.mu.Unlock()
	if revokeCalls != 1 || !revokeValid || len(deleteValid) != 2 || !deleteValid[0] || !deleteValid[1] ||
		len(setValid) != 1 || !setValid[0] {
		t.Fatalf("revoke calls=%d valid=%t delete=%#v set=%#v", revokeCalls, revokeValid, deleteValid, setValid)
	}
}

func TestPendingDirectoryPanicInvalidatesCommittedCredential(t *testing.T) {
	panicValue := &struct{ message string }{message: "directory panic after commit"}
	storage := newCommitUnknownSessionStorage()
	directory := &panicCreateSessionStore{panicValue: panicValue}
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
		pending.SetDeviceInfo(SessionRecordInput{UserID: 42, DeviceName: "panic"})
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			_ = pending.SaveContext(c.Context())
		}()
		if recovered != panicValue {
			t.Fatalf("recovered=%#v", recovered)
		}
		if userID, ok, err := manager.CurrentUserIDWithoutRenewal(c); err != nil || ok || userID != 0 {
			t.Fatalf("panic issue user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusServiceUnavailable)
	})
	app.Get("/session", func(c fiber.Ctx) error {
		userID, ok, err := manager.CurrentUserIDWithoutRenewal(c)
		if err != nil || ok || userID != 0 {
			t.Fatalf("replayed panic credential user=%d ok=%t err=%v", userID, ok, err)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusServiceUnavailable || issuedID == "" {
		t.Fatalf("status=%d issuedID=%q", response.StatusCode, issuedID)
	}
	directory.mu.Lock()
	createCalls, revokeCalls, revoked := directory.createCalls, directory.revokeCalls, directory.revoked
	directory.mu.Unlock()
	if createCalls != 1 || revokeCalls != 1 || !revoked {
		t.Fatalf("directory create=%d revoke=%d revoked=%t", createCalls, revokeCalls, revoked)
	}
	replay := httptest.NewRequest(fiber.MethodGet, "/session", nil)
	replay.AddCookie(&http.Cookie{Name: manager.store.Extractor.Key, Value: issuedID})
	replayResponse, err := app.Test(replay)
	if err != nil {
		t.Fatal(err)
	}
	defer replayResponse.Body.Close()
	if replayResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("replay status=%d", replayResponse.StatusCode)
	}
}

type panicCreateSessionStore struct {
	mu          sync.Mutex
	panicValue  any
	createCalls int
	revokeCalls int
	revoked     bool
}

func (s *panicCreateSessionStore) CreateSession(context.Context, SessionRecordInput) error {
	s.mu.Lock()
	s.createCalls++
	panicValue := s.panicValue
	s.mu.Unlock()
	panic(panicValue)
}

func (s *panicCreateSessionStore) IsSessionRevoked(context.Context, int64, string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.revoked, nil
}

func (*panicCreateSessionStore) TouchSessionLastSeen(context.Context, int64, string) error {
	return nil
}

func (s *panicCreateSessionStore) RevokeSession(context.Context, int64, string, string) error {
	s.mu.Lock()
	s.revokeCalls++
	s.revoked = true
	s.mu.Unlock()
	return nil
}

type blockingCleanupSessionStore struct {
	mu    sync.Mutex
	key   any
	want  any
	calls int
	valid bool
}

func (*blockingCleanupSessionStore) CreateSession(context.Context, SessionRecordInput) error {
	return nil
}

func (*blockingCleanupSessionStore) IsSessionRevoked(context.Context, int64, string) (bool, error) {
	return false, nil
}

func (*blockingCleanupSessionStore) TouchSessionLastSeen(context.Context, int64, string) error {
	return nil
}

func (s *blockingCleanupSessionStore) RevokeSession(ctx context.Context, _ int64, _ string, _ string) error {
	<-ctx.Done()
	s.mu.Lock()
	s.calls++
	s.valid = ctx.Value(s.key) == s.want && errors.Is(ctx.Err(), context.DeadlineExceeded)
	s.mu.Unlock()
	return ctx.Err()
}

type cleanupBudgetStorage struct {
	mu          sync.Mutex
	key         any
	want        any
	deleteValid []bool
	setValid    []bool
}

func (*cleanupBudgetStorage) GetWithContext(context.Context, string) ([]byte, error) { return nil, nil }
func (s *cleanupBudgetStorage) Get(key string) ([]byte, error) {
	return s.GetWithContext(context.Background(), key)
}

func (s *cleanupBudgetStorage) SetWithContext(ctx context.Context, _ string, _ []byte, _ time.Duration) error {
	s.mu.Lock()
	s.setValid = append(s.setValid, ctx.Err() == nil && ctx.Value(s.key) == s.want)
	s.mu.Unlock()
	return nil
}

func (s *cleanupBudgetStorage) Set(key string, value []byte, expiration time.Duration) error {
	return s.SetWithContext(context.Background(), key, value, expiration)
}

func (s *cleanupBudgetStorage) DeleteWithContext(ctx context.Context, _ string) error {
	s.mu.Lock()
	s.deleteValid = append(s.deleteValid, ctx.Err() == nil && ctx.Value(s.key) == s.want)
	call := len(s.deleteValid)
	s.mu.Unlock()
	if call <= 2 {
		return errors.New("force cleanup retry")
	}
	return nil
}

func (s *cleanupBudgetStorage) Delete(key string) error {
	return s.DeleteWithContext(context.Background(), key)
}
func (*cleanupBudgetStorage) ResetWithContext(context.Context) error { return nil }
func (s *cleanupBudgetStorage) Reset() error {
	return s.ResetWithContext(context.Background())
}
func (*cleanupBudgetStorage) Close() error { return nil }

type commitUnknownSessionStorage struct {
	mu              sync.Mutex
	values          map[string][]byte
	setKeys         []string
	deleteKeys      []string
	setCalls        int
	deleteCalls     int
	setErrors       map[int]error
	deleteErrors    map[int]error
	deleteCommits   map[int]bool
	setPrePanics    map[int]any
	deletePrePanics map[int]any
	setPanics       map[int]any
	deletePanics    map[int]any
}

func newCommitUnknownSessionStorage() *commitUnknownSessionStorage {
	return &commitUnknownSessionStorage{
		values: map[string][]byte{}, setErrors: map[int]error{},
		deleteErrors: map[int]error{}, deleteCommits: map[int]bool{},
		setPrePanics: map[int]any{}, deletePrePanics: map[int]any{},
		setPanics: map[int]any{}, deletePanics: map[int]any{},
	}
}

func (s *commitUnknownSessionStorage) GetWithContext(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return bytes.Clone(s.values[key]), nil
}

func (s *commitUnknownSessionStorage) Get(key string) ([]byte, error) {
	return s.GetWithContext(context.Background(), key)
}

func (s *commitUnknownSessionStorage) SetWithContext(_ context.Context, key string, value []byte, _ time.Duration) error {
	s.mu.Lock()
	s.setCalls++
	s.setKeys = append(s.setKeys, key)
	prePanicValue := s.setPrePanics[s.setCalls]
	if prePanicValue != nil {
		s.mu.Unlock()
		panic(prePanicValue)
	}
	s.values[key] = bytes.Clone(value)
	err := s.setErrors[s.setCalls]
	panicValue := s.setPanics[s.setCalls]
	s.mu.Unlock()
	if panicValue != nil {
		panic(panicValue)
	}
	return err
}

func (s *commitUnknownSessionStorage) Set(key string, value []byte, expiration time.Duration) error {
	return s.SetWithContext(context.Background(), key, value, expiration)
}

func (s *commitUnknownSessionStorage) DeleteWithContext(_ context.Context, key string) error {
	s.mu.Lock()
	s.deleteCalls++
	s.deleteKeys = append(s.deleteKeys, key)
	prePanicValue := s.deletePrePanics[s.deleteCalls]
	if prePanicValue != nil {
		s.mu.Unlock()
		panic(prePanicValue)
	}
	err := s.deleteErrors[s.deleteCalls]
	panicValue := s.deletePanics[s.deleteCalls]
	if err != nil && !s.deleteCommits[s.deleteCalls] {
		s.mu.Unlock()
		if panicValue != nil {
			panic(panicValue)
		}
		return err
	}
	delete(s.values, key)
	s.mu.Unlock()
	if panicValue != nil {
		panic(panicValue)
	}
	return err
}

func (s *commitUnknownSessionStorage) Delete(key string) error {
	return s.DeleteWithContext(context.Background(), key)
}

func (s *commitUnknownSessionStorage) ResetWithContext(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.values)
	return nil
}

func (s *commitUnknownSessionStorage) Reset() error {
	return s.ResetWithContext(context.Background())
}

func (*commitUnknownSessionStorage) Close() error { return nil }

func (s *commitUnknownSessionStorage) writtenKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.setKeys...)
}

func (s *commitUnknownSessionStorage) deletedKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.deleteKeys...)
}

type contextSessionStoreProbe struct {
	key          any
	want         any
	createCalls  int
	revokeCalls  int
	revokeReason string
	revokeValid  bool
}

func (p *contextSessionStoreProbe) CreateSession(ctx context.Context, _ SessionRecordInput) error {
	p.createCalls++
	if got := ctx.Value(p.key); got != p.want || !errors.Is(ctx.Err(), context.Canceled) {
		return errors.New("issue context was not propagated")
	}
	return context.Canceled
}

func (*contextSessionStoreProbe) IsSessionRevoked(context.Context, int64, string) (bool, error) {
	return false, nil
}

func (*contextSessionStoreProbe) TouchSessionLastSeen(context.Context, int64, string) error {
	return nil
}

func (p *contextSessionStoreProbe) RevokeSession(ctx context.Context, _ int64, _ string, reason string) error {
	p.revokeCalls++
	p.revokeReason = reason
	p.revokeValid = ctx.Value(p.key) == p.want && ctx.Err() == nil
	if !p.revokeValid {
		return errors.New("cleanup context lost value or cancellation was not detached")
	}
	return nil
}

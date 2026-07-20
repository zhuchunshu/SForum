package authsession

import (
	"bytes"
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func TestCurrentUserIDMemoizesRenewalStorageFailureBeforeClearedSession(t *testing.T) {
	t.Parallel()

	renewErr := errors.New("renewal storage failed")
	storage := newRenewalMemoStorage()
	var versionCalls atomic.Int32
	var gateCalls atomic.Int32
	var effectCalls atomic.Int32
	manager := NewManager(session.NewStore(session.Config{
		Storage: storage, IdleTimeout: time.Hour,
	}), Config{
		HashSecret:      "test-secret",
		RenewalInterval: time.Hour,
		TokenVersion: func(context.Context, int64) (int64, error) {
			versionCalls.Add(1)
			return 7, nil
		},
		RenewalEffectGate: func(ctx context.Context, _ int64, _ int64, effect RenewalEffect) error {
			gateCalls.Add(1)
			effectCalls.Add(1)
			return effect(ctx)
		},
	})
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }

	app := fiber.New()
	app.Post("/login", func(c fiber.Ctx) error {
		_, err := manager.Start(c, 42)
		return err
	})
	app.Get("/renew", func(c fiber.Ctx) error {
		var first renewalMemoObservation
		for lookup := 0; lookup < 3; lookup++ {
			userID, ok, err := manager.CurrentUserID(c)
			if userID != 0 || ok || !errors.Is(err, renewErr) {
				t.Fatalf("lookup %d user=%d ok=%t err=%v", lookup, userID, ok, err)
			}

			got := renewalMemoObservation{
				versionCalls: versionCalls.Load(),
				gateCalls:    gateCalls.Load(),
				effectCalls:  effectCalls.Load(),
				storage:      storage.snapshot(),
			}
			if lookup == 0 {
				if got.versionCalls != 2 || got.gateCalls != 1 || got.effectCalls != 1 || got.storage.setCalls != 1 {
					t.Fatalf("first lookup effects=%#v", got)
				}
				first = got
				continue
			}
			if got != first {
				t.Fatalf("lookup %d repeated provider/effect/storage: first=%#v got=%#v", lookup, first, got)
			}
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	loginResponse, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer loginResponse.Body.Close()
	if loginResponse.StatusCode != fiber.StatusOK || len(loginResponse.Cookies()) != 1 {
		t.Fatalf("login status=%d cookies=%#v", loginResponse.StatusCode, loginResponse.Cookies())
	}

	versionCalls.Store(0)
	storage.resetCounts()
	storage.failNextSet(renewErr)
	now = now.Add(2 * time.Hour)
	request := httptest.NewRequest(fiber.MethodGet, "/renew", nil)
	request.AddCookie(loginResponse.Cookies()[0])
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusNoContent {
		t.Fatalf("renew status=%d", response.StatusCode)
	}
}

type renewalMemoObservation struct {
	versionCalls int32
	gateCalls    int32
	effectCalls  int32
	storage      renewalMemoStorageCounts
}

type renewalMemoStorageCounts struct {
	getCalls    int
	setCalls    int
	deleteCalls int
	resetCalls  int
}

type renewalMemoStorage struct {
	mu      sync.Mutex
	values  map[string][]byte
	counts  renewalMemoStorageCounts
	nextSet error
}

func newRenewalMemoStorage() *renewalMemoStorage {
	return &renewalMemoStorage{values: make(map[string][]byte)}
}

func (s *renewalMemoStorage) GetWithContext(_ context.Context, stringKey string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts.getCalls++
	return bytes.Clone(s.values[stringKey]), nil
}

func (s *renewalMemoStorage) Get(stringKey string) ([]byte, error) {
	return s.GetWithContext(context.Background(), stringKey)
}

func (s *renewalMemoStorage) SetWithContext(
	_ context.Context,
	stringKey string,
	value []byte,
	_ time.Duration,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts.setCalls++
	if s.nextSet != nil {
		err := s.nextSet
		s.nextSet = nil
		return err
	}
	s.values[stringKey] = bytes.Clone(value)
	return nil
}

func (s *renewalMemoStorage) Set(stringKey string, value []byte, expiration time.Duration) error {
	return s.SetWithContext(context.Background(), stringKey, value, expiration)
}

func (s *renewalMemoStorage) DeleteWithContext(_ context.Context, stringKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts.deleteCalls++
	delete(s.values, stringKey)
	return nil
}

func (s *renewalMemoStorage) Delete(stringKey string) error {
	return s.DeleteWithContext(context.Background(), stringKey)
}

func (s *renewalMemoStorage) ResetWithContext(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts.resetCalls++
	clear(s.values)
	return nil
}

func (s *renewalMemoStorage) Reset() error {
	return s.ResetWithContext(context.Background())
}

func (*renewalMemoStorage) Close() error { return nil }

func (s *renewalMemoStorage) failNextSet(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextSet = err
}

func (s *renewalMemoStorage) resetCounts() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts = renewalMemoStorageCounts{}
}

func (s *renewalMemoStorage) snapshot() renewalMemoStorageCounts {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.counts
}

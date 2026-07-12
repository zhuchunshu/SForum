package webhookjobs

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/riverqueue/river"
	webhooks "github.com/zhuchunshu/sforum/apps/api/app/Models/Webhooks"
)

func TestDeliverWorkerSignsAndSucceeds(t *testing.T) {
	var gotSig, gotEvent string
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-SForum-Signature")
		gotEvent = r.Header.Get("X-SForum-Event")
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`ok`))
	}))
	defer server.Close()

	store := &fakeStore{
		delivery: webhooks.Delivery{
			ID: 9, EndpointID: 1, EventName: "topic.created", Status: webhooks.StatusQueued,
			Payload: []byte(`{"name":"topic.created"}`),
		},
		endpoint: webhooks.EndpointRecord{
			Endpoint: webhooks.Endpoint{ID: 1, TargetURL: server.URL, Enabled: true},
			Secret:   "whsec_test",
		},
	}
	fixed := time.Unix(1_700_000_000, 0).UTC()
	worker := &DeliverWorker{Store: store, Client: server.Client(), Now: func() time.Time { return fixed }}
	if err := worker.Work(context.Background(), &river.Job[DeliverArgs]{Args: DeliverArgs{DeliveryID: 9}}); err != nil {
		t.Fatal(err)
	}
	if store.update.Status != webhooks.StatusSent {
		t.Fatalf("status=%s", store.update.Status)
	}
	if gotEvent != "topic.created" {
		t.Fatalf("event header=%q", gotEvent)
	}
	want := signPayload("whsec_test", fixed.Unix(), body)
	if gotSig != want {
		t.Fatalf("sig=%q want %q", gotSig, want)
	}
}

func TestDeliverWorkerRetriesOn5xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()
	store := &fakeStore{
		delivery: webhooks.Delivery{ID: 2, EndpointID: 1, Status: webhooks.StatusQueued, Payload: []byte(`{}`)},
		endpoint: webhooks.EndpointRecord{Endpoint: webhooks.Endpoint{ID: 1, TargetURL: server.URL, Enabled: true}},
	}
	worker := &DeliverWorker{Store: store, Client: server.Client()}
	err := worker.Work(context.Background(), &river.Job[DeliverArgs]{Args: DeliverArgs{DeliveryID: 2}})
	if err == nil {
		t.Fatal("expected retryable error")
	}
	if store.update.Status != webhooks.StatusSending {
		t.Fatalf("status=%s", store.update.Status)
	}
}

func TestSignPayloadFormat(t *testing.T) {
	ts := int64(100)
	body := []byte(`{"a":1}`)
	sig := signPayload("secret", ts, body)
	if !strings.HasPrefix(sig, "t=100,v1=") {
		t.Fatalf("sig=%s", sig)
	}
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = fmt.Fprintf(mac, "%d.", ts)
	_, _ = mac.Write(body)
	want := "t=" + strconv.FormatInt(ts, 10) + ",v1=" + hex.EncodeToString(mac.Sum(nil))
	if sig != want {
		t.Fatalf("got %s want %s", sig, want)
	}
}

type fakeStore struct {
	delivery webhooks.Delivery
	endpoint webhooks.EndpointRecord
	update   webhooks.DeliveryUpdate
}

func (s *fakeStore) GetDelivery(context.Context, int64) (webhooks.Delivery, error) {
	return s.delivery, nil
}
func (s *fakeStore) UpdateDelivery(_ context.Context, update webhooks.DeliveryUpdate) error {
	s.update = update
	s.delivery.Status = update.Status
	s.delivery.AttemptCount = update.AttemptCount
	return nil
}
func (s *fakeStore) GetEndpoint(context.Context, int64) (webhooks.EndpointRecord, error) {
	return s.endpoint, nil
}

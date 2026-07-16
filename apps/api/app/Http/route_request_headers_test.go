package http

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestCopyRouteRequestHeadersStripsSensitiveAndConnectionNamedHeaders(t *testing.T) {
	source := http.Header{
		"Accept-Language":     {"zh-CN"},
		"Authorization":       {"Bearer browser-secret"},
		"Connection":          {"keep-alive, X-Hop-Secret"},
		"Cookie":              {"session=browser-secret"},
		"Host":                {"forum.example.com"},
		"Proxy-Authorization": {"Basic proxy-secret"},
		"Proxy-Connection":    {"keep-alive"},
		"X-Csrf-Token":        {"csrf-secret"},
		"X-Hop-Secret":        {"hop-secret"},
		"X-SForum-Forged":     {"forged"},
		"X-Trace-ID":          {"trace-1"},
	}
	target := make(http.Header)

	copyRouteRequestHeaders(target, source)

	if target.Get("Accept-Language") != "zh-CN" || target.Get("X-Trace-ID") != "trace-1" {
		t.Fatalf("ordinary headers were removed: %#v", target)
	}
	for _, name := range []string{
		"Authorization", "Connection", "Cookie", "Host", "Proxy-Authorization", "Proxy-Connection",
		"X-Csrf-Token", "X-Hop-Secret", "X-SForum-Forged",
	} {
		if values := target.Values(name); len(values) != 0 {
			t.Fatalf("blocked header %s survived: %#v", name, values)
		}
	}
}

func TestBufferedRouteStepInvokerDoesNotFollowRedirects(t *testing.T) {
	var destinationCalls atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		destinationCalls.Add(1)
	}))
	t.Cleanup(destination.Close)
	source := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Location", destination.URL)
		writer.WriteHeader(http.StatusTemporaryRedirect)
	}))
	t.Cleanup(source.Close)

	response, err := NewBufferedRouteStepInvoker(nil).Client.Get(source.URL)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusTemporaryRedirect || destinationCalls.Load() != 0 {
		t.Fatalf("status=%d destinationCalls=%d", response.StatusCode, destinationCalls.Load())
	}
}

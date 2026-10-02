package ai_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
)

func testInvoker(server *httptest.Server) *ai.HTTPInvoker {
	return ai.NewHTTPInvoker(ai.HTTPInvokerOptions{
		Client:            server.Client(),
		SkipURLValidation: true,
		Timeout:           5 * time.Second,
	})
}

func localProfile(server *httptest.Server) ai.Profile {
	profile := deepseekProfile()
	profile.BaseURL = server.URL
	profile.Defaults.TimeoutMS = 2_000
	return profile
}

func TestHTTPInvokerSendsProtocolRequestAndParsesResponse(t *testing.T) {
	var seenAuth string
	var seenPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		seenPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":4,"completion_tokens":2}}`))
	}))
	defer server.Close()

	wire, err := ai.BuildWireRequest(localProfile(server), "sk-live", validRequest())
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	response, err := testInvoker(server).Invoke(context.Background(), localProfile(server), wire)
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}
	if response.Text != "hello" || response.Usage.InputTokens != 4 {
		t.Fatalf("response = %+v", response)
	}
	if seenAuth != "Bearer sk-live" {
		t.Fatalf("authorization = %q", seenAuth)
	}
	if seenPath != "/chat/completions" {
		t.Fatalf("path = %q", seenPath)
	}
}

func TestHTTPInvokerSurfacesProviderStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer server.Close()

	wire, err := ai.BuildWireRequest(localProfile(server), "sk-live", validRequest())
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	_, err = testInvoker(server).Invoke(context.Background(), localProfile(server), wire)
	if !errors.Is(err, ai.ErrWire) {
		t.Fatalf("expected ErrWire, got %v", err)
	}
	if !strings.Contains(err.Error(), "429") {
		t.Fatalf("error should carry the status: %v", err)
	}
}

func TestHTTPInvokerHonoursProfileTimeout(t *testing.T) {
	// 供应商比 profile 允许的时间更慢：调用必须被 profile 的超时切断，
	// 而不是等客户端级预算。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2500 * time.Millisecond)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"late"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	profile := localProfile(server)
	profile.Defaults.TimeoutMS = ai.MinTimeoutMS
	wire, err := ai.BuildWireRequest(profile, "sk-live", validRequest())
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	// 客户端级预算设得比 profile 更紧，验证 profile 超时确实生效。
	invoker := ai.NewHTTPInvoker(ai.HTTPInvokerOptions{
		Client:            server.Client(),
		SkipURLValidation: true,
		Timeout:           5 * time.Second,
	})
	started := time.Now()
	if _, err := invoker.Invoke(context.Background(), profile, wire); err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed := time.Since(started); elapsed > 1800*time.Millisecond {
		t.Fatalf("call should have been cut by the profile timeout, took %s", elapsed)
	}
}

func TestHTTPInvokerRejectsUnsafeTargetWhenValidationIsOn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	profile := localProfile(server)
	wire, err := ai.BuildWireRequest(profile, "sk-live", validRequest())
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	// 127.0.0.1 是回环地址，出站守卫必须拒绝它。
	invoker := ai.NewHTTPInvoker(ai.HTTPInvokerOptions{Client: server.Client()})
	if _, err := invoker.Invoke(context.Background(), profile, wire); err == nil {
		t.Fatal("loopback target must be rejected by the SSRF guard")
	}
}

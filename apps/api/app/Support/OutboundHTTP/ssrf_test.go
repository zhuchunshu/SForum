package outboundhttp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestIsForbiddenIP(t *testing.T) {
	forbidden := []string{
		"127.0.0.1", "::1", "10.0.0.1", "172.16.5.1", "192.168.1.1",
		"169.254.169.254", "fe80::1", "0.0.0.0", "224.0.0.1",
		"100.64.0.1", "192.0.2.1", "198.51.100.1", "203.0.113.1",
		"2001:db8::1", "fc00::1",
	}
	for _, raw := range forbidden {
		ip := net.ParseIP(raw)
		if ip == nil {
			t.Fatalf("parse %s", raw)
		}
		if !IsForbiddenIP(ip) {
			t.Fatalf("expected forbidden: %s", raw)
		}
	}
	// 公网示例（非真实连通性）
	if IsForbiddenIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("8.8.8.8 should be allowed")
	}
	if IsForbiddenIP(net.ParseIP("1.1.1.1")) {
		t.Fatal("1.1.1.1 should be allowed")
	}
}

func TestValidatePublicURLRejects(t *testing.T) {
	opts := Options{AllowHTTP: true}
	cases := []string{
		"http://127.0.0.1/hook",
		"http://[::1]/hook",
		"http://10.0.0.5/x",
		"http://192.168.0.2/x",
		"http://169.254.169.254/latest/meta-data",
		"http://user:pass@example.com/x",
		"file:///etc/passwd",
		"ftp://example.com/x",
		"",
	}
	for _, raw := range cases {
		if err := ValidatePublicURL(raw, opts); err == nil {
			t.Fatalf("expected reject for %q", raw)
		} else if !errors.Is(err, ErrUnsafeURL) {
			t.Fatalf("%q: want ErrUnsafeURL, got %v", raw, err)
		}
	}
}

func TestValidatePublicURLRejectsHTTPWhenDisallowed(t *testing.T) {
	err := ValidatePublicURL("http://example.com/hook", Options{AllowHTTP: false})
	if !errors.Is(err, ErrUnsafeURL) {
		t.Fatalf("got %v", err)
	}
}

func TestValidatePublicURLAcceptsHTTPSPublic(t *testing.T) {
	// 使用固定公网 IP 字面量，避免依赖外部 DNS。
	if err := ValidatePublicURL("https://8.8.8.8/hook", Options{}); err != nil {
		t.Fatalf("expected allow: %v", err)
	}
}

func TestValidatePublicURLRejectsHostnameResolvingPrivate(t *testing.T) {
	r := staticResolver{ips: []net.IPAddr{{IP: net.ParseIP("10.1.2.3")}}}
	err := ValidatePublicURL("https://evil.example/hook", Options{Resolver: r})
	if !errors.Is(err, ErrUnsafeURL) {
		t.Fatalf("got %v", err)
	}
}

func TestDialContextBlocksDNSRebinding(t *testing.T) {
	// Dial 时解析结果始终为私网：模拟配置后 DNS rebinding。
	r := staticResolver{ips: []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}}
	client := NewSafeClient(Options{AllowHTTP: true, Resolver: r, Timeout: 2 * time.Second})

	req, err := http.NewRequest(http.MethodGet, "http://rebinding.test/", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("expected dial failure on rebinding")
	}
	if !errors.Is(err, ErrUnsafeURL) && !strings.Contains(err.Error(), "unsafe url") &&
		!strings.Contains(err.Error(), "not allowed") && !strings.Contains(err.Error(), "disallowed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolvePublicIPsRejectsAnyForbiddenAmongMany(t *testing.T) {
	// 多 A 记录中只要有一个私网即拒绝（防部分解析逃逸）。
	r := staticResolver{ips: []net.IPAddr{
		{IP: net.ParseIP("8.8.8.8")},
		{IP: net.ParseIP("10.0.0.1")},
	}}
	_, err := resolvePublicIPs(context.Background(), r, "mixed.example")
	if !errors.Is(err, ErrUnsafeURL) {
		t.Fatalf("got %v", err)
	}
}

func TestCheckRedirectStripsSignatureOnCrossOrigin(t *testing.T) {
	var hop2Sig string
	// hop1 公网字面量不可用本地 server；改测 CheckRedirect 逻辑本身。
	opts := Options{AllowHTTP: true, Resolver: staticResolver{ips: []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}}}
	client := NewSafeClient(opts)

	// 直接调用 CheckRedirect
	from, _ := url.Parse("https://a.example/hook")
	to, _ := url.Parse("https://b.example/hook")
	req, _ := http.NewRequest(http.MethodPost, to.String(), nil)
	req.Header.Set("X-SForum-Signature", "t=1,v1=abc")
	req.Header.Set("X-SForum-Event", "topic.created")
	via := []*http.Request{{URL: from, Header: http.Header{"X-SForum-Signature": []string{"t=1,v1=abc"}}}}
	if err := client.CheckRedirect(req, via); err != nil {
		t.Fatalf("redirect check: %v", err)
	}
	hop2Sig = req.Header.Get("X-SForum-Signature")
	if hop2Sig != "" {
		t.Fatalf("signature must be stripped on cross-origin, got %q", hop2Sig)
	}
	if req.Header.Get("X-SForum-Event") != "" {
		t.Fatal("event header must be stripped")
	}
}

func TestCheckRedirectRejectsPrivateDestination(t *testing.T) {
	opts := Options{AllowHTTP: true, Resolver: staticResolver{ips: []net.IPAddr{{IP: net.ParseIP("10.0.0.9")}}}}
	client := NewSafeClient(opts)
	from, _ := url.Parse("https://a.example/hook")
	to, _ := url.Parse("https://internal.example/hook")
	req, _ := http.NewRequest(http.MethodGet, to.String(), nil)
	err := client.CheckRedirect(req, []*http.Request{{URL: from}})
	if !errors.Is(err, ErrUnsafeURL) {
		t.Fatalf("got %v", err)
	}
}

func TestSafeClientDeliversToPublicLoopbackOverrideNotUsed(t *testing.T) {
	// 确认默认客户端拒绝 loopback httptest（生产路径）。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := NewSafeClient(Options{AllowHTTP: true})
	req, _ := http.NewRequest(http.MethodPost, server.URL, nil)
	_, err := client.Do(req)
	if err == nil {
		t.Fatal("expected loopback target rejected by safe client")
	}
}

type staticResolver struct {
	ips []net.IPAddr
}

func (s staticResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return s.ips, nil
}



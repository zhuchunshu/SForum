package clientip

import "testing"

func TestResolveDirectRemoteNoTrust(t *testing.T) {
	r := NewResolver(Config{})
	// 对端是公网、不信任任何代理：忽略伪造 XFF。
	got := r.Resolve("203.0.113.50", func(h string) string {
		if h == headerXForwardedFor {
			return "1.2.3.4"
		}
		return ""
	})
	if got != "203.0.113.50" {
		t.Fatalf("expected remote IP, got %q", got)
	}
}

func TestResolveXFFStripTrustedProxies(t *testing.T) {
	r := NewResolver(Config{TrustPrivate: true, TrustLoopback: true})
	// Docker/Nuxt 对端是私网，XFF = client, edge, nuxt
	got := r.Resolve("172.18.0.5", func(h string) string {
		if h == headerXForwardedFor {
			return "203.0.113.9, 10.0.0.2, 172.18.0.5"
		}
		return ""
	})
	if got != "203.0.113.9" {
		t.Fatalf("expected client from XFF, got %q", got)
	}
}

func TestResolveCFConnectingIPPreferred(t *testing.T) {
	r := NewResolver(Config{TrustPrivate: true})
	got := r.Resolve("10.0.0.1", func(h string) string {
		switch h {
		case "CF-Connecting-IP":
			return "198.51.100.7"
		case headerXForwardedFor:
			return "203.0.113.1, 10.0.0.1"
		}
		return ""
	})
	if got != "198.51.100.7" {
		t.Fatalf("expected CF-Connecting-IP, got %q", got)
	}
}

func TestResolveXRealIP(t *testing.T) {
	r := NewResolver(Config{TrustLoopback: true})
	got := r.Resolve("127.0.0.1", func(h string) string {
		if h == "X-Real-IP" {
			return "198.51.100.20"
		}
		return ""
	})
	if got != "198.51.100.20" {
		t.Fatalf("expected X-Real-IP, got %q", got)
	}
}

func TestResolveSpoofedHeaderFromUntrustedRemote(t *testing.T) {
	r := NewResolver(Config{TrustPrivate: true})
	// 公网客户端伪造 CF / XFF，对端不在信任列表 → 忽略头。
	got := r.Resolve("203.0.113.80", func(h string) string {
		switch h {
		case "CF-Connecting-IP":
			return "1.1.1.1"
		case headerXForwardedFor:
			return "8.8.8.8"
		}
		return ""
	})
	if got != "203.0.113.80" {
		t.Fatalf("expected ignore spoofed headers, got %q", got)
	}
}

func TestResolveExplicitProxyCIDR(t *testing.T) {
	r := NewResolver(Config{Proxies: []string{"203.0.113.0/24"}})
	got := r.Resolve("203.0.113.10", func(h string) string {
		if h == headerXForwardedFor {
			return "198.51.100.1, 203.0.113.10"
		}
		return ""
	})
	if got != "198.51.100.1" {
		t.Fatalf("expected client behind trusted CIDR, got %q", got)
	}
}

func TestResolveAllTrustedFallsBackToRemote(t *testing.T) {
	r := NewResolver(Config{TrustPrivate: true})
	// XFF 全是私网代理，无真实客户端 → 回退 remote。
	got := r.Resolve("10.0.0.2", func(h string) string {
		if h == headerXForwardedFor {
			return "10.0.0.3, 10.0.0.2"
		}
		return ""
	})
	if got != "10.0.0.2" {
		t.Fatalf("expected remote fallback, got %q", got)
	}
}

func TestResolveIPv6(t *testing.T) {
	r := NewResolver(Config{TrustLoopback: true})
	got := r.Resolve("::1", func(h string) string {
		if h == headerXForwardedFor {
			return "2001:db8::1, ::1"
		}
		return ""
	})
	if got != "2001:db8::1" {
		t.Fatalf("expected IPv6 client, got %q", got)
	}
}

func TestMaskIPv4AndIPv6(t *testing.T) {
	cases := map[string]string{
		"1.2.3.4":     "1.2.3.*",
		"192.168.1.1": "192.168.1.*",
		"2001:db8:1:2::3": "2001:db8:1:*",
		"":            "",
		"not-an-ip":   "",
	}
	for in, want := range cases {
		if got := Mask(in); got != want {
			t.Errorf("Mask(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalize(t *testing.T) {
	if got := Normalize("  2001:0db8::0001  "); got != "2001:db8::1" {
		t.Fatalf("Normalize IPv6 = %q", got)
	}
	if got := Normalize("not"); got != "" {
		t.Fatalf("Normalize invalid = %q", got)
	}
}

func TestConfigureDefault(t *testing.T) {
	// 恢复默认，避免污染其他测试包顺序。
	t.Cleanup(func() {
		Configure(Config{TrustPrivate: true, TrustLoopback: true})
	})
	Configure(Config{Proxies: []string{"10.0.0.0/8"}})
	got := FromCtx(nil)
	if got != "" {
		t.Fatalf("nil ctx should be empty, got %q", got)
	}
	r := Default()
	out := r.Resolve("10.1.2.3", func(h string) string {
		if h == "X-Real-IP" {
			return "203.0.113.1"
		}
		return ""
	})
	if out != "203.0.113.1" {
		t.Fatalf("configured default: got %q", out)
	}
}

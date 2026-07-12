package useragent

import "testing"

func TestParseBrowserAndOS(t *testing.T) {
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	info := Parse(ua, "1.2.3.4")

	if info.Browser == "" {
		t.Fatal("expected non-empty browser")
	}
	if info.OS == "" {
		t.Fatal("expected non-empty OS")
	}
	if info.DeviceName == "" {
		t.Fatal("expected non-empty device name")
	}
	if info.IPPrefix != "1.2.3.*" {
		t.Fatalf("expected IP prefix 1.2.3.*, got %q", info.IPPrefix)
	}
}

func TestParseEmptyUAAndIP(t *testing.T) {
	info := Parse("", "")
	if info.Browser != "" || info.OS != "" || info.DeviceName != "" {
		t.Fatalf("expected empty fields for empty UA, got %+v", info)
	}
	if info.IPPrefix != "" {
		t.Fatalf("expected empty IP prefix for empty IP, got %q", info.IPPrefix)
	}
}

func TestParseStoresFullIPAndMaskedPrefix(t *testing.T) {
	info := Parse("Mozilla/5.0", "1.2.3.4")
	if info.IPAddress != "1.2.3.4" {
		t.Fatalf("expected full IP, got %q", info.IPAddress)
	}
	if info.IPPrefix != "1.2.3.*" {
		t.Fatalf("expected masked prefix, got %q", info.IPPrefix)
	}
	// IPv6 也应有脱敏前缀（不再返回空串）。
	info6 := Parse("", "2001:db8:1:2::3")
	if info6.IPAddress != "2001:db8:1:2::3" {
		t.Fatalf("expected normalized IPv6, got %q", info6.IPAddress)
	}
	if info6.IPPrefix != "2001:db8:1:*" {
		t.Fatalf("expected IPv6 mask, got %q", info6.IPPrefix)
	}
}

func TestBuildDeviceName(t *testing.T) {
	cases := []struct {
		browser, os, want string
	}{
		{"Chrome", "macOS", "Chrome on macOS"},
		{"Firefox", "", "Firefox"},
		{"", "Windows", "Windows"},
		{"", "", ""},
	}
	for _, tc := range cases {
		got := buildDeviceName(tc.browser, tc.os)
		if got != tc.want {
			t.Errorf("buildDeviceName(%q,%q) = %q, want %q", tc.browser, tc.os, got, tc.want)
		}
	}
}

func TestTruncateUserAgent(t *testing.T) {
	long := ""
	for i := 0; i < 600; i++ {
		long += "x"
	}
	info := Parse(long, "")
	if len(info.UserAgentRaw) > 512 {
		t.Fatalf("expected UA truncated to <=512, got %d", len(info.UserAgentRaw))
	}
}

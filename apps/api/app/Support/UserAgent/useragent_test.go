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

func TestMaskIPv4(t *testing.T) {
	cases := map[string]string{
		"1.2.3.4":      "1.2.3.*",
		"192.168.1.1":  "192.168.1.*",
		"10.0.0.255":   "10.0.0.*",
		"":             "",
		"not-an-ip":    "",
		"::1":          "", // IPv6 统一空串
		"2001:db8::1":  "",
	}
	for input, want := range cases {
		got := maskIP(input)
		if got != want {
			t.Errorf("maskIP(%q) = %q, want %q", input, got, want)
		}
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

package options

import "testing"

// TestNormalizeAttachmentMIMETypesRejectsActiveContent 验证主动内容类型被 denylist 硬封禁，
// 防止运营者通过允许列表放开 HTML/SVG/JS 等形成存储型 XSS。
func TestNormalizeAttachmentMIMETypesRejectsActiveContent(t *testing.T) {
	for _, mime := range []string{
		"text/html",
		"text/xml",
		"application/xhtml+xml",
		"application/xml",
		"image/svg+xml",
		"application/javascript",
		"text/javascript",
		"application/ecmascript",
		"text/ecmascript",
		// 大小写不敏感。
		"TEXT/HTML",
		"Image/SVG+XML",
	} {
		if _, ok := normalizeAttachmentMIMETypes(mime); ok {
			t.Fatalf("expected active content MIME %q to be rejected, but it was accepted", mime)
		}
	}
}

// TestNormalizeAttachmentMIMETypesAcceptsSafeAndWildcard 验证安全类型与通配符仍被接受，
// denylist 不误伤正常配置。
func TestNormalizeAttachmentMIMETypesAcceptsSafeAndWildcard(t *testing.T) {
	cases := map[string]string{
		"image/jpeg":                       "image/jpeg",
		"image/png,image/webp":             "image/png,image/webp",
		"application/pdf,text/plain":       "application/pdf,text/plain",
		"image/*":                          "image/*",
		"image/jpeg, image/png":            "image/jpeg,image/png", // 去空白
	}
	for input, expected := range cases {
		got, ok := normalizeAttachmentMIMETypes(input)
		if !ok {
			t.Fatalf("expected %q to be accepted, but it was rejected", input)
		}
		if got != expected {
			t.Fatalf("normalize %q = %q, want %q", input, got, expected)
		}
	}
}

// TestNormalizeAttachmentMIMETypesRejectsActiveContentAmongList 验证
// 混合列表中只要含一个主动内容类型即整体拒绝（原子性），避免静默丢弃危险项后放行其余。
func TestNormalizeAttachmentMIMETypesRejectsActiveContentAmongList(t *testing.T) {
	if _, ok := normalizeAttachmentMIMETypes("image/png,text/html"); ok {
		t.Fatal("expected mixed list containing text/html to be rejected entirely")
	}
}

// TestIsAttachmentActiveContentType 验证 content 响应层判定函数语义一致。
func TestIsAttachmentActiveContentType(t *testing.T) {
	for _, mime := range []string{"text/html", "image/svg+xml", "application/javascript"} {
		if !IsAttachmentActiveContentType(mime) {
			t.Fatalf("expected %q to be active content", mime)
		}
	}
	if IsAttachmentActiveContentType("image/png") {
		t.Fatal("expected image/png to NOT be active content")
	}
	if !IsAttachmentActiveContentType("  TEXT/HTML  ") {
		t.Fatal("expected whitespace/case-insensitive text/html to be active content")
	}
}

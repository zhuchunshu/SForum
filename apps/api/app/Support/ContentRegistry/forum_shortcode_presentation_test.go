package contentregistry

import "testing"

func TestWrapForumShortcodeHTMLUsesHostOwnedClasses(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		fallback bool
		class    string
	}{
		{name: "topic reference", id: ShortcodeTopicID, class: "sf-shortcode sf-shortcode--reference sf-shortcode--topic"},
		{name: "user reference", id: ShortcodeUserID, class: "sf-shortcode sf-shortcode--reference sf-shortcode--user"},
		{name: "protected block", id: ShortcodeLoginID, class: "sf-shortcode sf-shortcode--block sf-shortcode--protected sf-shortcode--login"},
		{name: "unknown fallback", id: "foreign.shortcode", fallback: true, class: "sf-shortcode sf-shortcode--reference sf-shortcode--fallback"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := WrapForumShortcodeHTML(test.id, `<p>safe</p>`, test.fallback)
			want := `<div class="` + test.class + `"><p>safe</p></div>`
			if got != want {
				t.Fatalf("wrapped shortcode = %q, want %q", got, want)
			}
		})
	}
}

func TestWrapForumShortcodeHTMLEmptyIsEmpty(t *testing.T) {
	if got := WrapForumShortcodeHTML(ShortcodeTopicID, "  ", false); got != "" {
		t.Fatalf("empty shortcode output = %q, want empty", got)
	}
}

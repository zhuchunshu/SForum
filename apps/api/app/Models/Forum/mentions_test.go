package forum

import (
	"reflect"
	"testing"
)

func TestMentionedUsernamesUsesMarkdownTextAndDeduplicates(t *testing.T) {
	got := mentionedUsernames("hello @Alice and @张三, again @alice\n\n`@ignored`\n```\n@also_ignored\n```")
	want := []string{"Alice", "张三"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mentions=%#v want=%#v", got, want)
	}
}

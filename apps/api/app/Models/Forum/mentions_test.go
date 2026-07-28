package forum

import (
	"reflect"
	"testing"
)

func TestMentionedUsernamesUsesMarkdownTextAndDeduplicates(t *testing.T) {
	got := MentionedUsernames("hello @Alice and @张三, @approval_parent, again @alice\n\n`@ignored`\n```\n@also_ignored\n```")
	want := []string{"Alice", "张三", "approval_parent"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mentions=%#v want=%#v", got, want)
	}
}

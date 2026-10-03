package forum

import (
	"reflect"
	"strings"
	"testing"
)

func TestMentionedUsernamesUsesMarkdownTextAndDeduplicates(t *testing.T) {
	got := MentionedUsernames("hello @Alice and @张三, @approval_parent, again @alice\n\n`@ignored`\n```\n@also_ignored\n```")
	want := []string{"Alice", "张三", "approval_parent"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mentions=%#v want=%#v", got, want)
	}
}

// identity.UsernamePolicy 允许 '-'，提及语法必须能解析带连字符的用户名，
// 否则 @zhang-san 只会解析出 zhang，通知静默丢失。
func TestMentionedUsernamesKeepsHyphenatedUsernames(t *testing.T) {
	got := MentionedUsernames("ping @zhang-san and @li_wei, email user@example.com stays out")
	want := []string{"zhang-san", "li_wei"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mentions=%#v want=%#v", got, want)
	}
}

// 用户名长度上限是 identity.username.max_length（最大 64）；这里锁定解析上限。
// 超出上限的连续串会被截断成前缀（RE2 不支持否定前瞻），该前缀不是真实用户名，
// 因此不会产生通知——与前端「整段放弃高亮」的选择等价。
func TestMentionedUsernamesLengthBoundary(t *testing.T) {
	maxName := strings.Repeat("a", mentionUsernameMaxLength)
	if got := MentionedUsernames("@" + maxName); !reflect.DeepEqual(got, []string{maxName}) {
		t.Fatalf("max length mention=%#v want=%#v", got, []string{maxName})
	}

	overlong := maxName + "b"
	truncated := MentionedUsernames("@" + overlong)
	if !reflect.DeepEqual(truncated, []string{maxName}) {
		t.Fatalf("overlong mention=%#v want truncated prefix", truncated)
	}
}

// editor-document 存的是 native Tiptap JSON：必须先还原 Markdown，
// 否则代码块里的 @ 会被当成可见提及（与 goldmark 跳过代码块的合同不符）。
func TestMentionedUsernamesFromSourceIgnoresEditorDocumentCodeBlocks(t *testing.T) {
	raw := `{"type":"doc","content":[` +
		`{"type":"paragraph","content":[{"type":"text","text":"hi @bob"}]},` +
		`{"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"@alice"}]}]}`

	if got := MentionedUsernamesFromSource(raw, SourceFormatEditorDocument); !reflect.DeepEqual(got, []string{"bob"}) {
		t.Fatalf("editor-document mentions=%#v want=[bob]", got)
	}
}

func TestMentionedUsernamesFromSourceKeepsMarkdownAndFallsBackOnInvalidJSON(t *testing.T) {
	markdown := "hello @alice\n\n```\n@ignored\n```"
	if got := MentionedUsernamesFromSource(markdown, SourceFormatMarkdown); !reflect.DeepEqual(got, []string{"alice"}) {
		t.Fatalf("markdown mentions=%#v want=[alice]", got)
	}

	// 还原失败不得让创建/审批路径报错：退回原文解析即可。
	if got := MentionedUsernamesFromSource("broken @alice", SourceFormatEditorDocument); !reflect.DeepEqual(got, []string{"alice"}) {
		t.Fatalf("fallback mentions=%#v want=[alice]", got)
	}
}

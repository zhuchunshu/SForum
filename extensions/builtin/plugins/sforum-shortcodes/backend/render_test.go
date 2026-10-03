package main

import (
	"strings"
	"testing"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func TestRenderUserCardProducesValidDOMAndEscapedPath(t *testing.T) {
	t.Parallel()
	rendered := renderUserCard(publicUser{
		ID: 42, Username: `member/child?role=admin#bio\tail`, DisplayName: `<b>Alice</b>`,
	}, copyFor("en-US"))
	nodes := parseHTMLFragment(t, rendered)
	anchor := requireElement(t, nodes, "a", 1)[0]
	if href := attribute(anchor, "href"); href != `/u/member%2Fchild%3Frole=admin%23bio%5Ctail` {
		t.Fatalf("profile href = %q", href)
	}
	if textContent(anchor) != `<b>Alice</b>` {
		t.Fatalf("profile text = %q", textContent(anchor))
	}
	if strings.Contains(rendered, `href=\"`) {
		t.Fatalf("rendered HTML contains literal escape backslashes: %q", rendered)
	}
}

func TestRenderReferenceCardsFailClosedForInvalidProjectionFields(t *testing.T) {
	t.Parallel()
	label := copyFor("zh-CN")
	for name, rendered := range map[string]string{
		"empty username": renderUserCard(publicUser{ID: 1, DisplayName: "Name"}, label),
		"dot username":   renderUserCard(publicUser{ID: 1, Username: "..", DisplayName: "Name"}, label),
		"empty slug":     renderCategoryCard(publicCategory{ID: 1, Name: "Category"}, label),
		"empty name":     renderCategoryCard(publicCategory{ID: 1, Slug: "general"}, label),
	} {
		if rendered != "" {
			t.Fatalf("%s rendered malformed projection: %q", name, rendered)
		}
	}
	user := parseHTMLFragment(t, renderUserCard(publicUser{ID: 1, Username: "alice"}, label))
	if textContent(requireElement(t, user, "a", 1)[0]) != "alice" {
		t.Fatal("empty display name must fall back to the public username")
	}
}

func TestSafeLinkURLRejectsCredentialsAndUnsafeProtocols(t *testing.T) {
	t.Parallel()
	for _, value := range []string{
		"javascript:alert(1)", "data:text/html,unsafe", "//example.com/path",
		"https://user:secret@example.com/path", "https://example.com/" + strings.Repeat("x", 2049),
	} {
		if safe := safeLinkURL(value); safe != "" {
			t.Fatalf("unsafe URL %q accepted as %q", value, safe)
		}
	}
	if safe := safeLinkURL("https://example.com/path?q=1#section"); safe != "https://example.com/path?q=1#section" {
		t.Fatalf("safe URL changed to %q", safe)
	}
}

func TestRenderFriendLinksDOMSkipsBlankRowsAndQuarantinesInvalidURLs(t *testing.T) {
	t.Parallel()
	invalidOnly := renderFriendLinksBlock([]publicFriendLink{
		{ID: 0, Name: "No ID", URL: "https://example.com"},
		{ID: 1, Name: "  ", URL: "https://example.com"},
	}, copyFor("zh-CN"))
	if invalidOnly != "" {
		t.Fatalf("invalid-only friend links produced output: %q", invalidOnly)
	}

	rendered := renderFriendLinksBlock([]publicFriendLink{
		{ID: 1, Name: "Safe", URL: `https://example.com/a?x=1&y=2`, Description: `<img src=x onerror=alert(1)>`},
		{ID: 2, Name: "Credentials", URL: "https://user:secret@example.com"},
	}, copyFor("en-US"))
	nodes := parseHTMLFragment(t, rendered)
	items := requireElement(t, nodes, "li", 2)
	anchors := requireElement(t, nodes, "a", 1)
	if href := attribute(anchors[0], "href"); href != "https://example.com/a?x=1&y=2" {
		t.Fatalf("friend-link href = %q", href)
	}
	if textContent(items[1]) != "Credentials" || len(elements(items[1], "a")) != 0 {
		t.Fatalf("credential URL was not reduced to plain text: %q", textContent(items[1]))
	}
	if strings.Contains(rendered, "<img") || !strings.Contains(textContent(items[0]), "<img src=x onerror=alert(1)>") {
		t.Fatalf("description was not escaped as text: %q", rendered)
	}
}

func TestBoundedTextIncludesEllipsisWithinUnicodeLimit(t *testing.T) {
	t.Parallel()
	result := boundedText(strings.Repeat("界", 600))
	if got := len([]rune(result)); got != 512 {
		t.Fatalf("bounded text has %d runes, want 512", got)
	}
	if !strings.HasSuffix(result, "…") {
		t.Fatal("bounded text is missing ellipsis")
	}
}

func parseHTMLFragment(t *testing.T, value string) []*xhtml.Node {
	t.Helper()
	contextNode := &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := xhtml.ParseFragment(strings.NewReader(value), contextNode)
	if err != nil {
		t.Fatalf("parse HTML fragment: %v", err)
	}
	return nodes
}

func requireElement(t *testing.T, roots []*xhtml.Node, tag string, count int) []*xhtml.Node {
	t.Helper()
	found := make([]*xhtml.Node, 0, count)
	for _, root := range roots {
		found = append(found, elements(root, tag)...)
	}
	if len(found) != count {
		t.Fatalf("found %d <%s> elements, want %d", len(found), tag, count)
	}
	return found
}

func elements(root *xhtml.Node, tag string) []*xhtml.Node {
	var found []*xhtml.Node
	var visit func(*xhtml.Node)
	visit = func(node *xhtml.Node) {
		if node.Type == xhtml.ElementNode && node.Data == tag {
			found = append(found, node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	return found
}

func attribute(node *xhtml.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

func textContent(root *xhtml.Node) string {
	var builder strings.Builder
	var visit func(*xhtml.Node)
	visit = func(node *xhtml.Node) {
		if node.Type == xhtml.TextNode {
			builder.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	return builder.String()
}

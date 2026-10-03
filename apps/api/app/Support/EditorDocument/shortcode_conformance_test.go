package editordocument

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type shortcodeFixture struct {
	Version string                 `json:"version"`
	Cases   []shortcodeFixtureCase `json:"cases"`
}

type shortcodeFixtureCase struct {
	Name         string                    `json:"name"`
	ResourceKind string                    `json:"resourceKind"`
	Markdown     string                    `json:"markdown"`
	Activated    bool                      `json:"activated"`
	Canonical    string                    `json:"canonical"`
	Nodes        []shortcodeFixtureSummary `json:"nodes"`
}

type shortcodeFixtureSummary struct {
	Type      string         `json:"type"`
	ID        string         `json:"id"`
	Arguments map[string]any `json:"arguments"`
	Depth     int            `json:"depth"`
}

func TestShortcodeTextV1ConformanceFixture(t *testing.T) {
	t.Parallel()
	fixturePath := filepath.Join("..", "..", "..", "..", "..", "contracts", "fixtures", "shortcode-text-v1.json")
	body, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture shortcodeFixture
	if err := json.Unmarshal(body, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if fixture.Version != ShortcodeTextVersion {
		t.Fatalf("fixture version = %q", fixture.Version)
	}

	for _, test := range fixture.Cases {
		test := test
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			doc, activated, err := ParseShortcodeMarkdown(test.Markdown, test.ResourceKind)
			if err != nil {
				t.Fatalf("ParseShortcodeMarkdown: %v", err)
			}
			if activated != test.Activated {
				t.Fatalf("activated = %v, want %v; doc=%#v", activated, test.Activated, doc)
			}
			if !activated {
				return
			}
			native, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			accepted, err := Accept(Input{NativeJSON: native, Schema: CoreSchema(), ResourceKind: test.ResourceKind})
			if err != nil {
				t.Fatalf("Accept: %v", err)
			}
			if accepted.Markdown != test.Canonical {
				t.Fatalf("canonical:\n%s\nwant:\n%s", accepted.Markdown, test.Canonical)
			}
			actual := summarizeShortcodes(accepted.Native.Content, 0)
			actualJSON, _ := json.Marshal(actual)
			expectedJSON, _ := json.Marshal(test.Nodes)
			if string(actualJSON) != string(expectedJSON) {
				t.Fatalf("nodes = %#v, want %#v", actual, test.Nodes)
			}
		})
	}
}

func TestStructuredShortcodeValidationFailsClosed(t *testing.T) {
	t.Parallel()
	validRef := Node{Type: ShortcodeRefNode, Attrs: map[string]any{
		"id": "sforum-shortcodes.user", "contractVersion": "sforum-shortcodes.user@1",
		"arguments": map[string]any{"userId": int64(42)},
	}}
	validBlock := Node{Type: ShortcodeBlockNode, Attrs: map[string]any{
		"id": "sforum-shortcodes.login", "contractVersion": "sforum-shortcodes.login@1",
		"arguments": map[string]any{},
	}, Content: []Node{{Type: "paragraph", Content: []Node{{Type: "text", Text: "secret"}}}}}

	tests := map[string]struct {
		doc           Document
		resourceKind  string
		referencePath []ShortcodeReference
	}{
		"unsupported identity":  {doc: shortcodeDoc(mutateNode(validRef, func(node *Node) { node.Attrs["id"] = "sforum-shortcodes.unknown" }))},
		"unsupported attribute": {doc: shortcodeDoc(mutateNode(validRef, func(node *Node) { node.Attrs["snapshot"] = "forbidden" }))},
		"unsupported contract":  {doc: shortcodeDoc(mutateNode(validRef, func(node *Node) { node.Attrs["contractVersion"] = "sforum-shortcodes.user@2" }))},
		"missing argument":      {doc: shortcodeDoc(mutateNode(validRef, func(node *Node) { node.Attrs["arguments"] = map[string]any{} }))},
		"unknown argument":      {doc: shortcodeDoc(mutateNode(validRef, func(node *Node) { node.Attrs["arguments"] = map[string]any{"userId": int64(42), "name": "alice"} }))},
		"float argument":        {doc: shortcodeDoc(mutateNode(validRef, func(node *Node) { node.Attrs["arguments"] = map[string]any{"userId": 42.5} }))},
		"reference body":        {doc: shortcodeDoc(mutateNode(validRef, func(node *Node) { node.Content = []Node{{Type: "paragraph"}} }))},
		"protected ref type":    {doc: shortcodeDoc(mutateNode(validBlock, func(node *Node) { node.Type = ShortcodeRefNode }))},
		"empty protected body":  {doc: shortcodeDoc(mutateNode(validBlock, func(node *Node) { node.Content = nil }))},
		"inline placement":      {doc: Document{Type: "doc", Content: []Node{{Type: "paragraph", Content: []Node{validRef}}}}},
		"list placement":        {doc: Document{Type: "doc", Content: []Node{{Type: "bulletList", Content: []Node{{Type: "listItem", Content: []Node{{Type: "paragraph", Content: []Node{validRef}}}}}}}}},
		"only-author topic": {doc: shortcodeDoc(mutateNode(validBlock, func(node *Node) {
			node.Attrs["id"] = "sforum-shortcodes.only-author"
			node.Attrs["contractVersion"] = "sforum-shortcodes.only-author@1"
		})), resourceKind: "topic"},
		"cycle input": {doc: shortcodeDoc(validRef), referencePath: []ShortcodeReference{{ID: "sforum-shortcodes.user", StableID: 42}}},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			body, _ := json.Marshal(test.doc)
			if _, err := Accept(Input{NativeJSON: body, Schema: CoreSchema(), ResourceKind: test.resourceKind, ReferencePath: test.referencePath}); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestStructuredShortcodeRejectsDuplicateJSONArguments(t *testing.T) {
	t.Parallel()
	native := []byte(`{"type":"doc","content":[{"type":"sforumShortcodeRef","attrs":{"id":"sforum-shortcodes.user","contractVersion":"sforum-shortcodes.user@1","arguments":{"userId":1,"userId":2}}}]}`)
	if _, err := Accept(Input{NativeJSON: native, Schema: CoreSchema(), ResourceKind: "topic"}); err == nil {
		t.Fatal("expected duplicate argument rejection")
	}
}

func TestStructuredShortcodeBudgetsFailClosed(t *testing.T) {
	t.Parallel()
	var refs []Node
	for index := int64(1); index <= ShortcodeMaxNodes+1; index++ {
		refs = append(refs, Node{Type: ShortcodeRefNode, Attrs: map[string]any{
			"id": "sforum-shortcodes.user", "contractVersion": "sforum-shortcodes.user@1",
			"arguments": map[string]any{"userId": index},
		}})
	}
	assertShortcodeDocumentRejected(t, Document{Type: "doc", Content: refs}, "topic")

	deep := Node{Type: "paragraph", Content: []Node{{Type: "text", Text: "secret"}}}
	for index := 0; index < ShortcodeMaxDepth+1; index++ {
		deep = Node{Type: ShortcodeBlockNode, Attrs: map[string]any{
			"id": "sforum-shortcodes.login", "contractVersion": "sforum-shortcodes.login@1", "arguments": map[string]any{},
		}, Content: []Node{deep}}
	}
	assertShortcodeDocumentRejected(t, Document{Type: "doc", Content: []Node{deep}}, "topic")
}

func TestProtectedFallbackNeverTraversesChildren(t *testing.T) {
	t.Parallel()
	const secret = "M2_PROTECTED_CHILD_SECRET"
	doc := Document{Type: "doc", Content: []Node{{
		Type: ShortcodeBlockNode,
		Attrs: map[string]any{
			"id": "sforum-shortcodes.login", "contractVersion": "sforum-shortcodes.login@1", "arguments": map[string]any{},
		},
		Content: []Node{{Type: "paragraph", Content: []Node{
			{Type: "text", Text: secret},
			{Type: "sforumEmoji", Attrs: map[string]any{"name": secret, "label": secret, "native": "x"}},
		}}},
	}}}
	body, _ := json.Marshal(doc)
	accepted, err := Accept(Input{NativeJSON: body, Schema: CoreSchema(), ResourceKind: "topic"})
	if err != nil {
		t.Fatal(err)
	}
	for label, value := range map[string]string{
		"html": accepted.HTMLSanitized, "plain": accepted.PlainText, "excerpt": accepted.Excerpt, "search": accepted.SearchText,
	} {
		if strings.Contains(value, secret) {
			t.Fatalf("%s leaked protected child: %q", label, value)
		}
	}
	if !strings.Contains(accepted.Markdown, secret) {
		t.Fatal("authorized canonical source must retain protected child")
	}
}

func TestShortcodeNormalizationProducesDeterministicHash(t *testing.T) {
	t.Parallel()
	first := []byte(`{"type":"doc","content":[{"type":"sforumShortcodeRef","attrs":{"id":"sforum-shortcodes.user","contractVersion":"sforum-shortcodes.user@1","arguments":{"userId":42}}}]}`)
	second := []byte(`{ "content": [ { "attrs": { "arguments": { "userId": 42 }, "contractVersion": "sforum-shortcodes.user@1", "id": "sforum-shortcodes.user" }, "type": "sforumShortcodeRef" } ], "type": "doc" }`)
	a, err := Accept(Input{NativeJSON: first, Schema: CoreSchema(), ResourceKind: "topic"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Accept(Input{NativeJSON: second, Schema: CoreSchema(), ResourceKind: "topic"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ContentHash != b.ContentHash || a.Markdown != b.Markdown || !reflect.DeepEqual(a.Native, b.Native) {
		t.Fatalf("normalization drift: %#v %#v", a, b)
	}
}

func summarizeShortcodes(nodes []Node, depth int) []shortcodeFixtureSummary {
	var result []shortcodeFixtureSummary
	for _, node := range nodes {
		nextDepth := depth
		if node.Type == ShortcodeRefNode || node.Type == ShortcodeBlockNode {
			nextDepth++
			nodeType := "ref"
			if node.Type == ShortcodeBlockNode {
				nodeType = "block"
			}
			arguments, _ := node.Attrs["arguments"].(map[string]any)
			arguments = conformanceFixtureArguments(arguments)
			result = append(result, shortcodeFixtureSummary{
				Type: nodeType, ID: node.Attrs["id"].(string), Arguments: arguments, Depth: nextDepth,
			})
		}
		result = append(result, summarizeShortcodes(node.Content, nextDepth)...)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func conformanceFixtureArguments(arguments map[string]any) map[string]any {
	result := make(map[string]any, len(arguments))
	for key, value := range arguments {
		if id, ok := value.(int64); ok && id > 9007199254740991 {
			result[key] = strconv.FormatInt(id, 10)
		} else {
			result[key] = value
		}
	}
	return result
}

func mutateNode(node Node, mutate func(*Node)) Node {
	attrs := make(map[string]any, len(node.Attrs))
	for key, value := range node.Attrs {
		attrs[key] = value
	}
	node.Attrs = attrs
	mutate(&node)
	return node
}

func shortcodeDoc(node Node) Document {
	return Document{Type: "doc", Content: []Node{node}}
}

func assertShortcodeDocumentRejected(t *testing.T, doc Document, resourceKind string) {
	t.Helper()
	body, _ := json.Marshal(doc)
	if _, err := Accept(Input{NativeJSON: body, Schema: CoreSchema(), ResourceKind: resourceKind}); err == nil {
		t.Fatal("expected rejection")
	}
}

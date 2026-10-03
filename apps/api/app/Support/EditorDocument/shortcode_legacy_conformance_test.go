package editordocument

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// legacyConversionFixture mirrors contracts/fixtures/shortcode-legacy-conversion-v1.json.
// The fixture is language-neutral: Go and TypeScript conformance tests consume
// the same file and must reach the same classification and canonical output.
type legacyConversionFixture struct {
	Version                     string                 `json:"version"`
	Report                      string                 `json:"report"`
	Declarations                []legacyDeclaration    `json:"declarations"`
	LegacyAliases               []legacyAlias          `json:"legacyAliases"`
	RecognizedLegacyUnsupported []legacyUnsupported    `json:"recognizedLegacyUnsupported"`
	Cases                       []legacyConversionCase `json:"cases"`
	StructuredCases             []legacyStructuredCase `json:"structuredCases"`
}

type legacyDeclaration struct {
	Name            string `json:"name"`
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Kind            string `json:"kind"`
	CanonicalArg    string `json:"canonicalArgument"`
	BracketArg      string `json:"bracketArgument"`
	CommentOnly     bool   `json:"commentOnly"`
	FallbackCode    string `json:"fallbackCode"`
	LegacySyntax    string `json:"legacySyntax"`
}

type legacyAlias struct {
	Name              string `json:"name"`
	TargetDeclaration string `json:"targetDeclaration"`
	BracketArgument   string `json:"bracketArgument"`
	Note              string `json:"note"`
	LegacySyntax      string `json:"legacySyntax"`
}

type legacyUnsupported struct {
	Name         string `json:"name"`
	Note         string `json:"note"`
	LegacySyntax string `json:"legacySyntax"`
}

type legacyConversionCase struct {
	Name         string                    `json:"name"`
	ResourceKind string                    `json:"resourceKind"`
	Legacy       string                    `json:"legacy"`
	Expected     string                    `json:"expected"`
	Note         string                    `json:"note"`
	Canonical    string                    `json:"canonical"`
	Nodes        []shortcodeFixtureSummary `json:"nodes"`
}

type legacyStructuredCase struct {
	Name         string          `json:"name"`
	ResourceKind string          `json:"resourceKind"`
	Authority    string          `json:"authority"`
	Expected     string          `json:"expected"`
	Note         string          `json:"note"`
	Document     json.RawMessage `json:"document"`
}

func readLegacyConversionFixture(t *testing.T) legacyConversionFixture {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "contracts", "fixtures", "shortcode-legacy-conversion-v1.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture legacyConversionFixture
	if err := json.Unmarshal(body, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if fixture.Version != ShortcodeTextVersion {
		t.Fatalf("fixture version = %q, want %q", fixture.Version, ShortcodeTextVersion)
	}
	if err := validateLegacyFixtureShape(fixture); err != nil {
		t.Fatalf("invalid fixture shape: %v", err)
	}
	return fixture
}

func validateLegacyFixtureShape(fixture legacyConversionFixture) error {
	seen := map[string]bool{}
	for _, declaration := range fixture.Declarations {
		if seen[declaration.ID] {
			return errors.New("duplicate declaration id " + declaration.ID)
		}
		seen[declaration.ID] = true
		switch declaration.Kind {
		case "reference", "protected":
		default:
			return errors.New("unknown declaration kind " + declaration.Kind + " for " + declaration.ID)
		}
		if declaration.FallbackCode == "" {
			return errors.New("missing fallback code for " + declaration.ID)
		}
	}
	for _, alias := range fixture.LegacyAliases {
		if !seen[alias.TargetDeclaration] {
			return errors.New("alias targets undeclared id " + alias.TargetDeclaration)
		}
	}
	for _, unsupported := range fixture.RecognizedLegacyUnsupported {
		if seen["sforum-shortcodes."+unsupported.Name] {
			return errors.New("unsupported name is also a declaration: " + unsupported.Name)
		}
	}
	for _, test := range fixture.Cases {
		switch test.Expected {
		case "converted", "literal", "invalid", "unsupported", "over-limit":
		default:
			return errors.New("unknown classification " + test.Expected + " for " + test.Name)
		}
		if test.ResourceKind != "topic" && test.ResourceKind != "comment" {
			return errors.New("unknown resourceKind " + test.ResourceKind + " for " + test.Name)
		}
		if test.Expected == "converted" && test.Canonical == "" {
			return errors.New("converted case missing canonical: " + test.Name)
		}
		if test.Expected != "converted" && len(test.Nodes) != 0 {
			return errors.New("non-converted case must not list nodes: " + test.Name)
		}
	}
	for _, test := range fixture.StructuredCases {
		if test.Authority != "both" && test.Authority != "host" {
			return errors.New("unknown authority " + test.Authority + " for " + test.Name)
		}
		switch test.Expected {
		case "invalid", "over-limit":
		default:
			return errors.New("unknown structured classification " + test.Expected + " for " + test.Name)
		}
	}
	return nil
}

func TestLegacyConversionDeclarationParity(t *testing.T) {
	t.Parallel()
	fixture := readLegacyConversionFixture(t)
	if len(fixture.Declarations) != len(shortcodeDeclarations) {
		t.Fatalf("fixture declarations = %d, Host declarations = %d", len(fixture.Declarations), len(shortcodeDeclarations))
	}
	for index, declaration := range shortcodeDeclarations {
		want := fixture.Declarations[index]
		switch strings.TrimSpace(want.CanonicalArg) {
		case "":
			if declaration.CanonicalArg != "" {
				t.Fatalf("declaration %s canonical argument drift: got %q", declaration.ID, declaration.CanonicalArg)
			}
		default:
			if declaration.CanonicalArg != want.CanonicalArg {
				t.Fatalf("declaration %s canonical argument drift: got %q, want %q", declaration.ID, declaration.CanonicalArg, want.CanonicalArg)
			}
		}
		if declaration.Name != want.Name || declaration.ID != want.ID || declaration.Version != want.ContractVersion ||
			declaration.BracketArg != want.BracketArg || declaration.CommentOnly != want.CommentOnly ||
			declaration.FallbackCode != want.FallbackCode {
			t.Fatalf("declaration drift for %s:\nHost: %+v\nFixture: %+v", declaration.ID, declaration, want)
		}
		if (want.Kind == "protected") != (declaration.Kind == shortcodeProtected) {
			t.Fatalf("declaration %s kind drift: Host %v, fixture %q", declaration.ID, declaration.Kind, want.Kind)
		}
	}
}

func TestLegacyConversionAliasAndUnsupportedParity(t *testing.T) {
	t.Parallel()
	fixture := readLegacyConversionFixture(t)
	if len(fixture.LegacyAliases) != 1 || fixture.LegacyAliases[0].Name != "topic-tag" {
		t.Fatalf("aliases = %+v, want exactly the topic-tag alias", fixture.LegacyAliases)
	}
	alias := fixture.LegacyAliases[0]
	if alias.TargetDeclaration != "sforum-shortcodes.category" || alias.BracketArgument != "tag_id" {
		t.Fatalf("topic-tag alias = %+v, must map tag_id to category", alias)
	}
	if len(fixture.RecognizedLegacyUnsupported) != 1 || fixture.RecognizedLegacyUnsupported[0].Name != "password" {
		t.Fatalf("unsupported = %+v, want exactly password", fixture.RecognizedLegacyUnsupported)
	}
	// The alias and the legacy unsupported name must never be declarations.
	for _, declaration := range shortcodeDeclarations {
		if declaration.Name == "topic-tag" || declaration.Name == "password" || declaration.Name == "friend_links" {
			t.Fatalf("legacy-only name must not be a Host declaration: %s", declaration.Name)
		}
	}
}

func TestLegacyConversionTextCases(t *testing.T) {
	t.Parallel()
	fixture := readLegacyConversionFixture(t)
	for _, test := range fixture.Cases {
		test := test
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			doc, activated, err := ParseShortcodeMarkdown(test.Legacy, test.ResourceKind)
			if err != nil {
				t.Fatalf("ParseShortcodeMarkdown: %v", err)
			}
			if test.Expected == "converted" {
				if !activated {
					t.Fatalf("expected conversion, got literal input %q", test.Legacy)
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
					t.Fatalf("canonical:\n%q\nwant:\n%q", accepted.Markdown, test.Canonical)
				}
				actual := summarizeShortcodes(accepted.Native.Content, 0)
				actualJSON, _ := json.Marshal(actual)
				expectedJSON, _ := json.Marshal(test.Nodes)
				if string(actualJSON) != string(expectedJSON) {
					t.Fatalf("nodes = %#v, want %#v", actual, test.Nodes)
				}
				assertLegacyFallbackCodes(t, accepted.Native, fixture)
				return
			}
			if activated {
				t.Fatalf("expected %s classification but input activated: %q", test.Expected, test.Legacy)
			}
		})
	}
}

// assertLegacyFallbackCodes pins the deterministic per-node fallback codes the
// Host resolves for the accepted document, matching the frozen fixture table.
func assertLegacyFallbackCodes(t *testing.T, doc Document, fixture legacyConversionFixture) {
	t.Helper()
	codesByID := map[string]string{}
	for _, declaration := range fixture.Declarations {
		codesByID[declaration.ID] = declaration.FallbackCode
	}
	var collect func(nodes []Node)
	collect = func(nodes []Node) {
		for _, node := range nodes {
			if node.Type == ShortcodeRefNode || node.Type == ShortcodeBlockNode {
				id, _ := node.Attrs["id"].(string)
				code, _, _ := shortcodeFallback(node)
				want, ok := codesByID[id]
				if !ok {
					t.Fatalf("unexpected shortcode id %q in accepted document", id)
				}
				if code != want {
					t.Fatalf("fallback code for %s = %q, want %q", id, code, want)
				}
			}
			collect(node.Content)
		}
	}
	collect(doc.Content)
}

func TestLegacyConversionStructuredCases(t *testing.T) {
	t.Parallel()
	fixture := readLegacyConversionFixture(t)
	for _, test := range fixture.StructuredCases {
		test := test
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			doc, err := Parse(Input{NativeJSON: test.Document})
			if err != nil {
				t.Fatalf("Parse should accept the JSON shape: %v", err)
			}
			// The Host validation authority rejects every structured case.
			_, _, err = validateAndNormalize(doc, CoreSchema(), Input{ResourceKind: test.ResourceKind})
			if !errors.Is(err, ErrInvalidShortcode) {
				t.Fatalf("Host validation error = %v, want ErrInvalidShortcode", err)
			}
			// The full public pipeline must also fail closed.
			if _, err := Accept(Input{NativeJSON: test.Document, Schema: CoreSchema(), ResourceKind: test.ResourceKind}); err == nil {
				t.Fatal("Accept must reject the structured case")
			}
			if test.Authority != "both" && test.Authority != "host" {
				t.Fatalf("unknown authority %q", test.Authority)
			}
		})
	}
}

// TestLegacyConversionReportCompleteness lists every frozen declaration and
// classification so the report cannot silently lose a shortcode.
func TestLegacyConversionReportCompleteness(t *testing.T) {
	t.Parallel()
	fixture := readLegacyConversionFixture(t)
	convertedNames := map[string]bool{}
	for _, test := range fixture.Cases {
		if test.Expected == "converted" {
			for _, node := range test.Nodes {
				convertedNames[node.ID] = true
			}
		}
	}
	for _, declaration := range fixture.Declarations {
		if !convertedNames[declaration.ID] {
			t.Fatalf("declaration %s has no converted fixture case", declaration.ID)
		}
	}
	classificationCounts := map[string]int{}
	for _, test := range fixture.Cases {
		classificationCounts[test.Expected]++
	}
	for _, classification := range []string{"converted", "literal", "invalid", "unsupported", "over-limit"} {
		if classificationCounts[classification] == 0 {
			t.Fatalf("fixture has no %s case", classification)
		}
	}
}

// TestLegacyConversionFallbackTableParity pins the fixture fallback codes to
// the Host fallback labels without depending on plugin availability.
func TestLegacyConversionFallbackTableParity(t *testing.T) {
	t.Parallel()
	fixture := readLegacyConversionFixture(t)
	for _, declaration := range fixture.Declarations {
		code, label, omit := shortcodeFallbackForLocale(
			Node{Type: ShortcodeRefNode, Attrs: map[string]any{"id": declaration.ID}},
			"",
		)
		if code != declaration.FallbackCode {
			t.Fatalf("fallback code for %s = %q, want %q", declaration.ID, code, declaration.FallbackCode)
		}
		if declaration.ID == "sforum-shortcodes.friend-links" && !omit {
			t.Fatal("friend-links fallback must omit public output")
		}
		if label == "" && declaration.ID != "sforum-shortcodes.friend-links" {
			t.Fatalf("missing fallback label for %s", declaration.ID)
		}
	}
	// The frozen Host-owned protected policy codes must stay distinguishable
	// and carry both default-locale and English copy without a plugin.
	for _, code := range []string{"shortcode.protected.unavailable", "shortcode.login.required", "shortcode.reply.required", "shortcode.only_author.private"} {
		if shortcodeFallbackLabel(code, "") == "" || shortcodeFallbackLabel(code, "en") == "" {
			t.Fatalf("Host fallback label missing for %s", code)
		}
	}
}

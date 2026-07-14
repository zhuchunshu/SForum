package themecompiler

import (
	"context"
	"errors"
	htmltemplate "html/template"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHTMLTemplateContextEscapesDynamicValues(t *testing.T) {
	snapshot := compileTestSnapshot(t, fstest.MapFS{
		"templates/security.html": &fstest.MapFile{Data: []byte(`<a href="{{.URL}}" title="{{.Title}}">{{.Body}}</a>`)},
	}, testDigest('1'), Bindings{}, Limits{})
	output, err := snapshot.renderPassive(context.Background(), "templates/security.html", map[string]string{
		"URL": "javascript:alert(1)", "Title": `x" onclick="alert(1)`, "Body": `<script>alert(1)</script>`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, `href="#ZgotmplZ"`) {
		t.Fatalf("unsafe URL was not filtered: %s", output)
	}
	if strings.Contains(output, "<script") || strings.Contains(output, `title="x" onclick=`) {
		t.Fatalf("dynamic XSS escaped its context: %s", output)
	}
	if !strings.Contains(output, `&lt;script&gt;alert(1)&lt;/script&gt;`) {
		t.Fatalf("body was not HTML escaped: %s", output)
	}
}

func TestSnapshotRejectsExecutableAndTrustedContentViewModels(t *testing.T) {
	snapshot := compileTestSnapshot(t, fstest.MapFS{
		"templates/value.html": &fstest.MapFile{Data: []byte(`{{.Value}}`)},
	}, testDigest('2'), Bindings{}, Limits{})
	tests := []any{
		map[string]any{"Value": viewModelWithMethod{}},
		map[string]any{"Value": func() string { return "executed" }},
		map[string]any{"Value": htmltemplate.HTML(`<script>alert(1)</script>`)},
		map[int]string{1: "non-string-key"},
	}
	for _, data := range tests {
		if _, err := snapshot.renderPassive(context.Background(), "templates/value.html", data); !errors.Is(err, ErrInvalidViewModel) {
			t.Fatalf("view model %T error = %v", data, err)
		}
	}
}

func TestSafeHTMLHelperOnlyAcceptsHostProducedValues(t *testing.T) {
	snapshot := compileTestSnapshot(t, fstest.MapFS{
		"templates/content.html": &fstest.MapFile{Data: []byte(`<article>{{safeHTML .Rich}}</article><p>{{.Plain}}</p>`)},
	}, testDigest('3'), Bindings{}, Limits{})
	output, err := snapshot.renderPassive(context.Background(), "templates/content.html", map[string]any{
		"Rich":  NewSafeHTMLFromSanitized(`<p>Sanitized <strong>content</strong>.</p>`),
		"Plain": `<img src=x onerror=alert(1)>`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, `<article><p>Sanitized <strong>content</strong>.</p></article>`) {
		t.Fatalf("Host SafeHTML was not rendered as markup: %s", output)
	}
	if !strings.Contains(output, `&lt;img src=x onerror=alert(1)&gt;`) || strings.Contains(output, `<img src=x`) {
		t.Fatalf("ordinary text bypassed contextual escaping: %s", output)
	}

	for _, test := range []struct {
		forged any
		target error
	}{
		{forged: `<strong>ordinary string</strong>`, target: ErrSafeHTMLRequired},
		{forged: htmltemplate.HTML(`<strong>Go trusted alias</strong>`), target: ErrInvalidViewModel},
		{forged: map[string]any{"value": `<strong>plugin document</strong>`}, target: ErrSafeHTMLRequired},
	} {
		if rendered, err := snapshot.renderPassive(context.Background(), "templates/content.html", map[string]any{
			"Rich": test.forged, "Plain": "plain",
		}); !errors.Is(err, test.target) || rendered != "" {
			t.Fatalf("forged SafeHTML %T rendered %q with error %v", test.forged, rendered, err)
		}
	}
}

func TestSafeHTMLHelperCannotEscapeURLOrAttributeContexts(t *testing.T) {
	snapshot := compileTestSnapshot(t, fstest.MapFS{
		"templates/contexts.html": &fstest.MapFile{Data: []byte(`<a href="{{safeHTML .Rich}}" title="{{safeHTML .Rich}}">link</a>`)},
	}, testDigest('9'), Bindings{}, Limits{})
	output, err := snapshot.renderPassive(context.Background(), "templates/contexts.html", map[string]any{
		"Rich": NewSafeHTMLFromSanitized(`javascript:alert(1)" onclick="alert(2)`),
	})
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(output)
	if strings.Contains(lower, `" onclick="`) || !strings.Contains(output, `href="#ZgotmplZ"`) ||
		!strings.Contains(output, `&#34; onclick=&#34;`) {
		t.Fatalf("SafeHTML escaped its output context: %s", output)
	}
}

type viewModelWithMethod struct{}

func (viewModelWithMethod) Execute() string { return "executed" }

func TestCompilerRejectsUnsafeStaticHTML(t *testing.T) {
	cases := []string{
		`<script>alert(1)</script>`,
		`<scr{{if .Never}}{{end}}ipt>alert(1)</scr{{if .Never}}{{end}}ipt>`,
		`<svg/onload=alert(1)>`,
		`<svg{{if .Never}}{{end}}/onload=alert(1)>`,
		`<img/onerror=alert(1)>`,
		`<img on{{if .Never}}{{end}}error=alert(1)>`,
		`<img src="x" onerror="alert(1)">`,
		`<div style="background:url(x)">x</div>`,
		`<iframe src="https://example.com"></iframe>`,
		`<a href="java&#115;cript:alert(1)">x</a>`,
		"<a href=\"java\nscript:alert(1)\">x</a>",
		`<a href="data:image/svg+xml,unsafe">x</a>`,
		`<a href="file:///etc/passwd">x</a>`,
	}
	for index, source := range cases {
		_, err := NewCompiler(Limits{}).CompileFS(fstest.MapFS{
			"templates/unsafe.html": &fstest.MapFile{Data: []byte(source)},
		}, testDigest(byte('a'+index)), withTestBindingRevision(Bindings{}))
		if !errors.Is(err, ErrUnsafeStaticHTML) {
			t.Fatalf("source %q error = %v", source, err)
		}
	}
}

func TestHTMLTemplateCannotAssembleUnsafeURLAcrossActions(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "dynamic prefix",
			source: `<a href="{{.Prefix}}script:alert(1)">x</a>`,
		},
		{
			name:   "dynamic middle",
			source: `<a href="java{{.Middle}}script:alert(1)">x</a>`,
		},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewCompiler(Limits{}).CompileFS(fstest.MapFS{
				"templates/url.html": &fstest.MapFile{Data: []byte(test.source)},
			}, testDigest(byte('u'+index)), withTestBindingRevision(Bindings{}))
			if !errors.Is(err, ErrUnsafeStaticHTML) {
				t.Fatalf("unsafe URL assembly error = %v", err)
			}
		})
	}
}

func TestCompilerAllowsContextSafeDynamicURLs(t *testing.T) {
	snapshot := compileTestSnapshot(t, fstest.MapFS{
		"templates/url.html": &fstest.MapFile{Data: []byte(`<a href="{{.Full}}">full</a><a href="/users/{{.ID}}">user</a>`)},
	}, testDigest('8'), Bindings{}, Limits{})
	output, err := snapshot.renderPassive(context.Background(), "templates/url.html", map[string]string{
		"Full": "javascript:alert(1)", "ID": `x/\" onclick=\"alert(1)`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(output), "javascript:") || strings.Contains(output, ` onclick=`) {
		t.Fatalf("dynamic URL escaped its context: %s", output)
	}
}

func TestCompilerMasksQuotedActionsAndTemplateCommentsBeforeHTMLInspection(t *testing.T) {
	snapshot := compileTestSnapshot(t, fstest.MapFS{
		"templates/actions.html": &fstest.MapFile{Data: []byte(`{{/* an unmatched " quote is inert here */}}<a href="{{route "home"}}">home</a>`)},
	}, testDigest('7'), Bindings{Routes: map[string]string{"home": "/"}}, Limits{})
	output, err := snapshot.renderPassive(context.Background(), "templates/actions.html", nil)
	if err != nil || output != `<a href="/">home</a>` {
		t.Fatalf("masked action render = %q, %v", output, err)
	}
}

func TestBoundURLHelpersRemainContextEscaped(t *testing.T) {
	snapshot := compileTestSnapshot(t, fstest.MapFS{
		"templates/helpers.html": &fstest.MapFile{Data: []byte(`<a href="{{route "home"}}"><img src="{{asset "logo"}}"></a>`)},
	}, testDigest('5'), Bindings{
		Routes: map[string]string{"home": "javascript:alert(1)"},
		Assets: map[string]string{"logo": "data:text/html,<script>alert(1)</script>"},
	}, Limits{})
	output, err := snapshot.renderPassive(context.Background(), "templates/helpers.html", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(output), "javascript:") || strings.Contains(strings.ToLower(output), "data:text/html") {
		t.Fatalf("unsafe helper binding escaped its URL context: %s", output)
	}
}

func TestMissingBoundHelperValueKeepsTypedError(t *testing.T) {
	snapshot := compileTestSnapshot(t, fstest.MapFS{
		"templates/helper.html": &fstest.MapFile{Data: []byte(`<a href="{{route "missing"}}">missing</a>`)},
	}, testDigest('4'), Bindings{}, Limits{})
	if _, err := snapshot.renderPassive(context.Background(), "templates/helper.html", nil); !errors.Is(err, ErrHelperValueMissing) {
		t.Fatalf("missing helper error = %v", err)
	}
}

func TestCompilerRejectsUnknownAndDangerousBuiltInHelpers(t *testing.T) {
	cases := []string{
		`{{env "SECRET"}}`,
		`{{call .Function}}`,
		`{{html .Value}}`,
		`{{printf "%s" .Value}}`,
	}
	for index, source := range cases {
		_, err := NewCompiler(Limits{}).CompileFS(fstest.MapFS{
			"templates/helper.html": &fstest.MapFile{Data: []byte(source)},
		}, testDigest(byte('k'+index)), withTestBindingRevision(Bindings{}))
		if !errors.Is(err, ErrForbiddenHelper) {
			t.Fatalf("helper source %q error = %v", source, err)
		}
	}
}

func TestCompilerRejectsMissingInvalidAndRecursivePartials(t *testing.T) {
	tests := []struct {
		name   string
		files  fstest.MapFS
		target error
	}{
		{
			name: "missing",
			files: fstest.MapFS{
				"templates/home.html": &fstest.MapFile{Data: []byte(`{{template "partials/missing.html" .}}`)},
			},
			target: ErrInvalidPartial,
		},
		{
			name: "traversal name",
			files: fstest.MapFS{
				"templates/home.html": &fstest.MapFile{Data: []byte(`{{template "../outside.html" .}}`)},
			},
			target: ErrInvalidPartial,
		},
		{
			name: "direct recursion",
			files: fstest.MapFS{
				"partials/loop.html":  &fstest.MapFile{Data: []byte(`{{template "partials/loop.html" .}}`)},
				"templates/home.html": &fstest.MapFile{Data: []byte(`{{template "partials/loop.html" .}}`)},
			},
			target: ErrTemplateRecursion,
		},
		{
			name: "indirect recursion",
			files: fstest.MapFS{
				"partials/a.html":     &fstest.MapFile{Data: []byte(`{{template "partials/b.html" .}}`)},
				"partials/b.html":     &fstest.MapFile{Data: []byte(`{{template "partials/a.html" .}}`)},
				"templates/home.html": &fstest.MapFile{Data: []byte(`{{template "partials/a.html" .}}`)},
			},
			target: ErrTemplateRecursion,
		},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewCompiler(Limits{}).CompileFS(test.files, testDigest(byte('p'+index)), withTestBindingRevision(Bindings{}))
			if !errors.Is(err, test.target) {
				t.Fatalf("error = %v, want %v", err, test.target)
			}
		})
	}
}

func TestCompilerRejectsCallGraphBeyondConfiguredDepth(t *testing.T) {
	files := fstest.MapFS{
		"partials/a.html":     &fstest.MapFile{Data: []byte(`{{template "partials/b.html" .}}`)},
		"partials/b.html":     &fstest.MapFile{Data: []byte(`{{template "partials/c.html" .}}`)},
		"partials/c.html":     &fstest.MapFile{Data: []byte(`end`)},
		"templates/home.html": &fstest.MapFile{Data: []byte(`{{template "partials/a.html" .}}`)},
	}
	_, err := NewCompiler(Limits{MaxCallDepth: 3}).CompileFS(files, testDigest('z'), withTestBindingRevision(Bindings{}))
	if !errors.Is(err, ErrTemplateRecursion) {
		t.Fatalf("depth error = %v", err)
	}
}

func TestCompilerRejectsInconsistentHTMLTemplateContext(t *testing.T) {
	_, err := NewCompiler(Limits{}).CompileFS(fstest.MapFS{
		"templates/context.html": &fstest.MapFile{Data: []byte(`{{if .Open}}<a href="{{end}}{{.Value}}`)},
	}, testDigest('9'), withTestBindingRevision(Bindings{}))
	if !errors.Is(err, ErrInvalidTemplate) {
		t.Fatalf("context error = %v", err)
	}
}

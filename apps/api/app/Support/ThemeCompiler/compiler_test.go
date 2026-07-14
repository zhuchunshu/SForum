package themecompiler

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"
)

func TestCompilerSupportsLayoutsPartialsAndStandardControlActions(t *testing.T) {
	snapshot := compileTestSnapshot(t, fstest.MapFS{
		"layouts/base.html":    &fstest.MapFile{Data: []byte(`<!doctype html><html><body>{{block "body" .}}fallback{{end}}{{template "partials/footer.html" .}}</body></html>`)},
		"partials/footer.html": &fstest.MapFile{Data: []byte(`<footer>{{i18n .Locale "footer"}}</footer>`)},
		"templates/home.html":  &fstest.MapFile{Data: []byte(`{{define "body"}}{{if .Show}}<ul>{{range .Items}}{{with .}}<li>{{.Name}}</li>{{end}}{{end}}</ul>{{else}}hidden{{end}}{{end}}{{template "layouts/base.html" .}}`)},
	}, testDigest('a'), Bindings{
		BindingRevision: testDigest('0'),
		Translations:    map[string]map[string]string{"zh-CN": {"footer": "Translated footer"}},
	}, Limits{})

	output, err := snapshot.renderPassive(context.Background(), "templates/home.html", map[string]any{
		"Show": true, "Locale": "zh-CN",
		"Items": []map[string]string{{"Name": "one"}, {"Name": "two"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"<!doctype html>", "<li>one</li>", "<li>two</li>", "<footer>Translated footer</footer>"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("render missing %q: %s", expected, output)
		}
	}
	if snapshot.Key() != (SnapshotKey{
		PackageDigest: testDigest('a'), CompilerVersion: CompilerVersion, BindingRevision: testDigest('0'),
	}) {
		t.Fatalf("unexpected key: %#v", snapshot.Key())
	}
	if !snapshot.HasTemplate("templates/home.html") || snapshot.HasTemplate("partials/footer.html") {
		t.Fatalf("entry visibility is wrong: %#v", snapshot.Templates())
	}
	infos := snapshot.Templates()
	if len(infos) != 3 || infos[0].Digest == "" {
		t.Fatalf("incomplete immutable metadata: %#v", infos)
	}
}

func TestCompilerBindingsAreDeepCopiedAndDigestIsolated(t *testing.T) {
	source := fstest.MapFS{
		"templates/home.html": &fstest.MapFile{Data: []byte(`<a href="{{route "home"}}"><img src="{{asset "logo"}}" alt="{{i18n .Locale "logo"}}"></a>`)},
	}
	bindings := Bindings{
		BindingRevision: testDigest('1'),
		Assets:          map[string]string{"logo": "/assets/a.png"},
		Routes:          map[string]string{"home": "/a"},
		Translations:    map[string]map[string]string{"en-US": {"logo": "A"}},
	}
	compiler := NewCompiler(Limits{})
	first, err := compiler.CompileFS(source, testDigest('a'), withTestPageViewModels(source, bindings))
	if err != nil {
		t.Fatal(err)
	}
	bindings.Assets["logo"] = "/assets/b.png"
	bindings.Routes["home"] = "/b"
	bindings.Translations["en-US"]["logo"] = "B"
	second, err := compiler.CompileFS(source, testDigest('b'), withTestPageViewModels(source, bindings))
	if err != nil {
		t.Fatal(err)
	}
	firstOutput, err := first.renderPassive(context.Background(), "templates/home.html", map[string]string{"Locale": "en-US"})
	if err != nil {
		t.Fatal(err)
	}
	secondOutput, err := second.renderPassive(context.Background(), "templates/home.html", map[string]string{"Locale": "en-US"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(firstOutput, `/a`) || !strings.Contains(firstOutput, `/assets/a.png`) || !strings.Contains(firstOutput, `alt="A"`) {
		t.Fatalf("first snapshot changed with caller maps: %s", firstOutput)
	}
	if !strings.Contains(secondOutput, `/b`) || !strings.Contains(secondOutput, `/assets/b.png`) || !strings.Contains(secondOutput, `alt="B"`) {
		t.Fatalf("second snapshot did not isolate bindings: %s", secondOutput)
	}
	if first.CompiledCacheKey() == second.CompiledCacheKey() {
		t.Fatalf("digest-keyed compiled templates collided: %q", first.CompiledCacheKey())
	}
}

func TestSnapshotSeparatesCompiledAndRuntimeIdentity(t *testing.T) {
	source := fstest.MapFS{
		"templates/home.html": &fstest.MapFile{Data: []byte(`<a href="{{route "home"}}">home</a>`)},
	}
	compiler := NewCompiler(Limits{})
	first, err := compiler.CompileFS(source, testDigest('a'), withTestPageViewModels(source, Bindings{
		BindingRevision: testDigest('1'), Routes: map[string]string{"home": "/first"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	second, err := compiler.CompileFS(source, testDigest('a'), withTestPageViewModels(source, Bindings{
		BindingRevision: testDigest('2'), Routes: map[string]string{"home": "/second"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if first.CompiledKey() != second.CompiledKey() || first.CompiledCacheKey() != second.CompiledCacheKey() {
		t.Fatalf("same artifact must share compiled identity: %#v, %#v", first.CompiledKey(), second.CompiledKey())
	}
	if first.Key() == second.Key() || first.RuntimeKey() == second.RuntimeKey() {
		t.Fatalf("different binding revisions shared runtime identity: %#v, %#v", first.Key(), second.Key())
	}
	firstOutput, err := first.renderPassive(context.Background(), "templates/home.html", nil)
	if err != nil {
		t.Fatal(err)
	}
	secondOutput, err := second.renderPassive(context.Background(), "templates/home.html", nil)
	if err != nil {
		t.Fatal(err)
	}
	if firstOutput != `<a href="/first">home</a>` || secondOutput != `<a href="/second">home</a>` {
		t.Fatalf("runtime bindings crossed snapshots: %q, %q", firstOutput, secondOutput)
	}
}

func TestSnapshotRenderHasNoFilesystemAccessAndIsConcurrentReadOnly(t *testing.T) {
	counting := &countingFS{FS: fstest.MapFS{
		"partials/item.html":  &fstest.MapFile{Data: []byte(`<li>{{.}}</li>`)},
		"templates/list.html": &fstest.MapFile{Data: []byte(`<ul>{{range .}}{{template "partials/item.html" .}}{{end}}</ul>`)},
	}}
	snapshot := compileTestSnapshot(t, counting, testDigest('c'), Bindings{}, Limits{})
	afterCompile := counting.opens.Load()

	const workers = 32
	const iterations = 40
	var group sync.WaitGroup
	errorsSeen := make(chan error, workers)
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for iteration := 0; iteration < iterations; iteration++ {
				output, err := snapshot.renderPassive(context.Background(), "templates/list.html", []string{"a", "b"})
				if err != nil {
					errorsSeen <- err
					return
				}
				if output != "<ul><li>a</li><li>b</li></ul>" {
					errorsSeen <- errors.New("unexpected concurrent output: " + output)
					return
				}
			}
		}()
	}
	group.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatal(err)
	}
	if got := counting.opens.Load(); got != afterCompile {
		t.Fatalf("render touched filesystem: compile=%d render=%d", afterCompile, got)
	}
}

func TestSnapshotEnforcesMissingKeyOutputAndCancellation(t *testing.T) {
	missing := compileTestSnapshot(t, fstest.MapFS{
		"templates/missing.html": &fstest.MapFile{Data: []byte(`<p>{{.Required}}</p>`)},
	}, testDigest('d'), Bindings{}, Limits{})
	if _, err := missing.renderPassive(context.Background(), "templates/missing.html", map[string]string{}); !errors.Is(err, ErrMissingValue) {
		t.Fatalf("missing key error = %v", err)
	}

	limited := compileTestSnapshot(t, fstest.MapFS{
		"templates/large.html": &fstest.MapFile{Data: []byte(strings.Repeat("x", 64))},
	}, testDigest('e'), Bindings{}, Limits{MaxOutputBytes: 16})
	if output, err := limited.renderPassive(context.Background(), "templates/large.html", nil); !errors.Is(err, ErrOutputLimit) || output != "" {
		t.Fatalf("output limit = %q, %v", output, err)
	}
	exact := compileTestSnapshot(t, fstest.MapFS{
		"templates/exact.html": &fstest.MapFile{Data: []byte(strings.Repeat("x", 16))},
	}, testDigest('3'), Bindings{}, Limits{MaxOutputBytes: 16})
	if output, err := exact.renderPassive(context.Background(), "templates/exact.html", nil); err != nil || len(output) != 16 {
		t.Fatalf("exact output limit = %q, %v", output, err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := missing.renderPassive(cancelled, "templates/missing.html", map[string]string{"Required": "x"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled render = %v", err)
	}
	if _, err := missing.renderPassive(context.Background(), "partials/nope.html", nil); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("hidden/missing entry = %v", err)
	}
}

func TestSnapshotRenderDeadlineStopsWaitingForBlockedExecution(t *testing.T) {
	snapshot := compileTestSnapshot(t, fstest.MapFS{
		"templates/blocked.html": &fstest.MapFile{Data: []byte(`{{range .}}{{if .}}x{{end}}{{end}}`)},
	}, testDigest('6'), Bindings{}, Limits{RenderTimeout: time.Millisecond, MaxOutputBytes: 10 * 1024 * 1024})
	data := make([]bool, 99_000)
	for index := range data {
		data[index] = true
	}
	started := time.Now()
	_, err := snapshot.renderPassive(context.Background(), "templates/blocked.html", data)
	elapsed := time.Since(started)
	if !errors.Is(err, ErrRenderTimeout) {
		t.Fatalf("blocked render error = %v", err)
	}
	if elapsed >= time.Second {
		t.Fatalf("render deadline did not stop waiting: %s", elapsed)
	}
}

func TestCompilerRejectsInvalidDigestAndMissingPageTemplates(t *testing.T) {
	compiler := NewCompiler(Limits{})
	if _, err := compiler.CompileFS(fstest.MapFS{"templates/x.html": &fstest.MapFile{Data: []byte("x")}}, "ABC", Bindings{}); !errors.Is(err, ErrInvalidDigest) {
		t.Fatalf("invalid digest = %v", err)
	}
	if _, err := compiler.CompileFS(fstest.MapFS{"partials/x.html": &fstest.MapFile{Data: []byte("x")}}, testDigest('f'), withTestBindingRevision(Bindings{})); !errors.Is(err, ErrNoTemplates) {
		t.Fatalf("missing page templates = %v", err)
	}
	if _, err := compiler.CompileDir("", testDigest('f'), Bindings{}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("empty package root = %v", err)
	}
}

func TestCompilerRequiresCanonicalBindingRevision(t *testing.T) {
	source := fstest.MapFS{"templates/x.html": &fstest.MapFile{Data: []byte("x")}}
	for _, revision := range []string{"", "ABC", strings.Repeat("A", 64), strings.Repeat("0", 63)} {
		_, err := NewCompiler(Limits{}).CompileFS(source, testDigest('f'), Bindings{BindingRevision: revision})
		if !errors.Is(err, ErrInvalidBindingRevision) {
			t.Fatalf("binding revision %q error = %v", revision, err)
		}
	}
}

func TestCompilerEnforcesSourceCollectionLimits(t *testing.T) {
	tests := []struct {
		name   string
		files  fstest.MapFS
		limits Limits
	}{
		{
			name: "single source bytes",
			files: fstest.MapFS{
				"templates/page.html": &fstest.MapFile{Data: []byte("12345")},
			},
			limits: Limits{MaxSourceBytes: 4},
		},
		{
			name: "total source bytes",
			files: fstest.MapFS{
				"partials/one.html":   &fstest.MapFile{Data: []byte("123")},
				"templates/page.html": &fstest.MapFile{Data: []byte("456")},
			},
			limits: Limits{MaxTotalBytes: 5},
		},
		{
			name: "source files",
			files: fstest.MapFS{
				"partials/one.html":   &fstest.MapFile{Data: []byte("one")},
				"templates/page.html": &fstest.MapFile{Data: []byte("page")},
			},
			limits: Limits{MaxFiles: 1},
		},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewCompiler(test.limits).CompileFS(test.files, testDigest(byte('a'+index)), withTestBindingRevision(Bindings{}))
			if !errors.Is(err, ErrInvalidTemplate) {
				t.Fatalf("source limit error = %v", err)
			}
		})
	}
}

func TestCompileDirReadsSourcesBeforePublishingSnapshot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "templates", "page.html")
	if err := os.WriteFile(path, []byte(`<p>{{.Value}}</p>`), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot, err := NewCompiler(Limits{}).CompileDir(root, testDigest('7'), withTestBindingRevision(Bindings{
		PageViewModels: map[string]PageTemplateBinding{
			"templates/page.html": {PageID: "forum.home", SchemaVersion: "sforum.page.home@1"},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	output, err := snapshot.renderPassive(context.Background(), "templates/page.html", map[string]string{"Value": "still compiled"})
	if err != nil || output != "<p>still compiled</p>" {
		t.Fatalf("render after package removal = %q, %v", output, err)
	}
}

type countingFS struct {
	fs.FS
	opens atomic.Int64
}

func (f *countingFS) Open(name string) (fs.File, error) {
	f.opens.Add(1)
	return f.FS.Open(name)
}

func compileTestSnapshot(t testing.TB, source fs.FS, digest string, bindings Bindings, limits Limits) *Snapshot {
	t.Helper()
	bindings = withTestBindingRevision(bindings)
	bindings = withTestPageViewModels(source, bindings)
	snapshot, err := NewCompiler(limits).CompileFS(source, digest, bindings)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func withTestPageViewModels(source fs.FS, bindings Bindings) Bindings {
	if bindings.PageViewModels == nil {
		bindings.PageViewModels = map[string]PageTemplateBinding{}
	}
	_ = fs.WalkDir(source, "templates", func(name string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && filepath.Ext(name) == ".html" {
			if _, exists := bindings.PageViewModels[name]; !exists {
				bindings.PageViewModels[name] = PageTemplateBinding{
					PageID: "forum.home", SchemaVersion: "sforum.page.home@1",
				}
			}
		}
		return nil
	})
	return bindings
}

func withTestBindingRevision(bindings Bindings) Bindings {
	if bindings.BindingRevision == "" {
		bindings.BindingRevision = testDigest('0')
	}
	return bindings
}

func testDigest(value byte) string {
	const hexadecimal = "0123456789abcdef"
	return strings.Repeat(string(hexadecimal[int(value)%len(hexadecimal)]), 64)
}

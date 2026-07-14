package themecompiler

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
)

func TestThemeCompilerAllocationBudgets(t *testing.T) {
	tests := []struct {
		name      string
		benchmark func(*testing.B)
		maxBytes  int64
		maxAllocs int64
	}{
		{name: "compile small", benchmark: BenchmarkCompileSmall, maxBytes: 48 * 1024, maxAllocs: 224},
		{name: "compile large", benchmark: BenchmarkCompileLarge, maxBytes: 2816 * 1024, maxAllocs: 25_000},
		{name: "render small", benchmark: BenchmarkRenderSmall, maxBytes: 16 * 1024, maxAllocs: 80},
		{name: "render large", benchmark: BenchmarkRenderLarge, maxBytes: 2560 * 1024, maxAllocs: 25_000},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := testing.Benchmark(test.benchmark)
			bytesPerOp := result.AllocedBytesPerOp()
			allocsPerOp := result.AllocsPerOp()
			t.Logf("%s: %d B/op, %d allocs/op", test.name, bytesPerOp, allocsPerOp)
			if bytesPerOp > test.maxBytes {
				t.Fatalf("allocated bytes/op = %d, ceiling = %d", bytesPerOp, test.maxBytes)
			}
			if allocsPerOp > test.maxAllocs {
				t.Fatalf("allocations/op = %d, ceiling = %d", allocsPerOp, test.maxAllocs)
			}
		})
	}
}

func BenchmarkCompileSmall(b *testing.B) {
	benchmarkCompile(b, `<main><h1>{{.Base.SEO.Title}}</h1><sf-home-page></sf-home-page></main>`)
}

func BenchmarkCompileLarge(b *testing.B) {
	benchmarkCompile(b, `<main>`+strings.Repeat(`<section><h2>{{.Base.SEO.Title}}</h2><p>Compiled theme body</p></section>`, 1000)+`<sf-home-page></sf-home-page></main>`)
}

func benchmarkCompile(b *testing.B, source string) {
	files := fstest.MapFS{"templates/page.html": &fstest.MapFile{Data: []byte(source)}}
	compiler := NewCompiler(Limits{})
	bindings := benchmarkBindings(files)
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		if _, err := compiler.CompileFS(files, testDigest('b'), bindings); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderSmall(b *testing.B) {
	benchmarkRender(b, `<main><h1>{{.Base.SEO.Title}}</h1><sf-home-page></sf-home-page></main>`, 1)
}

func BenchmarkRenderLarge(b *testing.B) {
	benchmarkRender(b, `<main>{{range .Topics}}<article><h2>{{.Title}}</h2><p>{{.Excerpt}}</p></article>{{end}}<sf-home-page></sf-home-page></main>`, 1000)
}

func benchmarkRender(b *testing.B, source string, topicCount int) {
	files := fstest.MapFS{
		"templates/page.html": &fstest.MapFile{Data: []byte(source)},
	}
	snapshot, err := NewCompiler(Limits{MaxOutputBytes: 8 * 1024 * 1024}).CompileFS(files, testDigest('r'), benchmarkBindings(files))
	if err != nil {
		b.Fatal(err)
	}
	model := validHomeViewModel()
	model.Topics = make([]TopicSummaryView, topicCount)
	for index := range model.Topics {
		model.Topics[index] = TopicSummaryView{
			ID: int64(index + 1), Title: "A title", URL: "/t/benchmark", Excerpt: "A body with <escaped> content",
		}
	}
	bound, err := CorePageViewModelRegistry().Bind("forum.home", "sforum.page.home@1", testDigest('r'), model)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		if _, err := snapshot.Render(context.Background(), "templates/page.html", bound); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkBindings(files fstest.MapFS) Bindings {
	return withTestPageViewModels(files, Bindings{
		BindingRevision: testDigest('0'),
		Islands: map[string]IslandBinding{
			"sf-home-page": {ComponentID: "forum.component.home_page"},
		},
	})
}

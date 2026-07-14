package themecompiler

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
)

func BenchmarkCompileSmall(b *testing.B) {
	benchmarkCompile(b, `<main><h1>{{.Title}}</h1></main>`)
}

func BenchmarkCompileLarge(b *testing.B) {
	benchmarkCompile(b, `<main>`+strings.Repeat(`<section><h2>{{.Title}}</h2><p>{{.Body}}</p></section>`, 1000)+`</main>`)
}

func benchmarkCompile(b *testing.B, source string) {
	files := fstest.MapFS{"templates/page.html": &fstest.MapFile{Data: []byte(source)}}
	compiler := NewCompiler(Limits{})
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		if _, err := compiler.CompileFS(files, testDigest('b'), withTestPageViewModels(files, withTestBindingRevision(Bindings{}))); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderSmall(b *testing.B) {
	benchmarkRender(b, `<main><h1>{{.Title}}</h1></main>`)
}

func BenchmarkRenderLarge(b *testing.B) {
	benchmarkRender(b, `<main>{{range .Items}}<article><h2>{{.Title}}</h2><p>{{.Body}}</p></article>{{end}}</main>`)
}

func benchmarkRender(b *testing.B, source string) {
	snapshot := compileTestSnapshot(b, fstest.MapFS{
		"templates/page.html": &fstest.MapFile{Data: []byte(source)},
	}, testDigest('r'), Bindings{}, Limits{MaxOutputBytes: 8 * 1024 * 1024})
	items := make([]map[string]string, 1000)
	for index := range items {
		items[index] = map[string]string{"Title": "A title", "Body": "A body with <escaped> content"}
	}
	data := map[string]any{"Title": "A title", "Items": items}
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		if _, err := snapshot.renderPassive(context.Background(), "templates/page.html", data); err != nil {
			b.Fatal(err)
		}
	}
}

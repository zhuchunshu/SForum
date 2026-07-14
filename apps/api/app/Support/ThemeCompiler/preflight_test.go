package themecompiler

import (
	"errors"
	"testing"
	"testing/fstest"
)

func TestCompilerPreflightCompilesEveryInstallablePage(t *testing.T) {
	files := fstest.MapFS{
		"layouts/base.html":   &fstest.MapFile{Data: []byte(`{{define "layouts/base.html"}}<main>{{template "partials/body.html" .}}</main>{{end}}`)},
		"partials/body.html":  &fstest.MapFile{Data: []byte(`{{define "partials/body.html"}}{{if .Show}}{{range .Items}}<p>{{.}}</p>{{end}}{{end}}{{end}}`)},
		"templates/home.html": &fstest.MapFile{Data: []byte(`{{template "layouts/base.html" .}}`)},
	}

	if err := NewCompiler(Limits{}).PreflightFS(files); err != nil {
		t.Fatalf("PreflightFS() error = %v", err)
	}
}

func TestCompilerPreflightRejectsUnsafeAndInvalidUnusedPages(t *testing.T) {
	tests := []struct {
		name   string
		source string
		target error
	}{
		{name: "static script", source: `<scr{{if .Never}}{{end}}ipt>alert(1)</script>`, target: ErrUnsafeStaticHTML},
		{name: "assembled URL", source: `<a href="java{{.Middle}}script:alert(1)">x</a>`, target: ErrUnsafeStaticHTML},
		{name: "forbidden helper", source: `{{printf "%s" .Value}}`, target: ErrForbiddenHelper},
		{name: "inconsistent context", source: `{{if .Open}}<a href="{{end}}{{.Value}}`, target: ErrInvalidTemplate},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			files := fstest.MapFS{
				"templates/active.html": &fstest.MapFile{Data: []byte(`<main>{{.Title}}</main>`)},
				"templates/unused.html": &fstest.MapFile{Data: []byte(test.source)},
			}
			if err := NewCompiler(Limits{}).PreflightFS(files); !errors.Is(err, test.target) {
				t.Fatalf("PreflightFS() error = %v, want %v", err, test.target)
			}
		})
	}
}

func TestCompilerPreflightAllowsL0OnlyTheme(t *testing.T) {
	if err := NewCompiler(Limits{}).PreflightFS(fstest.MapFS{
		"assets/theme.css": &fstest.MapFile{Data: []byte(`body { color: black; }`)},
	}); err != nil {
		t.Fatalf("PreflightFS() error = %v", err)
	}
}

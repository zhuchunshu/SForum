package themecompiler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"testing/fstest"
)

func TestPluginPageViewModelContractValidatesAndSealsExactJSON(t *testing.T) {
	schema := []byte(`{
  "$schema":"https://json-schema.org/draft/2020-12/schema",
  "type":"object",
  "required":["title","state","sequence"],
  "additionalProperties":false,
  "properties":{
    "title":{"type":"string","maxLength":80},
    "state":{"type":"string","enum":["published","archived"]},
    "sequence":{"type":"integer","maximum":9007199254740993}
  }
}`)
	schemaDigest := sha256.Sum256(schema)
	contract, err := CompilePluginPageViewModelContract(
		"plugin.demo.template.article", "plugin.demo.page.article.data@1",
		hex.EncodeToString(schemaDigest[:]), schema,
	)
	if err != nil {
		t.Fatal(err)
	}
	themeDigest := strings.Repeat("a", 64)
	bound, err := contract.Bind(themeDigest, json.RawMessage(`{"title":"<script>alert(1)</script>","state":"published","sequence":9007199254740993}`), PageSEOView{Title: "Article"})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := NewCompiler(Limits{}).CompileFS(fstest.MapFS{
		"templates/article.html": {Data: []byte(`<main>{{.title}} / {{.state}} / {{.sequence}}</main>`)},
	}, themeDigest, Bindings{
		BindingRevision: strings.Repeat("b", 64),
		PageViewModels: map[string]PageTemplateBinding{
			"templates/article.html": {
				PageID: "plugin.demo.template.article", SchemaVersion: "plugin.demo.page.article.data@1",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	output, err := snapshot.Render(context.Background(), "templates/article.html", bound)
	if err != nil {
		t.Fatal(err)
	}
	html := output.HTMLSegments()[0].String()
	if !strings.Contains(html, "&lt;script&gt;alert(1)&lt;/script&gt; / published / 9007199254740993") || strings.Contains(html, "<script>") {
		t.Fatalf("rendered HTML = %q", html)
	}
	driftSchema := append([]byte(nil), schema...)
	driftSchema = bytes.Replace(driftSchema, []byte(`"maximum":9007199254740993`), []byte(`"maximum":9007199254740994`), 1)
	driftDigest := sha256.Sum256(driftSchema)
	driftContract, err := CompilePluginPageViewModelContract(
		"plugin.demo.template.article", "plugin.demo.page.article.data@1",
		hex.EncodeToString(driftDigest[:]), driftSchema,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := driftContract.Rebind(bound, strings.Repeat("c", 64)); !errors.Is(err, ErrViewModelSchema) {
		t.Fatalf("schema-drift rebind error = %v", err)
	}

	for _, invalid := range []string{
		`{"title":"missing state","sequence":1}`,
		`{"title":"wrong enum","state":"draft","sequence":1}`,
		`{"title":"extra","state":"published","sequence":1,"mutation":"theme-owned"}`,
		`{"title":"large sequence","state":"published","sequence":9007199254740994}`,
	} {
		if _, err := contract.Bind(themeDigest, json.RawMessage(invalid), PageSEOView{}); !errors.Is(err, ErrViewModelSchema) {
			t.Fatalf("invalid payload %s error = %v", invalid, err)
		}
	}
}

func TestPluginPageViewModelContractRejectsSchemaDriftAndExternalReferences(t *testing.T) {
	schema := []byte(`{"type":"object"}`)
	if _, err := CompilePluginPageViewModelContract(
		"plugin.demo.template.article", "plugin.demo.page.article.data@1",
		strings.Repeat("a", 64), schema,
	); !errors.Is(err, ErrPluginViewModelSchema) {
		t.Fatalf("digest drift error = %v", err)
	}
	external := []byte(`{"$ref":"https://example.test/schema.json"}`)
	digest := sha256.Sum256(external)
	if _, err := CompilePluginPageViewModelContract(
		"plugin.demo.template.article", "plugin.demo.page.article.data@1",
		hex.EncodeToString(digest[:]), external,
	); !errors.Is(err, ErrPluginViewModelSchema) {
		t.Fatalf("external reference error = %v", err)
	}
}

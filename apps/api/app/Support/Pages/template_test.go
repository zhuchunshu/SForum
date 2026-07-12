package pages

import "testing"

func TestRenderTemplate(t *testing.T) {
	out, err := RenderTemplate(`<h1>{{title}}</h1><sf-topic-list></sf-topic-list>`, map[string]string{"title": `<b>Hi</b>`})
	if err != nil {
		t.Fatal(err)
	}
	if out != `<h1>&lt;b&gt;Hi&lt;/b&gt;</h1><sf-topic-list></sf-topic-list>` {
		t.Fatalf("unexpected render: %s", out)
	}
}

func TestValidateTemplateRejectsScript(t *testing.T) {
	if err := ValidateTemplate(`<script>alert(1)</script>`); err == nil {
		t.Fatal("expected reject")
	}
}

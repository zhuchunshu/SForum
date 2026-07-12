package pages

import "testing"

func TestValidateCSS(t *testing.T) {
	if err := ValidateCSS(`:root { --sf-primary: #0ea5e9; } .x { color: red; }`); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		`body { width: expression(alert(1)); }`,
		`a { background: url(javascript:alert(1)); }`,
		`@import url("https://evil.example/x.css");`,
		`div { behavior: url(x.htc); }`,
		`x { -moz-binding: url(x.xml); }`,
		`/* <script>alert(1)</script> */`,
	} {
		if err := ValidateCSS(bad); err == nil {
			t.Fatalf("expected reject for %q", bad)
		}
	}
}

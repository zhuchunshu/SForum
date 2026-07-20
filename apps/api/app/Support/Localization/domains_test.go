package localization

import (
	"errors"
	"testing"
)

func TestDomainFallbackOverrideAndPlural(t *testing.T) {
	cat := NewCatalog()
	cat.SeedCore()
	if err := cat.Register("demo.plugin", "en-US", "hello", "Hello", true); err != nil {
		t.Fatal(err)
	}
	if err := cat.Register("demo.plugin", "zh-CN", "hello", "你好", true); err != nil {
		t.Fatal(err)
	}
	if got := cat.T("demo.plugin", "en-US", "hello"); got != "Hello" {
		t.Fatalf("en = %q", got)
	}
	// Missing en key falls back to zh-CN default locale for domain.
	if err := cat.Register("demo.plugin", "zh-CN", "only.zh", "仅中文", true); err != nil {
		t.Fatal(err)
	}
	if got := cat.T("demo.plugin", "en-US", "only.zh"); got != "仅中文" {
		t.Fatalf("fallback = %q", got)
	}
	if err := cat.SetOverride("demo.plugin", "en-US", "hello", "Hi"); err != nil {
		t.Fatal(err)
	}
	if got := cat.T("demo.plugin", "en-US", "hello"); got != "Hi" {
		t.Fatalf("override = %q", got)
	}

	_ = cat.Register("demo.plugin", "en-US", "items.one", "{count} item", true)
	_ = cat.Register("demo.plugin", "en-US", "items.other", "{count} items", true)
	if got := cat.TP("demo.plugin", "en-US", "items", 1); got != "1 item" {
		t.Fatalf("plural one = %q", got)
	}
	if got := cat.TP("demo.plugin", "en-US", "items", 3); got != "3 items" {
		t.Fatalf("plural other = %q", got)
	}
}

func TestCatalogCollisionStrict(t *testing.T) {
	cat := NewCatalog()
	if err := cat.Register("demo", "zh-CN", "k", "a", true); err != nil {
		t.Fatal(err)
	}
	if err := cat.Register("demo", "zh-CN", "k", "b", true); !errors.Is(err, ErrCatalogCollision) {
		t.Fatalf("collision = %v", err)
	}
	// Non-strict overwrite allowed.
	if err := cat.Register("demo", "zh-CN", "k", "b", false); err != nil {
		t.Fatal(err)
	}
	if got := cat.T("demo", "zh-CN", "k"); got != "b" {
		t.Fatalf("overwrite = %q", got)
	}
}

func TestLanguagePackEnableDisable(t *testing.T) {
	cat := NewCatalog()
	reg := NewPackRegistry(cat)
	pack := LanguagePack{
		ID: "pack.demo", Domain: "demo.i18n", Version: "1.0.0",
		Locales: []string{"en-US", "zh-CN"},
		Messages: map[string]map[string]string{
			"en-US": {"greet": "Hello pack"},
			"zh-CN": {"greet": "你好包"},
		},
	}
	if err := reg.Install(pack); err != nil {
		t.Fatal(err)
	}
	if err := reg.Enable("pack.demo"); err != nil {
		t.Fatal(err)
	}
	if got := cat.T("demo.i18n", "en-US", "greet"); got != "Hello pack" {
		t.Fatalf("enabled = %q", got)
	}
	// Collision with second pack same domain+key fails.
	other := LanguagePack{
		ID: "pack.other", Domain: "demo.i18n", Version: "1.0.0",
		Locales: []string{"en-US"},
		Messages: map[string]map[string]string{
			"en-US": {"greet": "Conflict"},
		},
	}
	if err := reg.Install(other); err != nil {
		t.Fatal(err)
	}
	if err := reg.Enable("pack.other"); !errors.Is(err, ErrCatalogCollision) {
		t.Fatalf("pack collision = %v", err)
	}
	if err := reg.Disable("pack.demo"); err != nil {
		t.Fatal(err)
	}
	list := reg.List()
	if len(list) != 2 {
		t.Fatalf("list = %#v", list)
	}
}

func TestPluralForm(t *testing.T) {
	if PluralForm("en-US", 1) != "one" || PluralForm("en-US", 2) != "other" {
		t.Fatal("en plural")
	}
	if PluralForm("zh-CN", 0) != "zero" || PluralForm("zh-CN", 5) != "other" {
		t.Fatal("zh plural")
	}
}

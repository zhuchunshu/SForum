package capabilities

import (
	"errors"
	"testing"
)

func TestCatalogKeysAreStableAndKnown(t *testing.T) {
	items := Catalog()
	if len(items) < 5 {
		t.Fatalf("expected catalog entries, got %d", len(items))
	}
	for _, item := range items {
		if !Known(item.Key) {
			t.Fatalf("catalog key not known: %s", item.Key)
		}
		if item.Risk != RiskLow && item.Risk != RiskMedium && item.Risk != RiskHigh {
			t.Fatalf("bad risk for %s: %s", item.Key, item.Risk)
		}
	}
}

func TestValidateKeys(t *testing.T) {
	if err := ValidateKeys([]string{NetOutbound, JobsEnqueue}); err != nil {
		t.Fatalf("expected valid keys: %v", err)
	}
	if err := ValidateKeys([]string{"evil.root"}); !errors.Is(err, ErrUnknown) {
		t.Fatalf("expected unknown, got %v", err)
	}
	if err := ValidateKeys([]string{NetOutbound, NetOutbound}); err == nil {
		t.Fatal("expected duplicate to fail")
	}
}

func TestSetRequire(t *testing.T) {
	set := NewSet([]string{NetOutbound, SettingsOwn})
	if err := set.Require(NetOutbound); err != nil {
		t.Fatalf("expected allow: %v", err)
	}
	if err := set.Require(UsersRead); !errors.Is(err, ErrDenied) {
		t.Fatalf("expected denied, got %v", err)
	}
}

func TestResolveImpliesMailAndJobs(t *testing.T) {
	keys, implied := Resolve(ResolveInput{
		Explicit:      nil,
		HasJobs:       true,
		HasSettings:   true,
		ProviderSlots: []string{"mail.provider"},
		HasBackend:    true,
	})
	set := NewSet(keys)
	for _, want := range []string{HostAPI, SettingsOwn, JobsEnqueue, NetOutbound} {
		if !set.Has(want) {
			t.Fatalf("missing implied %s in %#v", want, keys)
		}
		if !implied[want] {
			t.Fatalf("expected %s marked implied", want)
		}
	}
}

func TestResolveExplicitNotMarkedImplied(t *testing.T) {
	keys, implied := Resolve(ResolveInput{
		Explicit:   []string{NetOutbound},
		HasBackend: true,
	})
	set := NewSet(keys)
	if !set.Has(NetOutbound) || implied[NetOutbound] {
		t.Fatalf("explicit net.outbound should not be implied: keys=%v implied=%v", keys, implied)
	}
	if !set.Has(HostAPI) || !implied[HostAPI] {
		t.Fatalf("host.api should be implied for backend: keys=%v implied=%v", keys, implied)
	}
}

func TestGrantsForSMTPLikeInput(t *testing.T) {
	grants := GrantsFor(ResolveInput{
		HasBackend:    true,
		HasSettings:   true,
		ProviderSlots: []string{"mail.provider"},
	})
	if len(grants) == 0 {
		t.Fatal("expected grants")
	}
	// 高风险应排在前面。
	if grants[0].Risk != RiskHigh {
		t.Fatalf("expected high risk first, got %#v", grants[0])
	}
}

func TestRequiresConfirmation(t *testing.T) {
	if RequiresConfirmation(nil) {
		t.Fatal("empty should not require confirmation")
	}
	if !RequiresConfirmation([]string{SettingsOwn}) {
		t.Fatal("any capability should require confirmation")
	}
}

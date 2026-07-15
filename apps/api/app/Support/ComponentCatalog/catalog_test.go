package componentcatalog

import (
	"strings"
	"testing"
)

func TestCoreComponentCatalogIsValidAndDetached(t *testing.T) {
	first := CoreComponentCatalog()
	second := CoreComponentCatalog()
	if len(first) == 0 || len(first) != len(second) {
		t.Fatalf("catalog lengths = %d, %d", len(first), len(second))
	}
	if err := validateCoreComponentCatalog(first); err != nil {
		t.Fatal(err)
	}

	originalID := second[0].ID
	originalOwner := second[0].Owners[0]
	first[0].ID = "mutated"
	first[0].Owners[0] = Owner("mutated")
	if current := CoreComponentCatalog()[0]; current.ID != originalID || current.Owners[0] != originalOwner {
		t.Fatalf("caller mutation escaped into generated catalog: %#v", current)
	}
}

func TestFindCoreComponentReturnsReviewedPublicAndAdminTargets(t *testing.T) {
	tests := []struct {
		id    string
		owner Owner
	}{
		{id: "core.component.page.forum.home", owner: OwnerPublic},
		{id: "core.component.page.admin", owner: OwnerAdmin},
		{id: "core.component.shared.sfbutton", owner: OwnerPublic},
		{id: "core.component.shared.sfbutton", owner: OwnerAdmin},
	}
	for _, test := range tests {
		item, found := FindCoreComponent(test.id)
		if !found || !item.OwnedBy(test.owner) {
			t.Fatalf("FindCoreComponent(%q) = %#v, %v", test.id, item, found)
		}
		item.Owners[0] = Owner("mutated")
		again, _ := FindCoreComponent(test.id)
		if again.Owners[0] == Owner("mutated") {
			t.Fatal("lookup returned shared owner storage")
		}
	}
	if _, found := FindCoreComponent("core.component.page.missing"); found {
		t.Fatal("unknown core component resolved")
	}
}

func TestCoreComponentCatalogRejectsCollisionsAndOwnershipDrift(t *testing.T) {
	base := CoreComponentCatalog()
	tests := []struct {
		name string
		want string
		edit func([]CoreComponent) []CoreComponent
	}{
		{name: "id collision", want: "duplicate core id", edit: func(items []CoreComponent) []CoreComponent {
			items[1].ID = items[0].ID
			return items
		}},
		{name: "contract collision", want: "duplicate contract", edit: func(items []CoreComponent) []CoreComponent {
			items[1].ContractVersion = items[0].ContractVersion
			return items
		}},
		{name: "source collision", want: "duplicate source", edit: func(items []CoreComponent) []CoreComponent {
			items[1].Source = items[0].Source
			return items
		}},
		{name: "missing ownership", want: "explicit ownership", edit: func(items []CoreComponent) []CoreComponent {
			items[0].Owners = nil
			return items
		}},
		{name: "duplicate ownership", want: "duplicate or non-canonical", edit: func(items []CoreComponent) []CoreComponent {
			items[0].Owners = []Owner{OwnerAdmin, OwnerAdmin}
			return items
		}},
		{name: "invalid ownership", want: "invalid owner", edit: func(items []CoreComponent) []CoreComponent {
			items[0].Owners = []Owner{"staff"}
			return items
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := test.edit(CoreComponentCatalog())
			err := validateCoreComponentCatalog(items)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validation error = %v, want %q", err, test.want)
			}
		})
	}
	if err := validateCoreComponentCatalog(base); err != nil {
		t.Fatalf("test edits mutated the source catalog: %v", err)
	}
}

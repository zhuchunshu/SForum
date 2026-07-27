package extensiondatabase

import "testing"

func TestResolveIdentifiersProducesDistinctSafeNames(t *testing.T) {
	identifiers, err := ResolveIdentifiers("acme.storage")
	if err != nil {
		t.Fatal(err)
	}
	if !identifiers.Valid() || identifiers.Schema == identifiers.OwnerRole || identifiers.Schema == identifiers.RuntimeRole {
		t.Fatalf("identifiers = %#v", identifiers)
	}
}

func TestResolveIdentifiersRejectsInvalidExtensionID(t *testing.T) {
	if _, err := ResolveIdentifiers(" ../core "); err == nil {
		t.Fatal("expected invalid extension identity to fail closed")
	}
}

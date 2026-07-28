package sitechrome

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNavigationV1ContractIsFrozen(t *testing.T) {
	if NavigationBackupSchemaID != "sforum.site-navigation-backup@1" {
		t.Fatalf("backup schema = %q", NavigationBackupSchemaID)
	}
	locations := NavigationLocations()
	want := []string{
		NavigationLocationTopbar,
		NavigationLocationSidebar,
		NavigationLocationMobile,
		NavigationLocationFooter,
	}
	if len(locations) != len(want) {
		t.Fatalf("locations = %#v", locations)
	}
	for index, location := range want {
		if locations[index] != location {
			t.Fatalf("location[%d] = %q, want %q", index, locations[index], location)
		}
	}
	if NavigationMaxDefinitions <= 0 || NavigationMaxPlacements < NavigationMaxDefinitions || NavigationMaxSnapshots != 20 || NavigationMaxDynamicItems != 100 {
		t.Fatalf("unexpected navigation bounds: definitions=%d placements=%d snapshots=%d dynamicItems=%d", NavigationMaxDefinitions, NavigationMaxPlacements, NavigationMaxSnapshots, NavigationMaxDynamicItems)
	}
	if NavigationPreviewChangeLocation != "location" || NavigationPreviewChangeDefinitions != "definitions" || NavigationPreviewWarningExtensionReferenceInert != "extension_reference_inert" {
		t.Fatalf("unexpected navigation preview vocabulary: %q %q %q", NavigationPreviewChangeLocation, NavigationPreviewChangeDefinitions, NavigationPreviewWarningExtensionReferenceInert)
	}
	defaults := NavigationRecommendedPlacements()
	if len(defaults) != 10 {
		t.Fatalf("recommended placement count = %d, want 10", len(defaults))
	}
	wantByLocation := map[string]int{
		NavigationLocationTopbar:  3,
		NavigationLocationSidebar: 4,
		NavigationLocationMobile:  3,
		NavigationLocationFooter:  0,
	}
	for _, location := range locations {
		count := 0
		for _, placement := range defaults {
			if placement.Location == location {
				count++
			}
		}
		if count != wantByLocation[location] {
			t.Fatalf("recommended defaults for %q = %d, want %d", location, count, wantByLocation[location])
		}
	}
}

func TestNavigationDynamicItemLimitIsBoundedAndSourceSpecific(t *testing.T) {
	valid := NavigationDocument{Placements: []NavigationPlacement{{
		SourceKey: "core.dynamic.categories", Location: NavigationLocationSidebar, Order: 40,
		Enabled: true, Visibility: NavigationVisibilityPublic, MaxItems: 12,
	}}}
	if normalized, err := normalizeNavigationDocument(valid); err != nil || normalized.Placements[0].MaxItems != 12 {
		t.Fatalf("valid dynamic item limit normalized=%#v err=%v", normalized, err)
	}
	restored := navigationDefaultsDocument(valid, []string{NavigationLocationSidebar})
	foundDynamic := false
	for _, placement := range restored.Placements {
		if placement.SourceKey == "core.dynamic.categories" {
			foundDynamic = true
			if placement.MaxItems != 0 {
				t.Fatalf("recommended dynamic item limit = %d, want unlimited", placement.MaxItems)
			}
		}
	}
	if !foundDynamic {
		t.Fatal("recommended sidebar defaults omitted dynamic categories")
	}
	for _, invalid := range []NavigationPlacement{
		{SourceKey: "core.dynamic.categories", Location: NavigationLocationSidebar, Order: 40, Enabled: true, Visibility: NavigationVisibilityPublic, MaxItems: NavigationMaxDynamicItems + 1},
		{SourceKey: "core.home", Location: NavigationLocationSidebar, Order: 10, Enabled: true, Visibility: NavigationVisibilityPublic, MaxItems: 5},
	} {
		if _, err := normalizeNavigationDocument(NavigationDocument{Placements: []NavigationPlacement{invalid}}); err != ErrInvalid {
			t.Fatalf("invalid max-items placement accepted: %#v err=%v", invalid, err)
		}
	}
}

func TestNavigationSnapshotActorJSONIsBackwardCompatible(t *testing.T) {
	legacy, err := json.Marshal(NavigationSnapshot{ID: 1, Revision: 1})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(legacy), "actorUserId") {
		t.Fatalf("legacy snapshot exposed zero actor: %s", legacy)
	}
	attributed, err := json.Marshal(NavigationSnapshot{ID: 2, Revision: 2, ActorUserID: 77})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(attributed), `"actorUserId":77`) {
		t.Fatalf("attributed snapshot omitted actor: %s", attributed)
	}
}

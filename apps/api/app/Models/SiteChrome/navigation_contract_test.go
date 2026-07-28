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
	if NavigationMaxDefinitions <= 0 || NavigationMaxPlacements < NavigationMaxDefinitions || NavigationMaxSnapshots != 20 {
		t.Fatalf("unexpected navigation bounds: definitions=%d placements=%d snapshots=%d", NavigationMaxDefinitions, NavigationMaxPlacements, NavigationMaxSnapshots)
	}
	if NavigationPreviewChangeLocation != "location" || NavigationPreviewChangeDefinitions != "definitions" || NavigationPreviewWarningExtensionReferenceInert != "extension_reference_inert" {
		t.Fatalf("unexpected navigation preview vocabulary: %q %q %q", NavigationPreviewChangeLocation, NavigationPreviewChangeDefinitions, NavigationPreviewWarningExtensionReferenceInert)
	}
	defaults := NavigationRecommendedPlacements()
	if len(defaults) != 14 {
		t.Fatalf("recommended placement count = %d, want 14", len(defaults))
	}
	for _, location := range locations {
		found := false
		for _, placement := range defaults {
			if placement.Location == location {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("recommended defaults missing %q", location)
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

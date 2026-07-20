package privacy

import (
	"context"
	"errors"
	"testing"
)

func TestExportEraseAndExternalWarning(t *testing.T) {
	reg := New()
	if err := reg.Register(Contribution{
		ExtensionID: "demo.mail",
		Inventory: []InventoryItem{
			{ID: "demo.mail.messages", Kind: KindPersonalData, Description: "outbound mail copies"},
			{ID: "demo.mail.cdn", Kind: KindExternal, Description: "CDN avatar cache", External: true},
		},
	}, func(ctx context.Context, userID string) (ExportArtifact, error) {
		return ExportArtifact{MediaType: "application/json", Body: []byte(`{"user":"` + userID + `"}`)}, nil
	}, func(ctx context.Context, userID string) (EraseResult, error) {
		return EraseResult{
			Erased: true,
			RetainedExternal: []string{"cdn://avatars/" + userID},
			Warnings:         []string{"CDN purge is eventual"},
		}, nil
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := reg.ExportUser(context.Background(), "", "42"); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("export actor = %v", err)
	}
	bundle, err := reg.ExportUser(context.Background(), "admin", "42")
	if err != nil || len(bundle.Artifacts) != 1 || string(bundle.Artifacts[0].Body) != `{"user":"42"}` {
		t.Fatalf("export = %#v err=%v", bundle, err)
	}
	if len(bundle.Inventory) != 2 {
		t.Fatalf("inventory = %#v", bundle.Inventory)
	}

	report, err := reg.EraseUser(context.Background(), "admin", "42")
	if err != nil || len(report.Results) != 1 || !report.Results[0].Erased {
		t.Fatalf("erase = %#v err=%v", report, err)
	}
	// Retained external from hook + inventory declaration.
	if len(report.RetainedExternal) < 2 {
		t.Fatalf("retained external = %#v", report.RetainedExternal)
	}
	foundCDN := false
	for _, item := range report.RetainedExternal {
		if item == "cdn://avatars/42" || item == "demo.mail:demo.mail.cdn" {
			foundCDN = true
		}
	}
	if !foundCDN {
		t.Fatalf("expected CDN retention warning: %#v", report.RetainedExternal)
	}
}

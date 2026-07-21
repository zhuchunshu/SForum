package privacy

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExportErasePartialAndExternal(t *testing.T) {
	reg := New()
	auditor := &MemoryAuditor{}
	reg.SetAuditor(auditor)
	reg.SetPermissionCheck(func(_ context.Context, actor, userID, op string) error {
		if actor != "admin" {
			return ErrPermissionDenied
		}
		return nil
	})
	if err := reg.Register(Contribution{
		ExtensionID: "demo.a",
		Inventory: []InventoryItem{
			{ID: "a.profile", Kind: KindPersonalData, Description: "profile"},
			{ID: "a.cdn", Kind: KindExternal, Description: "cdn", External: true},
		},
	}, func(ctx context.Context, userID string) (ExportArtifact, error) {
		return ExportArtifact{MediaType: "application/json", Body: []byte(`{"ok":1}`)}, nil
	}, func(ctx context.Context, userID string) (EraseResult, error) {
		return EraseResult{Erased: true, RetainedExternal: []string{"cdn://demo"}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(Contribution{
		ExtensionID: "demo.b",
		Inventory:   []InventoryItem{{ID: "b.log", Kind: KindLog, Description: "logs"}},
	}, func(ctx context.Context, userID string) (ExportArtifact, error) {
		return ExportArtifact{}, errors.New("export down")
	}, func(ctx context.Context, userID string) (EraseResult, error) {
		return EraseResult{}, errors.New("erase down")
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := reg.ExportUser(context.Background(), "", "u1"); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("empty actor = %v", err)
	}
	if _, err := reg.ExportUser(context.Background(), "guest", "u1"); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("guest = %v", err)
	}

	bundle, err := reg.ExportUser(context.Background(), "admin", "u1")
	if !errors.Is(err, ErrPartial) || !bundle.Partial || len(bundle.Artifacts) != 1 {
		t.Fatalf("export partial = %#v err=%v", bundle, err)
	}
	report, err := reg.EraseUser(context.Background(), "admin", "u1")
	if !errors.Is(err, ErrPartial) || !report.Partial {
		t.Fatalf("erase partial = %#v err=%v", report, err)
	}
	if len(report.RetainedExternal) == 0 {
		t.Fatal("expected retained external")
	}
	if len(auditor.Events()) < 2 {
		t.Fatalf("audit = %#v", auditor.Events())
	}
}

func TestPublishContributionWithoutGoCallback(t *testing.T) {
	reg := New()
	if err := reg.PublishContribution(Contribution{
		ExtensionID:   "demo.pub",
		PackageDigest: stringsRepeat("ab", 32),
		Inventory:     []InventoryItem{{ID: "pub.data", Kind: KindPersonalData, Description: "published"}},
		SupportsExport: true,
	}); err != nil {
		t.Fatal(err)
	}
	inv := reg.Inventory()
	if len(inv) != 1 || inv[0].ID != "pub.data" {
		t.Fatalf("inventory = %#v", inv)
	}
}

func TestExportDeadline(t *testing.T) {
	reg := New()
	_ = reg.Register(Contribution{
		ExtensionID: "demo.slow",
		Inventory:   []InventoryItem{{ID: "s", Kind: KindPersonalData, Description: "s"}},
	}, func(ctx context.Context, userID string) (ExportArtifact, error) {
		select {
		case <-ctx.Done():
			return ExportArtifact{}, ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return ExportArtifact{Body: []byte(`{}`)}, nil
		}
	}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, err := reg.ExportUser(ctx, "admin", "u1")
	if err == nil {
		t.Fatal("expected deadline error")
	}
}

func stringsRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}

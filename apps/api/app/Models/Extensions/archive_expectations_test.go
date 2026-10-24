package extensions

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestInstallRejectsMismatchedArchiveExpectations(t *testing.T) {
	cases := []struct {
		name       string
		input      ArchiveInput
		wantFields []string
	}{
		{
			name:       "extension id",
			input:      ArchiveInput{ExpectedExtensionID: "other.plugin"},
			wantFields: []string{"extensionId"},
		},
		{
			name:       "type",
			input:      ArchiveInput{ExpectedExtensionID: "expect.plugin", ExpectedType: TypeTheme},
			wantFields: []string{"type"},
		},
		{
			name:       "version",
			input:      ArchiveInput{ExpectedExtensionID: "expect.plugin", ExpectedType: TypePlugin, ExpectedVersion: "2.0.0"},
			wantFields: []string{"version"},
		},
		{
			name: "every field at once",
			input: ArchiveInput{
				ExpectedExtensionID: "other.theme",
				ExpectedVersion:     "9.9.9",
				ExpectedType:        TypeTheme,
			},
			wantFields: []string{"extensionId", "type", "version"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			store := &fakeExtensionStore{}
			service := newLifecycleService(store, t.TempDir(), &fakeRuntimeManager{})
			input := testCase.input
			input.FileName = "package.zip"
			input.Data = extensionArchive(t, validManifest("expect.plugin", TypePlugin))

			_, err := service.InstallOrUpgradeArchive(context.Background(), extensionManager(), input)
			if !errors.Is(err, ErrArchiveExpectationMismatch) {
				t.Fatalf("expected ErrArchiveExpectationMismatch, got %v", err)
			}
			var mismatch *ArchiveExpectationMismatchError
			if !errors.As(err, &mismatch) {
				t.Fatalf("expected typed mismatch error, got %T", err)
			}
			if strings.Join(mismatch.Fields, ",") != strings.Join(testCase.wantFields, ",") {
				t.Fatalf("mismatched fields=%v, want %v", mismatch.Fields, testCase.wantFields)
			}
			if mismatch.ActualExtensionID != "expect.plugin" || mismatch.ActualType != TypePlugin ||
				mismatch.ActualVersion != "1.0.0" {
				t.Fatalf("actual manifest identity not reported: %#v", mismatch)
			}
			// 不匹配必须中止安装本身，而不是仅警告后继续暂存。
			if store.saved.Manifest.ID != "" {
				t.Fatalf("mismatch must not persist a package: %#v", store.saved)
			}
			if len(store.events) != 0 {
				t.Fatalf("mismatch must not emit lifecycle events: %#v", store.events)
			}
			if store.enabledID != "" {
				t.Fatalf("mismatch must not enable anything: %q", store.enabledID)
			}
		})
	}
}

func TestInstallAcceptsMatchingArchiveExpectations(t *testing.T) {
	store := &fakeExtensionStore{}
	service := newLifecycleService(store, t.TempDir(), &fakeRuntimeManager{})

	result, err := service.InstallOrUpgradeArchive(context.Background(), extensionManager(), ArchiveInput{
		FileName:            "package.zip",
		Data:                extensionArchive(t, validManifest("expect.plugin", TypePlugin)),
		ExpectedExtensionID: "expect.plugin",
		ExpectedVersion:     "1.0.0",
		ExpectedType:        TypePlugin,
	})
	if err != nil {
		t.Fatalf("matching expectations should install: %v", err)
	}
	if result.Extension.ID != "expect.plugin" {
		t.Fatalf("unexpected extension: %#v", result.Extension)
	}
	// 只暂存：期望校验通过也不会触发启用。
	if store.enabledID != "" {
		t.Fatalf("install must not enable the package: %q", store.enabledID)
	}
}

func TestArchiveSourceLabelIsRecordedInAudit(t *testing.T) {
	store := &fakeExtensionStore{}
	auditor := &recordingAuditor{}
	service := newLifecycleService(store, t.TempDir(), &fakeRuntimeManager{}, WithAuditor(auditor))

	if _, err := service.InstallOrUpgradeArchive(context.Background(), extensionManager(), ArchiveInput{
		FileName:    "package.zip",
		Data:        extensionArchive(t, validManifest("audit.plugin", TypePlugin)),
		SourceLabel: "admin-page:sforum.store",
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(auditor.events) != 1 {
		t.Fatalf("expected one audit event, got %#v", auditor.events)
	}
	if got := auditor.events[0].Metadata["sourceLabel"]; got != "admin-page:sforum.store" {
		t.Fatalf("sourceLabel=%v", got)
	}
	if _, exists := auditor.events[0].Metadata["sourceUrl"]; exists {
		t.Fatalf("empty source URL must not be recorded: %#v", auditor.events[0].Metadata)
	}
}

func TestSanitizeSourceLabel(t *testing.T) {
	cases := map[string]string{
		"admin-page:sforum.store":     "admin-page:sforum.store",
		"  Admin Page: sforum.store ": "adminpage:sforum.store",
		"admin-page:/etc/passwd":      "admin-page:etcpasswd",
		"line\nbreak":                 "linebreak",
		"":                            "",
		"::":                          "",
		strings.Repeat("a", 80):       strings.Repeat("a", maxSourceLabelLength),
	}
	for raw, want := range cases {
		if got := sanitizeSourceLabel(raw); got != want {
			t.Fatalf("sanitizeSourceLabel(%q)=%q, want %q", raw, got, want)
		}
	}
	if ValidSourceLabel("admin page:sforum.store") {
		t.Fatal("a label with spaces is not canonical and must be rejected")
	}
	if !ValidSourceLabel("admin-page:sforum.store") {
		t.Fatal("canonical label should be accepted")
	}
	if ValidSourceLabel(strings.Repeat("a", 80)) {
		t.Fatal("an over-long label must be rejected rather than truncated silently")
	}
}

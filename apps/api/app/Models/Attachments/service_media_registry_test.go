package attachments

import (
	"context"
	"errors"
	"strings"
	"testing"

	mediaregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/MediaRegistry"
	storage "github.com/zhuchunshu/sforum/apps/api/app/Support/Storage"
)

func TestServiceUploadHonorsMediaRegistryMIMEPolicy(t *testing.T) {
	service := NewServiceWithAdapterFactory(&fakeAttachmentStore{}, newAttachmentOptions(nil), func(storage.Config) (storage.Adapter, error) {
		return &fakeStorageAdapter{}, nil
	})
	registry := mediaregistry.New()
	packageDigest := strings.Repeat("ab", 32)
	impactDigest := strings.Repeat("cd", 32)
	if _, err := registry.Publish(mediaregistry.Publication{
		Artifact: mediaregistry.Artifact{
			ExtensionID: "demo.media", ExtensionVersion: "1.0.0",
			PackageDigest: packageDigest, ImpactDigest: impactDigest,
			VersionID: 1, RuntimeInstanceID: "demo.media-runtime",
		},
		Policies: []mediaregistry.MIMEPolicyDeclaration{{
			ID: "demo.media.policy", ContractVersion: "demo.media.policy@1",
			Purpose: "general", Priority: 10, RequiredPermission: "attachment.upload",
			AllowedMIMEs: []string{"image/png"}, AllowedExtensions: []string{"png"},
			StrictDeclaredMIME: true, Budget: mediaregistry.DefaultBudget(),
		}},
	}); err != nil {
		t.Fatal(err)
	}
	service.WithMediaRegistry(registry)

	// Host settings typically allow jpeg; media policy only allows png.
	jpegBody := string([]byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01}) + strings.Repeat("x", 64)
	_, err := service.Upload(context.Background(), uploadActor(), UploadInput{
		OriginalName: "photo.jpg",
		ContentType:  "image/jpeg",
		SizeBytes:    int64(len(jpegBody)),
		File:         newReadSeekCloser(jpegBody),
	})
	if !errors.Is(err, ErrInvalidAttachment) {
		t.Fatalf("expected media policy rejection, got %v", err)
	}
}

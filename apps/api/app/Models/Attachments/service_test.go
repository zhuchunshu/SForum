package attachments

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	storage "github.com/zhuchunshu/sforum/apps/api/app/Support/Storage"
)

func TestServiceUploadStoresObjectAndMetadata(t *testing.T) {
	store := &fakeAttachmentStore{}
	adapter := &fakeStorageAdapter{publicBaseURL: "https://cdn.example.com"}
	service := NewServiceWithAdapterFactory(store, newAttachmentOptions(nil), func(config storage.Config) (storage.Adapter, error) {
		if config.Provider != storage.ProviderLocal {
			t.Fatalf("expected local provider, got %q", config.Provider)
		}
		return adapter, nil
	})

	item, err := service.Upload(context.Background(), uploadActor(), UploadInput{
		OriginalName: " note.txt ",
		ContentType:  "text/plain",
		SizeBytes:    int64(len("hello")),
		File:         newReadSeekCloser("hello"),
	})
	if err != nil {
		t.Fatalf("upload returned error: %v", err)
	}

	if len(store.creates) != 1 {
		t.Fatalf("expected one metadata create, got %d", len(store.creates))
	}
	created := store.creates[0]
	if created.OwnerUserID != 42 || created.Provider != storage.ProviderLocal {
		t.Fatalf("unexpected create metadata: %#v", created)
	}
	if created.OriginalName != "note.txt" || created.ContentType != "text/plain" || created.Extension != ".txt" {
		t.Fatalf("unexpected upload metadata: %#v", created)
	}
	if created.SHA256 != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("unexpected sha256: %s", created.SHA256)
	}
	if !strings.HasSuffix(created.ObjectKey, ".txt") || strings.Contains(created.ObjectKey, "..") {
		t.Fatalf("unsafe object key generated: %q", created.ObjectKey)
	}
	if adapter.putKey != created.ObjectKey || adapter.putBody != "hello" {
		t.Fatalf("object was not written through adapter: key=%q body=%q", adapter.putKey, adapter.putBody)
	}
	if item.URL != "https://cdn.example.com/"+created.ObjectKey {
		t.Fatalf("expected decorated public URL, got %q", item.URL)
	}
}

func TestServiceUploadEmitsAttachmentUploadedEvent(t *testing.T) {
	store := &fakeAttachmentStore{}
	adapter := &fakeStorageAdapter{}
	publisher := &fakeAttachmentEventPublisher{}
	service := NewServiceWithAdapterFactory(store, newAttachmentOptions(nil), func(storage.Config) (storage.Adapter, error) {
		return adapter, nil
	})
	service.events = publisher

	_, err := service.Upload(context.Background(), uploadActor(), UploadInput{
		OriginalName: "note.txt",
		ContentType:  "text/plain",
		SizeBytes:    int64(len("hello")),
		File:         newReadSeekCloser("hello"),
	})
	if err != nil {
		t.Fatalf("upload returned error: %v", err)
	}
	if len(publisher.names) != 1 || publisher.names[0] != appevents.AttachmentUploaded {
		t.Fatalf("expected attachment uploaded event, got %#v", publisher.names)
	}
}

type fakeAttachmentEventPublisher struct {
	names []string
}

func (p *fakeAttachmentEventPublisher) Emit(_ context.Context, envelope appevents.Envelope) appevents.Result {
	p.names = append(p.names, envelope.Name)
	return appevents.Result{OK: true}
}

func TestServiceUploadDeletesRemoteObjectWhenMetadataCreateFails(t *testing.T) {
	expected := errors.New("database unavailable")
	store := &fakeAttachmentStore{createErr: expected}
	adapter := &fakeStorageAdapter{}
	service := NewServiceWithAdapterFactory(store, newAttachmentOptions(nil), func(storage.Config) (storage.Adapter, error) {
		return adapter, nil
	})

	_, err := service.Upload(context.Background(), uploadActor(), UploadInput{
		OriginalName: "note.txt",
		ContentType:  "text/plain",
		SizeBytes:    int64(len("hello")),
		File:         newReadSeekCloser("hello"),
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected create error, got %v", err)
	}
	if adapter.deletedKey == "" || adapter.deletedKey != adapter.putKey {
		t.Fatalf("expected failed metadata create to delete remote object, put=%q deleted=%q", adapter.putKey, adapter.deletedKey)
	}
}

func TestServiceUploadRequiresPermissionAndValidFileType(t *testing.T) {
	service := NewServiceWithAdapterFactory(&fakeAttachmentStore{}, newAttachmentOptions(nil), func(storage.Config) (storage.Adapter, error) {
		return &fakeStorageAdapter{}, nil
	})

	_, err := service.Upload(context.Background(), identity.Actor{ID: 7, Status: identity.UserStatusActive}, UploadInput{
		OriginalName: "note.txt",
		ContentType:  "text/plain",
		SizeBytes:    5,
		File:         newReadSeekCloser("hello"),
	})
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}

	_, err = service.Upload(context.Background(), uploadActor(), UploadInput{
		OriginalName: "shell.sh",
		ContentType:  "text/plain",
		SizeBytes:    5,
		File:         newReadSeekCloser("hello"),
	})
	if !errors.Is(err, ErrInvalidAttachment) {
		t.Fatalf("expected invalid extension, got %v", err)
	}
}

func TestServiceUploadAvatarCompressesJPEGAndStoresPublicAttachment(t *testing.T) {
	store := &fakeAttachmentStore{}
	adapter := &fakeStorageAdapter{}
	service := NewServiceWithAdapterFactory(store, newAttachmentOptions(map[string]string{
		options.NameAvatarTargetDimension: "128",
		options.NameAvatarMaxDimension:    "512",
		options.NameAvatarCompressEnabled: "enabled",
	}), func(storage.Config) (storage.Adapter, error) {
		return adapter, nil
	})
	original := testJPEG(320, 240)

	item, err := service.UploadAvatar(context.Background(), uploadActor(), UploadInput{
		OriginalName: "avatar.jpg",
		ContentType:  "image/jpeg",
		SizeBytes:    int64(len(original)),
		File:         newReadSeekCloserBytes(original),
	})
	if err != nil {
		t.Fatalf("UploadAvatar returned error: %v", err)
	}
	if len(store.creates) != 1 {
		t.Fatalf("expected one attachment create, got %d", len(store.creates))
	}
	created := store.creates[0]
	if created.Visibility != VisibilityPublic || created.ContentType != "image/jpeg" || created.Extension != ".jpg" {
		t.Fatalf("unexpected avatar metadata: %#v", created)
	}
	if created.SizeBytes != int64(len(adapter.putBytes)) || item.SizeBytes != int64(len(adapter.putBytes)) {
		t.Fatalf("expected compressed size metadata to match stored bytes, create=%d item=%d body=%d", created.SizeBytes, item.SizeBytes, len(adapter.putBytes))
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(adapter.putBytes))
	if err != nil {
		t.Fatalf("decode compressed avatar: %v", err)
	}
	if format != "jpeg" || config.Width != 128 || config.Height != 128 {
		t.Fatalf("expected 128x128 jpeg avatar, got format=%s size=%dx%d", format, config.Width, config.Height)
	}
}

func TestServiceUploadAvatarRejectsCompressedOutputOverLimit(t *testing.T) {
	service := NewServiceWithAdapterFactory(&fakeAttachmentStore{}, newAttachmentOptions(map[string]string{
		options.NameAvatarMaxSizeKB:         "1",
		options.NameAvatarTargetDimension:   "256",
		options.NameAvatarCompressEnabled:   "enabled",
		options.NameAvatarCompressQuality:   "100",
		options.NameAvatarMaxDimension:      "512",
		options.NameAttachmentPathTemplate:  "{public_id}{ext}",
		options.NameAttachmentUploadEnabled: "enabled",
	}), func(storage.Config) (storage.Adapter, error) {
		return &fakeStorageAdapter{}, nil
	})
	body := testPNG(64, 64)
	if len(body) > 1024 {
		t.Fatalf("test fixture must be under 1KB before compression, got %d bytes", len(body))
	}

	_, err := service.UploadAvatar(context.Background(), uploadActor(), UploadInput{
		OriginalName: "avatar.png",
		ContentType:  "image/png",
		SizeBytes:    int64(len(body)),
		File:         newReadSeekCloserBytes(body),
	})
	if !errors.Is(err, ErrInvalidAttachment) {
		t.Fatalf("expected compressed avatar over limit to be rejected, got %v", err)
	}
}

func TestServiceUploadAvatarRejectsGIFWhenDisabled(t *testing.T) {
	service := NewServiceWithAdapterFactory(&fakeAttachmentStore{}, newAttachmentOptions(nil), func(storage.Config) (storage.Adapter, error) {
		return &fakeStorageAdapter{}, nil
	})
	body := testGIF()

	_, err := service.UploadAvatar(context.Background(), uploadActor(), UploadInput{
		OriginalName: "avatar.gif",
		ContentType:  "image/gif",
		SizeBytes:    int64(len(body)),
		File:         newReadSeekCloserBytes(body),
	})
	if !errors.Is(err, ErrInvalidAttachment) {
		t.Fatalf("expected GIF to be rejected while avatar.allow_gif is disabled, got %v", err)
	}
}

func TestServiceCleanupUsesConfiguredRetentionWindow(t *testing.T) {
	store := &fakeAttachmentStore{
		cleanupItems: []Attachment{{ID: 1, Provider: storage.ProviderLocal, ObjectKey: "old.txt"}},
	}
	adapter := &fakeStorageAdapter{}
	service := NewServiceWithAdapterFactory(store, newAttachmentOptions(map[string]string{
		options.NameAttachmentCleanupOrphanDays: "7",
	}), func(storage.Config) (storage.Adapter, error) {
		return adapter, nil
	})

	before := time.Now().AddDate(0, 0, -7)
	result, err := service.Cleanup(context.Background(), manageActor(), 25)
	after := time.Now().AddDate(0, 0, -7)
	if err != nil {
		t.Fatalf("cleanup returned error: %v", err)
	}
	if result.Deleted != 1 || result.Failed != 0 {
		t.Fatalf("unexpected cleanup result: %#v", result)
	}
	if store.cleanupLimit != 25 {
		t.Fatalf("expected cleanup limit 25, got %d", store.cleanupLimit)
	}
	if store.cleanupCutoff.Before(before.Add(-time.Second)) || store.cleanupCutoff.After(after.Add(time.Second)) {
		t.Fatalf("cleanup cutoff did not use configured retention: %s", store.cleanupCutoff)
	}
	if adapter.deletedKey != "old.txt" {
		t.Fatalf("expected adapter delete old.txt, got %q", adapter.deletedKey)
	}
	if len(store.deletedMetadataIDs) != 1 || store.deletedMetadataIDs[0] != 1 {
		t.Fatalf("expected metadata delete for attachment 1, got %#v", store.deletedMetadataIDs)
	}
}

func TestStorageConfigUsesSettingsLocalRoot(t *testing.T) {
	settings := settingsFromValues(map[string]string{
		options.NameAttachmentLocalRoot:         "storage/custom-attachments",
		options.NameAttachmentLocalPublicPrefix: "/uploads",
	}, nil)

	config := storageConfig(settings)

	if config.LocalRoot != "storage/custom-attachments" {
		t.Fatalf("expected local root from settings, got %q", config.LocalRoot)
	}
	if config.Local.PublicPrefix != "/uploads" {
		t.Fatalf("expected local public prefix from settings, got %q", config.Local.PublicPrefix)
	}
}

func TestServiceUpdateSettingsPersistsLocalRoot(t *testing.T) {
	optionStore := &fakeOptionStore{}
	service := NewServiceWithAdapterFactory(&fakeAttachmentStore{}, options.NewServiceWithCacheTTL(optionStore, time.Minute), func(storage.Config) (storage.Adapter, error) {
		return &fakeStorageAdapter{}, nil
	})
	settings := settingsFromValues(nil, nil)
	settings.Local.Root = "storage/custom-attachments"

	_, err := service.UpdateSettings(context.Background(), attachmentSettingsActor(), settings)
	if err != nil {
		t.Fatalf("UpdateSettings returned error: %v", err)
	}
	if got := optionStore.items[options.NameAttachmentLocalRoot]; got != "storage/custom-attachments" {
		t.Fatalf("expected local root to be persisted, got %q", got)
	}
}

func uploadActor() identity.Actor {
	return identity.Actor{
		ID:          42,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionAttachmentUpload: true},
	}
}

func manageActor() identity.Actor {
	return identity.Actor{
		ID:          7,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionAttachmentManage: true},
	}
}

func attachmentSettingsActor() identity.Actor {
	return identity.Actor{
		ID:          9,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionAttachmentSettings: true},
	}
}

func newAttachmentOptions(values map[string]string) *options.Service {
	store := &fakeOptionStore{items: values}
	return options.NewServiceWithCacheTTL(store, time.Minute)
}

type readSeekCloser struct {
	*bytes.Reader
}

func newReadSeekCloser(value string) *readSeekCloser {
	return &readSeekCloser{Reader: bytes.NewReader([]byte(value))}
}

func newReadSeekCloserBytes(value []byte) *readSeekCloser {
	return &readSeekCloser{Reader: bytes.NewReader(value)}
}

func (r *readSeekCloser) Close() error { return nil }

func testJPEG(width int, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 180, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func testGIF() []byte {
	img := image.NewPaletted(image.Rect(0, 0, 32, 32), []color.Color{color.White, color.Black})
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func testPNG(width int, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

type fakeStorageAdapter struct {
	publicBaseURL string
	putKey        string
	putBody       string
	putBytes      []byte
	deletedKey    string
}

func (a *fakeStorageAdapter) Put(_ context.Context, key string, input storage.PutInput) error {
	body, err := io.ReadAll(input.Reader)
	if err != nil {
		return err
	}
	a.putKey = key
	a.putBody = string(body)
	a.putBytes = body
	return nil
}

func (a *fakeStorageAdapter) Open(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(a.putBody)), nil
}

func (a *fakeStorageAdapter) Delete(_ context.Context, key string) error {
	a.deletedKey = key
	return nil
}

func (a *fakeStorageAdapter) Stat(_ context.Context, key string) (storage.ObjectInfo, error) {
	return storage.ObjectInfo{Key: key, Size: int64(len(a.putBody))}, nil
}

func (a *fakeStorageAdapter) Exists(context.Context, string) (bool, error) { return true, nil }

func (a *fakeStorageAdapter) PublicURL(key string) string {
	if a.publicBaseURL == "" {
		return ""
	}
	return strings.TrimRight(a.publicBaseURL, "/") + "/" + key
}

func (a *fakeStorageAdapter) SignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return a.PublicURL(key), nil
}

func (a *fakeStorageAdapter) Probe(context.Context) error { return nil }

type fakeAttachmentStore struct {
	creates            []CreateAttachmentInput
	createErr          error
	cleanupItems       []Attachment
	cleanupCutoff      time.Time
	cleanupLimit       int
	deletedMetadataIDs []int64
}

func (s *fakeAttachmentStore) Create(_ context.Context, input CreateAttachmentInput) (Attachment, error) {
	s.creates = append(s.creates, input)
	if s.createErr != nil {
		return Attachment{}, s.createErr
	}
	return Attachment{
		ID:             1,
		PublicID:       input.PublicID,
		Owner:          &OwnerSummary{ID: input.OwnerUserID},
		Provider:       input.Provider,
		ObjectKey:      input.ObjectKey,
		OriginalName:   input.OriginalName,
		ContentType:    input.ContentType,
		Extension:      input.Extension,
		SizeBytes:      input.SizeBytes,
		SHA256:         input.SHA256,
		Visibility:     input.Visibility,
		Status:         StatusActive,
		ReferenceCount: 0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

func (s *fakeAttachmentStore) GetByPublicID(context.Context, string) (Attachment, error) {
	return Attachment{}, ErrAttachmentNotFound
}

func (s *fakeAttachmentStore) GetByID(context.Context, int64) (Attachment, error) {
	return Attachment{}, ErrAttachmentNotFound
}

func (s *fakeAttachmentStore) List(context.Context, AttachmentListInput) (AttachmentList, error) {
	return AttachmentList{}, nil
}

func (s *fakeAttachmentStore) ListReferences(context.Context, int64) ([]AttachmentReference, error) {
	return nil, nil
}

func (s *fakeAttachmentStore) UpdateStatus(context.Context, int64, string, bool) (Attachment, error) {
	return Attachment{}, nil
}

func (s *fakeAttachmentStore) ListCleanupCandidates(_ context.Context, cutoff time.Time, limit int) ([]Attachment, error) {
	s.cleanupCutoff = cutoff
	s.cleanupLimit = limit
	return s.cleanupItems, nil
}

func (s *fakeAttachmentStore) DeleteMetadata(_ context.Context, id int64) error {
	s.deletedMetadataIDs = append(s.deletedMetadataIDs, id)
	return nil
}

type fakeOptionStore struct {
	items map[string]string
}

func (s *fakeOptionStore) List(context.Context) ([]options.Option, error) {
	items := make([]options.Option, 0, len(s.items))
	for name, value := range s.items {
		items = append(items, options.Option{Name: name, Value: value})
	}
	return items, nil
}

func (s *fakeOptionStore) InsertMissing(_ context.Context, input options.UpdateInput) error {
	if s.items == nil {
		s.items = map[string]string{}
	}
	if _, ok := s.items[input.Name]; !ok {
		s.items[input.Name] = input.Value
	}
	return nil
}

func (s *fakeOptionStore) Upsert(_ context.Context, input options.UpdateInput) (options.Option, error) {
	if s.items == nil {
		s.items = map[string]string{}
	}
	s.items[input.Name] = input.Value
	return options.Option{Name: input.Name, Value: input.Value}, nil
}

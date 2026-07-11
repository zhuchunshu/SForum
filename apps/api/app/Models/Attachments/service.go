package attachments

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"context"

	"github.com/disintegration/imaging"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	storage "github.com/zhuchunshu/sforum/apps/api/app/Support/Storage"
)

const defaultSignedURLTTL = 5 * time.Minute

type Service struct {
	store          Store
	options        *options.Service
	events         appevents.Publisher
	adapterFactory func(storage.Config) (storage.Adapter, error)
}

func NewService(store Store, optionsService *options.Service) *Service {
	return NewServiceWithEvents(store, optionsService, nil)
}

func NewServiceWithEvents(store Store, optionsService *options.Service, publisher appevents.Publisher) *Service {
	return &Service{
		store:          store,
		options:        optionsService,
		events:         appevents.EnsurePublisher(publisher),
		adapterFactory: storage.NewAdapter,
	}
}

func NewServiceWithAdapterFactory(store Store, optionsService *options.Service, factory func(storage.Config) (storage.Adapter, error)) *Service {
	service := NewServiceWithEvents(store, optionsService, nil)
	if factory != nil {
		service.adapterFactory = factory
	}
	return service
}

func (s *Service) Upload(ctx context.Context, actor identity.Actor, input UploadInput) (Attachment, error) {
	if !actor.Can(identity.PermissionAttachmentUpload) {
		return Attachment{}, identity.ErrPermissionDenied
	}
	settings, err := s.runtimeSettings(ctx)
	if err != nil {
		return Attachment{}, err
	}
	if !settings.UploadEnabled {
		return Attachment{}, ErrUploadDisabled
	}
	if input.File == nil || input.SizeBytes <= 0 {
		return Attachment{}, ErrInvalidAttachment
	}
	maxBytes := int64(settings.MaxFileSizeMB) * 1024 * 1024
	if input.SizeBytes > maxBytes {
		return Attachment{}, ErrInvalidAttachment
	}

	metadata, err := inspectUpload(input, settings)
	if err != nil {
		return Attachment{}, err
	}
	return s.storePreparedUpload(ctx, actor, settings, preparedUpload{
		Reader:     input.File,
		SizeBytes:  input.SizeBytes,
		Metadata:   metadata,
		Visibility: settings.DefaultVisibility,
	})
}

func (s *Service) UploadAvatar(ctx context.Context, actor identity.Actor, input UploadInput) (Attachment, error) {
	if !actor.Can(identity.PermissionAttachmentUpload) {
		return Attachment{}, identity.ErrPermissionDenied
	}
	avatarOptions, err := s.options.AvatarOptions(ctx)
	if err != nil {
		return Attachment{}, err
	}
	if !avatarOptions.AllowUpload {
		return Attachment{}, ErrUploadDisabled
	}
	settings, err := s.runtimeSettings(ctx)
	if err != nil {
		return Attachment{}, err
	}
	prepared, err := prepareAvatarUpload(input, avatarOptions)
	if err != nil {
		return Attachment{}, err
	}
	prepared.Visibility = VisibilityPublic
	return s.storePreparedUpload(ctx, actor, settings, prepared)
}

// UploadSEOImage 复用附件存储链路，但授权由 SEO 专用权限控制。
func (s *Service) UploadSEOImage(ctx context.Context, actor identity.Actor, input UploadInput) (Attachment, error) {
	if !actor.Can(identity.PermissionSEOManage) {
		return Attachment{}, identity.ErrPermissionDenied
	}
	settings, err := s.runtimeSettings(ctx)
	if err != nil {
		return Attachment{}, err
	}
	if !settings.UploadEnabled {
		return Attachment{}, ErrUploadDisabled
	}
	if input.File == nil || input.SizeBytes <= 0 || input.SizeBytes > int64(settings.MaxFileSizeMB)*1024*1024 {
		return Attachment{}, ErrInvalidAttachment
	}
	metadata, err := inspectUpload(input, settings)
	if err != nil || !strings.HasPrefix(metadata.ContentType, "image/") {
		return Attachment{}, ErrInvalidAttachment
	}
	return s.storePreparedUpload(ctx, actor, settings, preparedUpload{
		Reader: input.File, SizeBytes: input.SizeBytes, Metadata: metadata, Visibility: VisibilityPublic,
	})
}

type preparedUpload struct {
	Reader     io.Reader
	SizeBytes  int64
	Metadata   uploadMetadata
	Visibility string
}

func (s *Service) storePreparedUpload(ctx context.Context, actor identity.Actor, settings AttachmentSettings, prepared preparedUpload) (Attachment, error) {
	publicID, err := randomPublicID()
	if err != nil {
		return Attachment{}, err
	}
	objectKey, err := renderObjectKey(settings.PathTemplate, publicID, prepared.Metadata.Extension, time.Now())
	if err != nil {
		return Attachment{}, err
	}
	adapter, err := s.adapterForSettings(settings, settings.Provider)
	if err != nil {
		return Attachment{}, ErrStorageUnavailable
	}

	hash := sha256.New()
	reader := io.TeeReader(prepared.Reader, hash)
	if err := adapter.Put(ctx, objectKey, storage.PutInput{
		Reader:      reader,
		Size:        prepared.SizeBytes,
		ContentType: prepared.Metadata.ContentType,
	}); err != nil {
		return Attachment{}, err
	}

	created, err := s.store.Create(ctx, CreateAttachmentInput{
		PublicID:     publicID,
		OwnerUserID:  actor.ID,
		Provider:     settings.Provider,
		ObjectKey:    objectKey,
		OriginalName: prepared.Metadata.OriginalName,
		ContentType:  prepared.Metadata.ContentType,
		Extension:    prepared.Metadata.Extension,
		SizeBytes:    prepared.SizeBytes,
		SHA256:       hex.EncodeToString(hash.Sum(nil)),
		ImageWidth:   prepared.Metadata.ImageWidth,
		ImageHeight:  prepared.Metadata.ImageHeight,
		Visibility:   prepared.Visibility,
	})
	if err != nil {
		_ = adapter.Delete(ctx, objectKey)
		return Attachment{}, err
	}
	s.events.Emit(ctx, appevents.Envelope{
		Name:          appevents.AttachmentUploaded,
		Kind:          appevents.KindObserve,
		ActorUserID:   actor.ID,
		ResourceType:  "attachment",
		ResourceID:    strconv.FormatInt(created.ID, 10),
		CorrelationID: appevents.NewID(),
		Payload: map[string]any{
			"attachmentId": created.ID,
			"publicId":     created.PublicID,
			"ownerUserId":  actor.ID,
			"provider":     created.Provider,
			"contentType":  created.ContentType,
			"sizeBytes":    created.SizeBytes,
		},
		OccurredAt: time.Now().UTC(),
	})
	return s.decorateURL(ctx, created), nil
}

func (s *Service) Get(ctx context.Context, actor identity.Actor, publicID string) (Attachment, error) {
	attachment, err := s.store.GetByPublicID(ctx, publicID)
	if err != nil {
		return Attachment{}, err
	}
	if !canViewAttachment(actor, attachment) {
		return Attachment{}, identity.ErrPermissionDenied
	}
	return s.decorateURL(ctx, attachment), nil
}

func (s *Service) OpenContent(ctx context.Context, actor identity.Actor, publicID string) (Attachment, io.ReadCloser, error) {
	attachment, err := s.Get(ctx, actor, publicID)
	if err != nil {
		return Attachment{}, nil, err
	}
	settings, err := s.runtimeSettings(ctx)
	if err != nil {
		return Attachment{}, nil, err
	}
	adapter, err := s.adapterForSettings(settings, attachment.Provider)
	if err != nil {
		return Attachment{}, nil, ErrStorageUnavailable
	}
	reader, err := adapter.Open(ctx, attachment.ObjectKey)
	if err != nil {
		return Attachment{}, nil, err
	}
	return attachment, reader, nil
}

func (s *Service) List(ctx context.Context, actor identity.Actor, input AttachmentListInput) (AttachmentList, error) {
	if !actor.Can(identity.PermissionAttachmentManage) {
		return AttachmentList{}, identity.ErrPermissionDenied
	}
	// M6/L4：转义 LIKE/ILIKE 元字符，配合 store SQL 的 ESCAPE '\'，防止通配符失控匹配。
	input.Query = escapeLike(strings.TrimSpace(input.Query))
	input.ContentType = escapeLike(strings.TrimSpace(input.ContentType))
	list, err := s.store.List(ctx, input)
	if err != nil {
		return AttachmentList{}, err
	}
	for index := range list.Items {
		list.Items[index] = s.decorateURL(ctx, list.Items[index])
	}
	return list, nil
}

func (s *Service) Detail(ctx context.Context, actor identity.Actor, id int64) (AttachmentDetail, error) {
	if !actor.Can(identity.PermissionAttachmentManage) {
		return AttachmentDetail{}, identity.ErrPermissionDenied
	}
	attachment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return AttachmentDetail{}, err
	}
	references, err := s.store.ListReferences(ctx, id)
	if err != nil {
		return AttachmentDetail{}, err
	}
	return AttachmentDetail{Attachment: s.decorateURL(ctx, attachment), References: references}, nil
}

func (s *Service) UpdateStatus(ctx context.Context, actor identity.Actor, id int64, status string) (Attachment, error) {
	if !actor.Can(identity.PermissionAttachmentManage) {
		return Attachment{}, identity.ErrPermissionDenied
	}
	status = strings.TrimSpace(status)
	switch status {
	case StatusActive, StatusDisabled:
		attachment, err := s.store.UpdateStatus(ctx, id, status, false)
		if err != nil {
			return Attachment{}, err
		}
		return s.decorateURL(ctx, attachment), nil
	default:
		return Attachment{}, ErrInvalidAttachment
	}
}

func (s *Service) Delete(ctx context.Context, actor identity.Actor, id int64) (Attachment, error) {
	if !actor.Can(identity.PermissionAttachmentManage) {
		return Attachment{}, identity.ErrPermissionDenied
	}
	attachment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return Attachment{}, err
	}
	if attachment.ReferenceCount > 0 {
		return Attachment{}, ErrReferenced
	}
	updated, err := s.store.UpdateStatus(ctx, id, StatusDeleted, true)
	if err != nil {
		return Attachment{}, err
	}
	return s.decorateURL(ctx, updated), nil
}

func (s *Service) Cleanup(ctx context.Context, actor identity.Actor, limit int) (CleanupResult, error) {
	if !actor.Can(identity.PermissionAttachmentManage) {
		return CleanupResult{}, identity.ErrPermissionDenied
	}
	return s.cleanupOrphans(ctx, limit)
}

func (s *Service) CleanupOrphanAttachments(ctx context.Context, limit int) error {
	_, err := s.cleanupOrphans(ctx, limit)
	return err
}

func (s *Service) cleanupOrphans(ctx context.Context, limit int) (CleanupResult, error) {
	settings, err := s.runtimeSettings(ctx)
	if err != nil {
		return CleanupResult{}, err
	}
	cutoff := time.Now().AddDate(0, 0, -settings.CleanupOrphanAfterDays)
	items, err := s.store.ListCleanupCandidates(ctx, cutoff, limit)
	if err != nil {
		return CleanupResult{}, err
	}
	result := CleanupResult{}
	for _, item := range items {
		adapter, err := s.adapterForSettings(settings, item.Provider)
		if err != nil {
			result.Failed++
			continue
		}
		if err := adapter.Delete(ctx, item.ObjectKey); err != nil {
			result.Failed++
			continue
		}
		if err := s.store.DeleteMetadata(ctx, item.ID); err != nil {
			result.Failed++
			continue
		}
		result.Deleted++
	}
	return result, nil
}

func (s *Service) Settings(ctx context.Context, actor identity.Actor) (AttachmentSettings, error) {
	if !actor.Can(identity.PermissionAttachmentSettings) {
		return AttachmentSettings{}, identity.ErrPermissionDenied
	}
	adminOptions, err := s.options.ListAdmin(ctx, actor)
	if err != nil {
		return AttachmentSettings{}, err
	}
	values := map[string]string{}
	secrets := map[string]bool{}
	for _, item := range adminOptions {
		values[item.Name] = item.Value
		if item.Secret {
			secrets[item.Name] = item.SecretSet
		}
	}
	return settingsFromValues(values, secrets), nil
}

func (s *Service) UpdateSettings(ctx context.Context, actor identity.Actor, input AttachmentSettings) (AttachmentSettings, error) {
	if !actor.Can(identity.PermissionAttachmentSettings) {
		return AttachmentSettings{}, identity.ErrPermissionDenied
	}
	updated, err := s.options.UpdateMany(ctx, actor, settingsUpdateInputs(input))
	if err != nil {
		return AttachmentSettings{}, err
	}
	values := map[string]string{}
	secrets := map[string]bool{}
	for _, item := range updated {
		values[item.Name] = item.Value
		if item.Secret {
			secrets[item.Name] = item.SecretSet
		}
	}
	return settingsFromValues(values, secrets), nil
}

func (s *Service) Probe(ctx context.Context, actor identity.Actor) (ProbeResult, error) {
	if !actor.Can(identity.PermissionAttachmentSettings) {
		return ProbeResult{}, identity.ErrPermissionDenied
	}
	settings, err := s.runtimeSettings(ctx)
	if err != nil {
		return ProbeResult{}, err
	}
	adapter, err := s.adapterForSettings(settings, settings.Provider)
	if err != nil {
		return ProbeResult{Provider: settings.Provider, OK: false, Message: err.Error()}, nil
	}
	if err := adapter.Probe(ctx); err != nil {
		return ProbeResult{Provider: settings.Provider, OK: false, Message: err.Error()}, nil
	}
	return ProbeResult{Provider: settings.Provider, OK: true, Message: "ok"}, nil
}

func (s *Service) runtimeSettings(ctx context.Context) (AttachmentSettings, error) {
	values, err := s.options.InternalValues(ctx)
	if err != nil {
		return AttachmentSettings{}, err
	}
	return settingsFromValues(values, nil), nil
}

func (s *Service) adapterForSettings(settings AttachmentSettings, provider string) (storage.Adapter, error) {
	config := storageConfig(settings)
	config.Provider = provider
	return s.adapterFactory(config)
}

func (s *Service) decorateURL(ctx context.Context, attachment Attachment) Attachment {
	settings, err := s.runtimeSettings(ctx)
	if err != nil {
		attachment.URL = "/api/v1/attachments/" + attachment.PublicID + "/content"
		return attachment
	}
	adapter, err := s.adapterForSettings(settings, attachment.Provider)
	if err == nil {
		attachment.URL = adapter.PublicURL(attachment.ObjectKey)
	}
	if attachment.URL == "" {
		attachment.URL = "/api/v1/attachments/" + attachment.PublicID + "/content"
	}
	return attachment
}

type uploadMetadata struct {
	OriginalName string
	ContentType  string
	Extension    string
	ImageWidth   *int
	ImageHeight  *int
}

func inspectUpload(input UploadInput, settings AttachmentSettings) (uploadMetadata, error) {
	name := strings.TrimSpace(filepath.Base(input.OriginalName))
	if name == "" || name == "." {
		return uploadMetadata{}, ErrInvalidAttachment
	}
	extension := strings.ToLower(filepath.Ext(name))
	if !contains(settings.AllowedExtensions, extension) {
		return uploadMetadata{}, ErrInvalidAttachment
	}

	var sniff [512]byte
	n, _ := io.ReadFull(input.File, sniff[:])
	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(sniff[:n])
	}
	contentType = strings.ToLower(strings.Split(contentType, ";")[0])
	if !mimeAllowed(settings.AllowedMIMETypes, contentType) {
		return uploadMetadata{}, ErrInvalidAttachment
	}

	var width *int
	var height *int
	if strings.HasPrefix(contentType, "image/") {
		if _, err := input.File.Seek(0, io.SeekStart); err == nil {
			if config, _, err := image.DecodeConfig(io.LimitReader(input.File, 4*1024*1024)); err == nil {
				widthValue := config.Width
				heightValue := config.Height
				width = &widthValue
				height = &heightValue
			}
		}
	}
	if _, err := input.File.Seek(0, io.SeekStart); err != nil {
		return uploadMetadata{}, err
	}
	return uploadMetadata{
		OriginalName: name,
		ContentType:  contentType,
		Extension:    extension,
		ImageWidth:   width,
		ImageHeight:  height,
	}, nil
}

func prepareAvatarUpload(input UploadInput, avatarOptions options.AvatarOptions) (preparedUpload, error) {
	name := strings.TrimSpace(filepath.Base(input.OriginalName))
	if name == "" || name == "." || input.File == nil || input.SizeBytes <= 0 {
		return preparedUpload{}, ErrInvalidAttachment
	}
	maxBytes := int64(avatarOptions.MaxSizeKB) * 1024
	if maxBytes <= 0 || input.SizeBytes > maxBytes {
		return preparedUpload{}, ErrInvalidAttachment
	}

	config, format, err := decodeImageConfig(input.File)
	if err != nil {
		return preparedUpload{}, ErrInvalidAttachment
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > avatarOptions.MaxDimension || config.Height > avatarOptions.MaxDimension {
		return preparedUpload{}, ErrInvalidAttachment
	}

	width := config.Width
	height := config.Height
	contentType, extension, ok := avatarFormatMetadata(format)
	if !ok {
		return preparedUpload{}, ErrInvalidAttachment
	}
	if format == "gif" {
		if !avatarOptions.AllowGIF {
			return preparedUpload{}, ErrInvalidAttachment
		}
		if _, err := input.File.Seek(0, io.SeekStart); err != nil {
			return preparedUpload{}, err
		}
		return preparedUpload{
			Reader:    input.File,
			SizeBytes: input.SizeBytes,
			Metadata: uploadMetadata{
				OriginalName: normalizedAvatarName(name, extension),
				ContentType:  contentType,
				Extension:    extension,
				ImageWidth:   &width,
				ImageHeight:  &height,
			},
		}, nil
	}

	if !avatarOptions.CompressEnabled {
		if _, err := input.File.Seek(0, io.SeekStart); err != nil {
			return preparedUpload{}, err
		}
		return preparedUpload{
			Reader:    input.File,
			SizeBytes: input.SizeBytes,
			Metadata: uploadMetadata{
				OriginalName: normalizedAvatarName(name, extension),
				ContentType:  contentType,
				Extension:    extension,
				ImageWidth:   &width,
				ImageHeight:  &height,
			},
		}, nil
	}

	if _, err := input.File.Seek(0, io.SeekStart); err != nil {
		return preparedUpload{}, err
	}
	img, err := imaging.Decode(input.File, imaging.AutoOrientation(true))
	if err != nil {
		return preparedUpload{}, ErrInvalidAttachment
	}
	target := avatarOptions.TargetDimension
	if target <= 0 {
		target = 256
	}
	processed := imaging.Fill(img, target, target, imaging.Center, imaging.Lanczos)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, processed, &jpeg.Options{Quality: avatarOptions.CompressQuality}); err != nil {
		return preparedUpload{}, err
	}
	if int64(buf.Len()) > maxBytes {
		return preparedUpload{}, ErrInvalidAttachment
	}
	width = target
	height = target
	extension = ".jpg"
	contentType = "image/jpeg"
	return preparedUpload{
		Reader:    bytes.NewReader(buf.Bytes()),
		SizeBytes: int64(buf.Len()),
		Metadata: uploadMetadata{
			OriginalName: normalizedAvatarName(name, extension),
			ContentType:  contentType,
			Extension:    extension,
			ImageWidth:   &width,
			ImageHeight:  &height,
		},
	}, nil
}

func decodeImageConfig(file ReadSeekCloser) (image.Config, string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return image.Config{}, "", err
	}
	config, format, err := image.DecodeConfig(io.LimitReader(file, 8*1024*1024))
	if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil && err == nil {
		err = seekErr
	}
	return config, strings.ToLower(format), err
}

func avatarFormatMetadata(format string) (string, string, bool) {
	switch strings.ToLower(format) {
	case "jpeg":
		return "image/jpeg", ".jpg", true
	case "png":
		return "image/png", ".png", true
	case "gif":
		return "image/gif", ".gif", true
	default:
		return "", "", false
	}
}

func normalizedAvatarName(name string, extension string) string {
	base := strings.TrimSuffix(strings.TrimSpace(filepath.Base(name)), filepath.Ext(name))
	if base == "" || base == "." {
		base = "avatar"
	}
	return base + extension
}

func renderObjectKey(template string, publicID string, extension string, now time.Time) (string, error) {
	replacements := map[string]string{
		"{yyyy}":      now.Format("2006"),
		"{mm}":        now.Format("01"),
		"{dd}":        now.Format("02"),
		"{public_id}": publicID,
		"{ext}":       extension,
	}
	key := template
	for placeholder, value := range replacements {
		key = strings.ReplaceAll(key, placeholder, value)
	}
	key = path.Clean(strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/"))
	if key == "." || key == "" || strings.HasPrefix(key, "../") || strings.Contains(key, "/../") {
		return "", ErrInvalidAttachment
	}
	return key, nil
}

func randomPublicID() (string, error) {
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(token[:]), nil
}

func canViewAttachment(actor identity.Actor, attachment Attachment) bool {
	if attachment.Status != StatusActive {
		return actor.Can(identity.PermissionAttachmentManage)
	}
	if attachment.Visibility == VisibilityPublic {
		return true
	}
	if attachment.Owner != nil && attachment.Owner.ID == actor.ID && actor.IsActive() {
		return true
	}
	return actor.Can(identity.PermissionAttachmentManage)
}

func settingsFromValues(values map[string]string, secrets map[string]bool) AttachmentSettings {
	return AttachmentSettings{
		Provider:               read(values, options.NameAttachmentProvider, storage.ProviderLocal),
		UploadEnabled:          enabled(values, options.NameAttachmentUploadEnabled, true),
		PathTemplate:           read(values, options.NameAttachmentPathTemplate, "{yyyy}/{mm}/{dd}/{public_id}{ext}"),
		PublicBaseURL:          read(values, options.NameAttachmentPublicBaseURL, ""),
		MaxFileSizeMB:          readInt(values, options.NameAttachmentMaxFileSizeMB, 20),
		AllowedExtensions:      splitList(read(values, options.NameAttachmentAllowedExtensions, ".jpg,.jpeg,.png,.gif,.webp,.pdf,.txt,.zip")),
		AllowedMIMETypes:       splitList(read(values, options.NameAttachmentAllowedMIMETypes, "image/jpeg,image/png,image/gif,image/webp,application/pdf,text/plain,application/zip")),
		DefaultVisibility:      read(values, options.NameAttachmentDefaultVisibility, VisibilityPublic),
		CleanupOrphanAfterDays: readInt(values, options.NameAttachmentCleanupOrphanDays, 30),
		Local: LocalSettings{
			Root:         read(values, options.NameAttachmentLocalRoot, "storage/app/attachments"),
			PublicPrefix: read(values, options.NameAttachmentLocalPublicPrefix, ""),
		},
		AliyunOSS: AliyunOSSSettings{
			Endpoint:           read(values, options.NameAttachmentAliyunEndpoint, ""),
			Bucket:             read(values, options.NameAttachmentAliyunBucket, ""),
			Region:             read(values, options.NameAttachmentAliyunRegion, ""),
			AccessKeyID:        read(values, options.NameAttachmentAliyunAccessKeyID, ""),
			AccessKeySecret:    read(values, options.NameAttachmentAliyunAccessKeySecret, ""),
			AccessKeySecretSet: secretSet(secrets, options.NameAttachmentAliyunAccessKeySecret),
		},
		TencentCOS: TencentCOSSettings{
			Region:       read(values, options.NameAttachmentTencentRegion, ""),
			Bucket:       read(values, options.NameAttachmentTencentBucket, ""),
			SecretID:     read(values, options.NameAttachmentTencentSecretID, ""),
			SecretKey:    read(values, options.NameAttachmentTencentSecretKey, ""),
			SecretKeySet: secretSet(secrets, options.NameAttachmentTencentSecretKey),
			CDNDomain:    read(values, options.NameAttachmentTencentCDNDomain, ""),
		},
		FTP: FTPSettings{
			Host:          read(values, options.NameAttachmentFTPHost, ""),
			Port:          readInt(values, options.NameAttachmentFTPPort, 21),
			Username:      read(values, options.NameAttachmentFTPUsername, ""),
			Password:      read(values, options.NameAttachmentFTPPassword, ""),
			PasswordSet:   secretSet(secrets, options.NameAttachmentFTPPassword),
			RootPath:      read(values, options.NameAttachmentFTPRootPath, "/"),
			Passive:       enabled(values, options.NameAttachmentFTPPassive, true),
			ExplicitTLS:   enabled(values, options.NameAttachmentFTPExplicitTLS, false),
			PublicBaseURL: read(values, options.NameAttachmentFTPPublicBaseURL, ""),
		},
		SFTP: SFTPSettings{
			Host:               read(values, options.NameAttachmentSFTPHost, ""),
			Port:               readInt(values, options.NameAttachmentSFTPPort, 22),
			Username:           read(values, options.NameAttachmentSFTPUsername, ""),
			Password:           read(values, options.NameAttachmentSFTPPassword, ""),
			PasswordSet:        secretSet(secrets, options.NameAttachmentSFTPPassword),
			PrivateKey:         read(values, options.NameAttachmentSFTPPrivateKey, ""),
			PrivateKeySet:      secretSet(secrets, options.NameAttachmentSFTPPrivateKey),
			Passphrase:         read(values, options.NameAttachmentSFTPPassphrase, ""),
			PassphraseSet:      secretSet(secrets, options.NameAttachmentSFTPPassphrase),
			RootPath:           read(values, options.NameAttachmentSFTPRootPath, "/"),
			HostKeyFingerprint: read(values, options.NameAttachmentSFTPHostKeyFingerprint, ""),
			PublicBaseURL:      read(values, options.NameAttachmentSFTPPublicBaseURL, ""),
		},
	}
}

func settingsUpdateInputs(settings AttachmentSettings) []options.UpdateInput {
	inputs := []options.UpdateInput{
		{Name: options.NameAttachmentProvider, Value: settings.Provider},
		{Name: options.NameAttachmentUploadEnabled, Value: enabledValue(settings.UploadEnabled)},
		{Name: options.NameAttachmentPathTemplate, Value: settings.PathTemplate},
		{Name: options.NameAttachmentPublicBaseURL, Value: settings.PublicBaseURL},
		{Name: options.NameAttachmentMaxFileSizeMB, Value: strconv.Itoa(settings.MaxFileSizeMB)},
		{Name: options.NameAttachmentAllowedExtensions, Value: strings.Join(settings.AllowedExtensions, ",")},
		{Name: options.NameAttachmentAllowedMIMETypes, Value: strings.Join(settings.AllowedMIMETypes, ",")},
		{Name: options.NameAttachmentDefaultVisibility, Value: settings.DefaultVisibility},
		{Name: options.NameAttachmentCleanupOrphanDays, Value: strconv.Itoa(settings.CleanupOrphanAfterDays)},
		{Name: options.NameAttachmentLocalRoot, Value: settings.Local.Root},
		{Name: options.NameAttachmentLocalPublicPrefix, Value: settings.Local.PublicPrefix},
		{Name: options.NameAttachmentAliyunEndpoint, Value: settings.AliyunOSS.Endpoint},
		{Name: options.NameAttachmentAliyunBucket, Value: settings.AliyunOSS.Bucket},
		{Name: options.NameAttachmentAliyunRegion, Value: settings.AliyunOSS.Region},
		{Name: options.NameAttachmentAliyunAccessKeyID, Value: settings.AliyunOSS.AccessKeyID},
		{Name: options.NameAttachmentTencentRegion, Value: settings.TencentCOS.Region},
		{Name: options.NameAttachmentTencentBucket, Value: settings.TencentCOS.Bucket},
		{Name: options.NameAttachmentTencentSecretID, Value: settings.TencentCOS.SecretID},
		{Name: options.NameAttachmentTencentCDNDomain, Value: settings.TencentCOS.CDNDomain},
		{Name: options.NameAttachmentFTPHost, Value: settings.FTP.Host},
		{Name: options.NameAttachmentFTPPort, Value: strconv.Itoa(settings.FTP.Port)},
		{Name: options.NameAttachmentFTPUsername, Value: settings.FTP.Username},
		{Name: options.NameAttachmentFTPRootPath, Value: settings.FTP.RootPath},
		{Name: options.NameAttachmentFTPPassive, Value: enabledValue(settings.FTP.Passive)},
		{Name: options.NameAttachmentFTPExplicitTLS, Value: enabledValue(settings.FTP.ExplicitTLS)},
		{Name: options.NameAttachmentFTPPublicBaseURL, Value: settings.FTP.PublicBaseURL},
		{Name: options.NameAttachmentSFTPHost, Value: settings.SFTP.Host},
		{Name: options.NameAttachmentSFTPPort, Value: strconv.Itoa(settings.SFTP.Port)},
		{Name: options.NameAttachmentSFTPUsername, Value: settings.SFTP.Username},
		{Name: options.NameAttachmentSFTPRootPath, Value: settings.SFTP.RootPath},
		{Name: options.NameAttachmentSFTPHostKeyFingerprint, Value: settings.SFTP.HostKeyFingerprint},
		{Name: options.NameAttachmentSFTPPublicBaseURL, Value: settings.SFTP.PublicBaseURL},
	}
	secretInputs := []options.UpdateInput{
		{Name: options.NameAttachmentAliyunAccessKeySecret, Value: settings.AliyunOSS.AccessKeySecret},
		{Name: options.NameAttachmentTencentSecretKey, Value: settings.TencentCOS.SecretKey},
		{Name: options.NameAttachmentFTPPassword, Value: settings.FTP.Password},
		{Name: options.NameAttachmentSFTPPassword, Value: settings.SFTP.Password},
		{Name: options.NameAttachmentSFTPPrivateKey, Value: settings.SFTP.PrivateKey},
		{Name: options.NameAttachmentSFTPPassphrase, Value: settings.SFTP.Passphrase},
	}
	for _, input := range secretInputs {
		if strings.TrimSpace(input.Value) != "" {
			inputs = append(inputs, input)
		}
	}
	return inputs
}

func storageConfig(settings AttachmentSettings) storage.Config {
	return storage.Config{
		Provider:      settings.Provider,
		LocalRoot:     settings.Local.Root,
		PublicBaseURL: settings.PublicBaseURL,
		Local: storage.LocalConfig{
			PublicPrefix: settings.Local.PublicPrefix,
		},
		OSS: storage.AliyunOSSConfig{
			Endpoint:        settings.AliyunOSS.Endpoint,
			Bucket:          settings.AliyunOSS.Bucket,
			Region:          settings.AliyunOSS.Region,
			AccessKeyID:     settings.AliyunOSS.AccessKeyID,
			AccessKeySecret: settings.AliyunOSS.AccessKeySecret,
		},
		COS: storage.TencentCOSConfig{
			Region:    settings.TencentCOS.Region,
			Bucket:    settings.TencentCOS.Bucket,
			SecretID:  settings.TencentCOS.SecretID,
			SecretKey: settings.TencentCOS.SecretKey,
			CDNDomain: settings.TencentCOS.CDNDomain,
		},
		FTP: storage.FTPConfig{
			Host:          settings.FTP.Host,
			Port:          settings.FTP.Port,
			Username:      settings.FTP.Username,
			Password:      settings.FTP.Password,
			RootPath:      settings.FTP.RootPath,
			Passive:       settings.FTP.Passive,
			ExplicitTLS:   settings.FTP.ExplicitTLS,
			PublicBaseURL: settings.FTP.PublicBaseURL,
		},
		SFTP: storage.SFTPConfig{
			Host:               settings.SFTP.Host,
			Port:               settings.SFTP.Port,
			Username:           settings.SFTP.Username,
			Password:           settings.SFTP.Password,
			PrivateKey:         settings.SFTP.PrivateKey,
			Passphrase:         settings.SFTP.Passphrase,
			RootPath:           settings.SFTP.RootPath,
			HostKeyFingerprint: settings.SFTP.HostKeyFingerprint,
			PublicBaseURL:      settings.SFTP.PublicBaseURL,
		},
	}
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(value)) {
			return true
		}
	}
	return false
}

func mimeAllowed(allowed []string, contentType string) bool {
	for _, item := range allowed {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == contentType || item == "*/*" {
			return true
		}
		if strings.HasSuffix(item, "/*") && strings.HasPrefix(contentType, strings.TrimSuffix(item, "*")) {
			return true
		}
	}
	return false
}

func read(values map[string]string, name string, fallback string) string {
	if values == nil {
		return fallback
	}
	value := strings.TrimSpace(values[name])
	if value == "" {
		return fallback
	}
	return value
}

func readInt(values map[string]string, name string, fallback int) int {
	parsed, err := strconv.Atoi(read(values, name, ""))
	if err != nil {
		return fallback
	}
	return parsed
}

func enabled(values map[string]string, name string, fallback bool) bool {
	switch strings.ToLower(read(values, name, "")) {
	case "enabled", "true", "1", "yes", "on":
		return true
	case "disabled", "false", "0", "no", "off":
		return false
	default:
		return fallback
	}
}

func enabledValue(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func secretSet(secrets map[string]bool, name string) bool {
	if secrets == nil {
		return false
	}
	return secrets[name]
}

func (s AttachmentSettings) String() string {
	return fmt.Sprintf("provider=%s upload=%t", s.Provider, s.UploadEnabled)
}

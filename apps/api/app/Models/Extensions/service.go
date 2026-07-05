package extensions

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

const maxArchiveBytes = 50 * 1024 * 1024

var manifestIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,80}$`)

type Service struct {
	store         Store
	extensionRoot string
	builtinRoot   string
	runtime       RuntimeManager
	themeBuilder  ThemeBuilder
}

func NewService(store Store, extensionRoot string) *Service {
	return NewServiceWithHooks(store, extensionRoot, nil, nil)
}

func NewServiceWithHooks(store Store, extensionRoot string, runtime RuntimePreflight, themeBuilder ThemeBuilder) *Service {
	return NewServiceWithBuiltinsAndHooks(store, extensionRoot, "", runtime, themeBuilder)
}

func NewServiceWithBuiltins(store Store, extensionRoot string, builtinRoot string) *Service {
	return NewServiceWithBuiltinsAndRuntime(store, extensionRoot, builtinRoot, nil, nil)
}

func NewServiceWithBuiltinsAndHooks(store Store, extensionRoot string, builtinRoot string, runtime RuntimePreflight, themeBuilder ThemeBuilder) *Service {
	var manager RuntimeManager
	if runtime != nil {
		if full, ok := runtime.(RuntimeManager); ok {
			manager = full
		} else {
			manager = preflightRuntimeManager{RuntimePreflight: runtime}
		}
	}
	return NewServiceWithBuiltinsAndRuntime(store, extensionRoot, builtinRoot, manager, themeBuilder)
}

func NewServiceWithRuntime(store Store, extensionRoot string, runtime RuntimeManager, themeBuilder ThemeBuilder) *Service {
	return NewServiceWithBuiltinsAndRuntime(store, extensionRoot, "", runtime, themeBuilder)
}

func NewServiceWithBuiltinsAndRuntime(store Store, extensionRoot string, builtinRoot string, runtime RuntimeManager, themeBuilder ThemeBuilder) *Service {
	if strings.TrimSpace(extensionRoot) == "" {
		extensionRoot = "storage/extensions"
	}
	if runtime == nil {
		runtime = LocalRuntimeManager{}
	}
	if themeBuilder == nil {
		themeBuilder = LocalThemeBuilder{}
	}
	return &Service{
		store:         store,
		extensionRoot: extensionRoot,
		builtinRoot:   strings.TrimSpace(builtinRoot),
		runtime:       runtime,
		themeBuilder:  themeBuilder,
	}
}

type LocalRuntimePreflight struct{}

func (LocalRuntimePreflight) Check(_ context.Context, extension Extension) error {
	if extension.Manifest.Backend.Entry == "" {
		return nil
	}
	entry, ok := installedFilePath(extension, extension.Manifest.Backend.Entry)
	if !ok {
		return ErrInvalidManifest
	}
	info, err := os.Stat(entry)
	if err != nil || info.IsDir() {
		return fmt.Errorf("backend entry %s is not available", extension.Manifest.Backend.Entry)
	}
	return nil
}

type LocalRuntimeManager struct {
	LocalRuntimePreflight
}

func (LocalRuntimeManager) Start(context.Context, Extension) error {
	return nil
}

func (LocalRuntimeManager) Stop(context.Context, Extension) error {
	return nil
}

func (LocalRuntimeManager) Status(_ context.Context, extension Extension) RuntimeStatus {
	return RuntimeStatus{
		State:         RuntimeStopped,
		RouteCount:    len(extension.Manifest.Routes),
		HookCount:     len(extension.Manifest.Hooks),
		ProviderCount: len(extension.Manifest.Providers),
	}
}

type preflightRuntimeManager struct {
	RuntimePreflight
}

func (preflightRuntimeManager) Start(context.Context, Extension) error {
	return nil
}

func (preflightRuntimeManager) Stop(context.Context, Extension) error {
	return nil
}

func (preflightRuntimeManager) Status(_ context.Context, extension Extension) RuntimeStatus {
	return RuntimeStatus{
		State:         RuntimeStopped,
		RouteCount:    len(extension.Manifest.Routes),
		HookCount:     len(extension.Manifest.Hooks),
		ProviderCount: len(extension.Manifest.Providers),
	}
}

type LocalThemeBuilder struct{}

func (LocalThemeBuilder) Build(_ context.Context, extension Extension) error {
	if extension.Manifest.Frontend.Layer == "" {
		return nil
	}
	layer, ok := installedFilePath(extension, extension.Manifest.Frontend.Layer)
	if !ok {
		return ErrInvalidManifest
	}
	info, err := os.Stat(layer)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("nuxt layer %s is not available", extension.Manifest.Frontend.Layer)
	}
	return nil
}

func (s *Service) List(ctx context.Context, actor identity.Actor) ([]Extension, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
		return nil, identity.ErrPermissionDenied
	}
	items, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index] = s.decorateRuntime(ctx, items[index])
	}
	return items, nil
}

func (s *Service) SyncBuiltins(ctx context.Context) ([]Extension, error) {
	if strings.TrimSpace(s.builtinRoot) == "" {
		return nil, nil
	}

	groups := []struct {
		dir           string
		extensionType string
	}{
		{dir: "plugins", extensionType: TypePlugin},
		{dir: "themes", extensionType: TypeTheme},
	}
	items := []Extension{}
	for _, group := range groups {
		root := filepath.Join(s.builtinRoot, group.dir)
		entries, err := os.ReadDir(root)
		if errorsIsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			manifestPath := filepath.Join(root, entry.Name(), ManifestFileName)
			body, err := os.ReadFile(manifestPath)
			if err != nil {
				return nil, err
			}
			var manifest Manifest
			if err := json.Unmarshal(body, &manifest); err != nil {
				return nil, ErrInvalidManifest
			}
			manifest = normalizeManifest(manifest)
			if manifest.Type != group.extensionType {
				return nil, ErrInvalidManifest
			}
			if err := validateManifest(manifest); err != nil {
				return nil, err
			}
			item, err := s.store.SaveBuiltin(ctx, SaveBuiltinInput{
				Manifest:    manifest,
				PackagePath: filepath.Dir(manifestPath),
			})
			if err != nil {
				return nil, err
			}
			_, _ = s.store.CreateEvent(ctx, EventInput{
				ExtensionID: item.ID,
				Action:      EventBuiltinSynced,
				Message:     "Built-in extension synchronized from Git.",
			})
			items = append(items, item)
		}
	}
	if _, err := s.EnsureDefaultThemeActive(ctx); err != nil && !errors.Is(err, ErrExtensionNotFound) {
		return nil, err
	}
	return items, nil
}

func (s *Service) Events(ctx context.Context, actor identity.Actor, extensionID string, limit int) ([]ExtensionEvent, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
		return nil, identity.ErrPermissionDenied
	}
	return s.store.ListEvents(ctx, normalizeID(extensionID), limit)
}

func (s *Service) InstallArchive(ctx context.Context, actor identity.Actor, input ArchiveInput) (Extension, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
		return Extension{}, identity.ErrPermissionDenied
	}
	if len(input.Data) == 0 || len(input.Data) > maxArchiveBytes {
		return Extension{}, ErrInvalidArchive
	}

	manifest, files, err := readArchive(input.Data)
	if err != nil {
		return Extension{}, err
	}
	if err := validateManifest(manifest); err != nil {
		return Extension{}, err
	}
	if manifest.ID == DefaultThemeID {
		return Extension{}, ErrInvalidManifest
	}

	versionDir := filepath.Join(s.extensionRoot, manifest.ID, manifest.Version)
	if err := os.MkdirAll(filepath.Join(versionDir, "files"), 0o755); err != nil {
		return Extension{}, err
	}
	packagePath := filepath.Join(versionDir, "package.zip")
	if err := os.WriteFile(packagePath, input.Data, 0o600); err != nil {
		return Extension{}, err
	}
	if err := writeManifest(versionDir, manifest); err != nil {
		return Extension{}, err
	}
	if err := extractArchiveFiles(versionDir, files); err != nil {
		return Extension{}, err
	}

	installed, err := s.store.SaveInstalled(ctx, SaveInstalledInput{
		Manifest:    manifest,
		PackagePath: packagePath,
	})
	if err != nil {
		return Extension{}, err
	}
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: installed.ID,
		ActorUserID: actor.ID,
		Action:      EventInstalled,
		Message:     "Extension archive installed.",
	})
	return s.decorateRuntime(ctx, installed), nil
}

func (s *Service) Enable(ctx context.Context, actor identity.Actor, id string) (Extension, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
		return Extension{}, identity.ErrPermissionDenied
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return Extension{}, err
	}
	if extension.Type == TypeTheme {
		return Extension{}, ErrThemeActivationRequired
	}

	if extension.Type == TypePlugin && extension.Manifest.Backend.Entry != "" && s.runtime != nil {
		if err := s.runtime.Check(ctx, extension); err != nil {
			s.recordEnableFailure(ctx, actor, extension.ID, err)
			return Extension{}, fmt.Errorf("%w: %v", ErrPreflightFailed, err)
		}
	}
	enabled, err := s.store.Enable(ctx, extension.ID, extension.Type)
	if err != nil {
		return Extension{}, err
	}
	if enabled.Type == TypePlugin && enabled.Manifest.Backend.Entry != "" && s.runtime != nil {
		if err := s.runtime.Start(ctx, enabled); err != nil {
			_, _ = s.store.Disable(ctx, enabled.ID)
			s.recordEnableFailure(ctx, actor, enabled.ID, err)
			return Extension{}, fmt.Errorf("%w: %v", ErrRuntimeFailed, err)
		}
	}
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: enabled.ID,
		ActorUserID: actor.ID,
		Action:      EventEnabled,
		Message:     "Extension enabled.",
	})
	return s.decorateRuntime(ctx, enabled), nil
}

func (s *Service) Disable(ctx context.Context, actor identity.Actor, id string) (Extension, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
		return Extension{}, identity.ErrPermissionDenied
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return Extension{}, err
	}
	if extension.Type == TypeTheme {
		return Extension{}, ErrThemeActivationRequired
	}
	disabled, err := s.store.Disable(ctx, extension.ID)
	if err != nil {
		return Extension{}, err
	}
	if disabled.Type == TypePlugin && s.runtime != nil {
		_ = s.runtime.Stop(ctx, disabled)
	}
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: disabled.ID,
		ActorUserID: actor.ID,
		Action:      EventDisabled,
		Message:     "Extension disabled.",
	})
	return s.decorateRuntime(ctx, disabled), nil
}

func (s *Service) VerifyExtension(ctx context.Context, actor identity.Actor, id string) (Extension, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
		return Extension{}, identity.ErrPermissionDenied
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return Extension{}, err
	}
	if err := s.verifyExtension(ctx, extension); err != nil {
		_, _ = s.store.CreateEvent(ctx, EventInput{
			ExtensionID: extension.ID,
			ActorUserID: actor.ID,
			Action:      EventEnableFailed,
			Message:     err.Error(),
		})
		return Extension{}, err
	}
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: extension.ID,
		ActorUserID: actor.ID,
		Action:      EventVerified,
		Message:     "Extension preflight verified.",
	})
	return s.decorateRuntime(ctx, extension), nil
}

func (s *Service) ActivateTheme(ctx context.Context, actor identity.Actor, id string) (Extension, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
		return Extension{}, identity.ErrPermissionDenied
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return Extension{}, err
	}
	if extension.Type != TypeTheme {
		return Extension{}, ErrThemeActivationRequired
	}
	if extension.ID != DefaultThemeID || extension.Source != SourceBuiltin {
		return Extension{}, ErrThemeRuntimeUnavailable
	}
	if err := s.verifyExtension(ctx, extension); err != nil {
		s.recordEnableFailure(ctx, actor, extension.ID, err)
		return Extension{}, err
	}
	active, err := s.store.ActivateTheme(ctx, extension.ID)
	if err != nil {
		return Extension{}, err
	}
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: active.ID,
		ActorUserID: actor.ID,
		Action:      EventThemeActivated,
		Message:     "Theme activated.",
	})
	return active, nil
}

func (s *Service) EnsureDefaultThemeActive(ctx context.Context) (Extension, error) {
	active, err := s.store.ActiveTheme(ctx)
	if err == nil && active.ID == DefaultThemeID && active.Source == SourceBuiltin {
		return active, nil
	}
	defaultTheme, getErr := s.store.Get(ctx, DefaultThemeID)
	if getErr != nil {
		if err != nil {
			return Extension{}, err
		}
		return Extension{}, getErr
	}
	if defaultTheme.Type != TypeTheme || defaultTheme.Source != SourceBuiltin {
		return Extension{}, ErrInvalidManifest
	}
	return s.store.ActivateTheme(ctx, DefaultThemeID)
}

func (s *Service) verifyExtension(ctx context.Context, extension Extension) error {
	if extension.Type == TypePlugin && extension.Manifest.Backend.Entry != "" && s.runtime != nil {
		if err := s.runtime.Check(ctx, extension); err != nil {
			return fmt.Errorf("%w: %v", ErrPreflightFailed, err)
		}
	}
	if extension.Type == TypeTheme && extension.Manifest.Frontend.Layer != "" && s.themeBuilder != nil {
		if err := s.themeBuilder.Build(ctx, extension); err != nil {
			return fmt.Errorf("%w: %v", ErrBuildFailed, err)
		}
	}
	return nil
}

func (s *Service) recordEnableFailure(ctx context.Context, actor identity.Actor, extensionID string, cause error) {
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: extensionID,
		ActorUserID: actor.ID,
		Action:      EventEnableFailed,
		Message:     cause.Error(),
	})
}

func (s *Service) decorateRuntime(ctx context.Context, item Extension) Extension {
	if item.Type == TypePlugin && s.runtime != nil {
		status := s.runtime.Status(ctx, item)
		item.Runtime = &status
	}
	return item
}

type archiveFile struct {
	name string
	mode os.FileMode
	body []byte
}

func readArchive(data []byte) (Manifest, []archiveFile, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Manifest{}, nil, ErrInvalidArchive
	}

	var manifest Manifest
	manifestFound := false
	files := []archiveFile{}
	var total uint64
	for _, file := range reader.File {
		name, ok := safeArchivePath(file.Name)
		if !ok {
			return Manifest{}, nil, ErrInvalidArchive
		}
		if file.FileInfo().IsDir() {
			continue
		}
		total += file.UncompressedSize64
		if total > maxArchiveBytes {
			return Manifest{}, nil, ErrInvalidArchive
		}
		body, err := readZipFile(file)
		if err != nil {
			return Manifest{}, nil, ErrInvalidArchive
		}
		if name == ManifestFileName {
			if err := json.Unmarshal(body, &manifest); err != nil {
				return Manifest{}, nil, ErrInvalidManifest
			}
			manifestFound = true
			continue
		}
		files = append(files, archiveFile{name: name, mode: file.Mode(), body: body})
	}
	if !manifestFound {
		return Manifest{}, nil, ErrInvalidArchive
	}
	return normalizeManifest(manifest), files, nil
}

func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func writeManifest(versionDir string, manifest Manifest) error {
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(versionDir, ManifestFileName), body, 0o600)
}

func extractArchiveFiles(versionDir string, files []archiveFile) error {
	root := filepath.Join(versionDir, "files")
	for _, file := range files {
		target := filepath.Join(root, filepath.FromSlash(file.name))
		if !strings.HasPrefix(target, root+string(os.PathSeparator)) {
			return ErrInvalidArchive
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		mode := file.mode.Perm()
		if mode == 0 {
			mode = 0o644
		}
		if err := os.WriteFile(target, file.body, mode); err != nil {
			return err
		}
	}
	return nil
}

func installedFilePath(extension Extension, manifestPath string) (string, bool) {
	name, ok := safeArchivePath(manifestPath)
	if !ok {
		return "", false
	}
	root := filepath.Join(filepath.Dir(extension.PackagePath), "files")
	if extension.Source == SourceBuiltin {
		// 内置扩展直接位于 Git 包目录；上传扩展才解压到相邻 files 目录。
		root = filepath.Clean(extension.PackagePath)
	}
	target := filepath.Join(root, filepath.FromSlash(name))
	return target, strings.HasPrefix(target, root+string(os.PathSeparator))
}

func validateManifest(manifest Manifest) error {
	if !manifestIDPattern.MatchString(manifest.ID) {
		return ErrInvalidManifest
	}
	if manifest.Name == "" || manifest.Version == "" || manifest.SForumVersion == "" {
		return ErrInvalidManifest
	}
	if manifest.Type != TypePlugin && manifest.Type != TypeTheme {
		return ErrInvalidManifest
	}
	if manifest.Backend.Entry != "" {
		if _, ok := safeArchivePath(manifest.Backend.Entry); !ok {
			return ErrInvalidManifest
		}
	}
	if manifest.Backend.RPC != "" && manifest.Backend.RPC != "hashicorp-go-plugin" {
		return ErrInvalidManifest
	}
	if manifest.Backend.ProtocolVersion < 0 || manifest.Backend.ProtocolVersion > 1 {
		return ErrInvalidManifest
	}
	if manifest.Frontend.Layer != "" {
		if _, ok := safeArchivePath(manifest.Frontend.Layer); !ok {
			return ErrInvalidManifest
		}
	}
	for _, migration := range manifest.Migrations {
		if _, ok := safeArchivePath(migration.Path); !ok || !strings.HasSuffix(migration.Path, ".sql") {
			return ErrInvalidManifest
		}
	}
	for _, route := range manifest.Routes {
		if route.Path == "" || !strings.HasPrefix(route.Path, "/") || strings.Contains(route.Path, "..") {
			return ErrInvalidManifest
		}
		access := route.Access
		if access == "" {
			access = RouteAccessLogin
		}
		if access != RouteAccessPublic && access != RouteAccessLogin && access != RouteAccessPermission {
			return ErrInvalidManifest
		}
		if len(route.Methods) == 0 {
			return ErrInvalidManifest
		}
		for _, method := range route.Methods {
			switch method {
			case "GET", "HEAD", "OPTIONS", "POST", "PUT", "PATCH", "DELETE":
			default:
				return ErrInvalidManifest
			}
			if access == RouteAccessPublic && method != "GET" && method != "HEAD" && method != "OPTIONS" {
				return ErrInvalidManifest
			}
		}
		if access == RouteAccessPermission && (route.Permission == "" || !manifestHasPermission(manifest, route.Permission)) {
			return ErrInvalidManifest
		}
		if route.TimeoutMS < 0 {
			return ErrInvalidManifest
		}
	}
	for _, hook := range manifest.Hooks {
		if !knownHookPoint(hook.Name) {
			return ErrInvalidManifest
		}
	}
	for _, provider := range manifest.Providers {
		if provider.Label == "" || !knownProviderSlot(provider.Slot) || provider.TimeoutMS < 0 {
			return ErrInvalidManifest
		}
	}
	return nil
}

func normalizeManifest(manifest Manifest) Manifest {
	manifest.ID = normalizeID(manifest.ID)
	manifest.Name = strings.TrimSpace(manifest.Name)
	manifest.Version = strings.TrimSpace(manifest.Version)
	manifest.Type = strings.ToLower(strings.TrimSpace(manifest.Type))
	manifest.SForumVersion = strings.TrimSpace(manifest.SForumVersion)
	manifest.Backend.Entry = strings.TrimSpace(manifest.Backend.Entry)
	manifest.Backend.RPC = strings.TrimSpace(manifest.Backend.RPC)
	if manifest.Backend.ProtocolVersion == 0 && manifest.Backend.RPC != "" {
		manifest.Backend.ProtocolVersion = 1
	}
	manifest.Frontend.Layer = strings.TrimSpace(manifest.Frontend.Layer)
	for index := range manifest.Routes {
		manifest.Routes[index].Path = normalizeRoutePath(manifest.Routes[index].Path)
		manifest.Routes[index].Access = strings.ToLower(strings.TrimSpace(manifest.Routes[index].Access))
		manifest.Routes[index].Permission = strings.TrimSpace(manifest.Routes[index].Permission)
		for methodIndex := range manifest.Routes[index].Methods {
			manifest.Routes[index].Methods[methodIndex] = strings.ToUpper(strings.TrimSpace(manifest.Routes[index].Methods[methodIndex]))
		}
	}
	for index := range manifest.Hooks {
		manifest.Hooks[index].Name = strings.TrimSpace(manifest.Hooks[index].Name)
	}
	for index := range manifest.Providers {
		manifest.Providers[index].Slot = strings.TrimSpace(manifest.Providers[index].Slot)
		manifest.Providers[index].Label = strings.TrimSpace(manifest.Providers[index].Label)
	}
	return manifest
}

func normalizeID(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

func safeArchivePath(name string) (string, bool) {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	if name == "" || strings.HasPrefix(name, "/") {
		return "", false
	}
	clean := path.Clean(name)
	if clean == "." || clean == ManifestFileName {
		return clean, true
	}
	if strings.HasPrefix(clean, "../") || clean == ".." || strings.Contains(clean, "/../") {
		return "", false
	}
	return clean, true
}

func normalizeRoutePath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return ""
	}
	if strings.Contains(value, "..") {
		return value
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return path.Clean(value)
}

func knownHookPoint(name string) bool {
	switch name {
	case "extension.enabled", "extension.disabled", "user.registered", "topic.before_create", "topic.created", "attachment.uploaded":
		return true
	default:
		return false
	}
}

func knownProviderSlot(slot string) bool {
	switch slot {
	case "search.provider", "attachment.storage.provider", "human_verification.provider", "auth.risk.provider", "editor.sanitizer.provider":
		return true
	default:
		return false
	}
}

func manifestHasPermission(manifest Manifest, permission string) bool {
	for _, item := range manifest.Permissions {
		if strings.TrimSpace(item) == permission {
			return true
		}
	}
	return false
}

func errorsIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}

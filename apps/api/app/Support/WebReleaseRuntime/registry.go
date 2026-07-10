package webreleaseruntime

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

type RegistryExtension struct {
	SourceRoot string
	Snapshot   extensions.WebReleaseExtension
}

type RegistryInput struct {
	Root       string
	ReleaseID  int64
	Extensions []RegistryExtension
}

type RegistryResult struct {
	MetadataPath string
	RegistryPath string
}

type registryMetadata struct {
	Point          string            `json:"point"`
	ExtensionID    string            `json:"extensionId"`
	ContributionID string            `json:"contributionId"`
	ComponentID    string            `json:"componentId"`
	Order          int               `json:"order"`
	Label          map[string]string `json:"label"`
	Options        map[string]any    `json:"options"`
	modulePath     string
}

func GenerateRegistry(input RegistryInput) (RegistryResult, error) {
	if input.ReleaseID <= 0 || strings.TrimSpace(input.Root) == "" {
		return RegistryResult{}, fmt.Errorf("web release registry requires a release id and root")
	}
	root, err := filepath.Abs(input.Root)
	if err != nil {
		return RegistryResult{}, err
	}
	items := make([]registryMetadata, 0)
	localeMessages := make(map[string]map[string]json.RawMessage)
	seen := make(map[string]struct{})
	for _, item := range input.Extensions {
		snapshot := item.Snapshot
		frontendRoot, err := secureChildDirectory(item.SourceRoot, snapshot.FrontendRoot)
		if err != nil {
			return RegistryResult{}, fmt.Errorf("extension %s frontend root: %w", snapshot.ExtensionID, err)
		}
		locales := make(map[string]json.RawMessage, len(snapshot.LocaleMap))
		for locale, relative := range snapshot.LocaleMap {
			body, err := secureChildFile(frontendRoot, relative)
			if err != nil {
				return RegistryResult{}, fmt.Errorf("extension %s locale %s: %w", snapshot.ExtensionID, locale, err)
			}
			var messages map[string]any
			if err := json.Unmarshal(body, &messages); err != nil {
				return RegistryResult{}, fmt.Errorf("extension %s locale %s: %w", snapshot.ExtensionID, locale, err)
			}
			canonical, _ := json.Marshal(messages)
			locales[locale] = canonical
		}
		localeMessages[snapshot.ExtensionID] = locales

		for _, contribution := range snapshot.TrustedComponents {
			var payload map[string]any
			if err := json.Unmarshal(contribution.Payload, &payload); err != nil {
				return RegistryResult{}, fmt.Errorf("extension %s contribution %s payload: %w", snapshot.ExtensionID, contribution.ID, err)
			}
			componentID, ok := payload["component"].(string)
			if !ok || strings.TrimSpace(componentID) == "" {
				return RegistryResult{}, fmt.Errorf("extension %s contribution %s has no component", snapshot.ExtensionID, contribution.ID)
			}
			componentPath := snapshot.ComponentMap[componentID]
			module, err := secureChildPath(frontendRoot, componentPath, false)
			if err != nil {
				return RegistryResult{}, fmt.Errorf("extension %s component %s: %w", snapshot.ExtensionID, componentID, err)
			}
			delete(payload, "component")
			key := snapshot.ExtensionID + ":" + contribution.ID
			if _, duplicate := seen[key]; duplicate {
				return RegistryResult{}, fmt.Errorf("duplicate admin extension contribution %s", key)
			}
			seen[key] = struct{}{}
			items = append(items, registryMetadata{
				Point: contribution.Point, ExtensionID: snapshot.ExtensionID,
				ContributionID: contribution.ID, ComponentID: componentID,
				Order: contribution.Order, Label: contribution.Label, Options: payload,
				modulePath: module,
			})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Order != items[j].Order {
			return items[i].Order < items[j].Order
		}
		if items[i].ExtensionID != items[j].ExtensionID {
			return items[i].ExtensionID < items[j].ExtensionID
		}
		return items[i].ContributionID < items[j].ContributionID
	})
	if err := os.MkdirAll(root, 0o755); err != nil {
		return RegistryResult{}, err
	}
	metadataPath := filepath.Join(root, "metadata.ts")
	registryPath := filepath.Join(root, "registry.client.ts")
	if err := writeMetadata(metadataPath, input.ReleaseID, items, localeMessages); err != nil {
		return RegistryResult{}, err
	}
	if err := writeRegistry(registryPath, root, items); err != nil {
		return RegistryResult{}, err
	}
	return RegistryResult{MetadataPath: metadataPath, RegistryPath: registryPath}, nil
}

func writeMetadata(target string, releaseID int64, items []registryMetadata, locales map[string]map[string]json.RawMessage) error {
	publicItems := make([]registryMetadata, len(items))
	copy(publicItems, items)
	for index := range publicItems {
		publicItems[index].modulePath = ""
		if publicItems[index].Label == nil {
			publicItems[index].Label = map[string]string{}
		}
		if publicItems[index].Options == nil {
			publicItems[index].Options = map[string]any{}
		}
	}
	itemBody, err := json.Marshal(publicItems)
	if err != nil {
		return err
	}
	localeBody, err := json.Marshal(locales)
	if err != nil {
		return err
	}
	body := "import type { AdminComponentMetadata, AdminExtensionLocaleMessages } from '~/runtime/admin-extensions/types'\n\n" +
		"export const releaseId = " + strconv.Quote(strconv.FormatInt(releaseID, 10)) + "\n" +
		"export const contributions: readonly AdminComponentMetadata[] = " + string(itemBody) + "\n" +
		"export const locales: AdminExtensionLocaleMessages = " + string(localeBody) + "\n"
	return os.WriteFile(target, []byte(body), 0o644)
}

func writeRegistry(target string, registryRoot string, items []registryMetadata) error {
	var body strings.Builder
	body.WriteString("import type { AdminComponentRegistry } from '~/runtime/admin-extensions/types'\n\nexport const registry: AdminComponentRegistry = {\n")
	for _, item := range items {
		relative, err := filepath.Rel(registryRoot, item.modulePath)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if !strings.HasPrefix(relative, ".") {
			relative = "./" + relative
		}
		body.WriteString("  ")
		body.WriteString(strconv.Quote(item.ExtensionID + ":" + item.ContributionID))
		body.WriteString(": () => import(")
		body.WriteString(strconv.Quote(relative))
		body.WriteString("),\n")
	}
	body.WriteString("}\n")
	return os.WriteFile(target, []byte(body.String()), 0o644)
}

func secureChildDirectory(root string, relative string) (string, error) {
	return secureChildPath(root, relative, true)
}

func secureChildFile(root string, relative string) ([]byte, error) {
	target, err := secureChildPath(root, relative, false)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(target)
}

func secureChildPath(root string, relative string, directory bool) (string, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(relative)))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe relative path %q", relative)
	}
	target := filepath.Join(absoluteRoot, clean)
	if !pathWithin(absoluteRoot, target) {
		return "", fmt.Errorf("path escapes root: %q", relative)
	}
	realRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return "", err
	}
	realTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", err
	}
	realRelative, relativeErr := filepath.Rel(realRoot, realTarget)
	if relativeErr != nil || !pathWithin(realRoot, realTarget) || filepath.Clean(realRelative) != clean {
		return "", fmt.Errorf("symbolic links are not allowed: %q", relative)
	}
	info, err := os.Lstat(target)
	if err != nil {
		return "", err
	}
	if info.Mode()&fs.ModeSymlink != 0 || (directory && !info.IsDir()) || (!directory && !info.Mode().IsRegular()) {
		return "", fmt.Errorf("path is not an allowed %s", map[bool]string{true: "directory", false: "file"}[directory])
	}
	return target, nil
}

func pathWithin(root string, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

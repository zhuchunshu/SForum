package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

type scaffoldManifest struct {
	ID            string                         `json:"id"`
	Name          string                         `json:"name"`
	Description   string                         `json:"description"`
	URL           string                         `json:"url"`
	Author        extensions.ManifestAuthor      `json:"author"`
	Version       string                         `json:"version"`
	Type          string                         `json:"type"`
	SForumVersion string                         `json:"sforumVersion"`
	Permissions   []string                       `json:"permissions,omitempty"`
	Settings      []extensions.ManifestSetting   `json:"settings,omitempty"`
	Backend       *extensions.ManifestBackend    `json:"backend,omitempty"`
	Frontend      *extensions.ManifestFrontend   `json:"frontend,omitempty"`
	AdminPages    []extensions.ManifestAdminPage `json:"adminPages,omitempty"`
}

func GenerateExtensionScaffold(opts makeOptions) (string, error) {
	opts = normalizeMakeOptions(opts)
	if err := validateMakeOptions(opts); err != nil {
		return "", err
	}
	target, err := resolveOutputDir(opts)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(target, extensions.ManifestFileName)); err == nil {
		return "", fmt.Errorf("extension scaffold already exists at %s", target)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", err
	}

	manifest := buildManifest(opts)
	if err := validateGeneratedManifest(manifest); err != nil {
		return "", err
	}
	if err := writeJSON(filepath.Join(target, extensions.ManifestFileName), manifest); err != nil {
		return "", err
	}
	if err := writeFile(filepath.Join(target, "README.md"), readmeBody(opts), 0o644); err != nil {
		return "", err
	}
	if opts.Kind == extensions.TypePlugin {
		return target, writePluginFiles(target, opts)
	}
	return target, writeThemeFiles(target, opts)
}

func normalizeMakeOptions(opts makeOptions) makeOptions {
	opts.Kind = strings.ToLower(strings.TrimSpace(opts.Kind))
	opts.ID = strings.ToLower(strings.TrimSpace(opts.ID))
	opts.Name = strings.TrimSpace(opts.Name)
	opts.Description = strings.TrimSpace(opts.Description)
	opts.URL = strings.TrimSpace(opts.URL)
	opts.AuthorName = strings.TrimSpace(opts.AuthorName)
	opts.AuthorURL = strings.TrimSpace(opts.AuthorURL)
	opts.AuthorEmail = strings.TrimSpace(opts.AuthorEmail)
	opts.Out = strings.TrimSpace(opts.Out)
	return opts
}

func validateMakeOptions(opts makeOptions) error {
	if opts.Kind != extensions.TypePlugin && opts.Kind != extensions.TypeTheme {
		return errors.New("kind must be plugin or theme")
	}
	if opts.ID == "" || opts.Name == "" || opts.Description == "" || opts.URL == "" || opts.AuthorName == "" {
		return errors.New("id, name, description, url, and author-name are required")
	}
	return nil
}

func buildManifest(opts makeOptions) scaffoldManifest {
	manifest := scaffoldManifest{
		ID:            opts.ID,
		Name:          opts.Name,
		Description:   opts.Description,
		URL:           opts.URL,
		Author:        extensions.ManifestAuthor{Name: opts.AuthorName, URL: opts.AuthorURL, Email: opts.AuthorEmail},
		Version:       "0.1.0",
		Type:          opts.Kind,
		SForumVersion: "^1.0.0",
		Settings: []extensions.ManifestSetting{{
			Key:         opts.ID + ".enabled",
			Label:       "Enabled",
			Description: "Enable this extension's recommended behavior.",
			Type:        "boolean",
			Default:     "true",
		}},
		AdminPages: []extensions.ManifestAdminPage{{
			Path:        "/settings",
			Label:       "Settings",
			Description: "Configure this extension.",
			Icon:        "i-lucide-settings",
			View:        "settings",
			Order:       100,
		}},
	}
	if opts.Kind == extensions.TypePlugin {
		manifest.Permissions = []string{opts.ID + ".manage"}
		if opts.Backend {
			manifest.Backend = &extensions.ManifestBackend{Entry: "backend/plugin", RPC: "hashicorp-go-plugin", ProtocolVersion: 1}
		}
		return manifest
	}
	manifest.Frontend = &extensions.ManifestFrontend{Layer: "layer"}
	return manifest
}

func validateGeneratedManifest(manifest scaffoldManifest) error {
	body, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	var model extensions.Manifest
	if err := json.Unmarshal(body, &model); err != nil {
		return err
	}
	return extensions.ValidateManifest(model)
}

func resolveOutputDir(opts makeOptions) (string, error) {
	if opts.Out != "" {
		return filepath.Abs(opts.Out)
	}
	root, err := findRepoRoot()
	if err != nil {
		return "", err
	}
	source := "dev"
	if opts.Builtin {
		source = "builtin"
	}
	group := "plugins"
	if opts.Kind == extensions.TypeTheme {
		group = "themes"
	}
	return filepath.Join(root, "extensions", source, group, opts.ID), nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "extensions")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not locate SForum repository root")
		}
		dir = parent
	}
}

func writeJSON(path string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return writeFile(path, string(body), 0o644)
}

func writeFile(path string, body string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), mode)
}

func writePluginFiles(target string, opts makeOptions) error {
	if !opts.Backend {
		return nil
	}
	if err := writeFile(filepath.Join(target, "backend", "README.md"), pluginBackendReadme(opts), 0o644); err != nil {
		return err
	}
	return writeFile(filepath.Join(target, "backend", "plugin"), "#!/usr/bin/env sh\nprintf 'Build the HashiCorp go-plugin server for "+opts.ID+" into this file.\\n'\n", 0o755)
}

func writeThemeFiles(target string, opts makeOptions) error {
	if err := writeFile(filepath.Join(target, "layer", "nuxt.config.ts"), "export default defineNuxtConfig({})\n", 0o644); err != nil {
		return err
	}
	return writeFile(filepath.Join(target, "layer", "README.md"), "# "+opts.Name+" Nuxt Layer\n\nAdd public theme pages, layouts, components, and assets here.\n", 0o644)
}

func readmeBody(opts makeOptions) string {
	return "# " + opts.Name + "\n\n" + opts.Description + "\n\n- ID: `" + opts.ID + "`\n- Type: `" + opts.Kind + "`\n- Website: " + opts.URL + "\n"
}

func pluginBackendReadme(opts makeOptions) string {
	return "# Backend Stub\n\nBuild a HashiCorp go-plugin compatible executable named `plugin` in this directory before enabling `" + opts.ID + "`.\n"
}

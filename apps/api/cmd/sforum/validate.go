package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func newExtensionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extension",
		Short: "Extension package helpers",
	}
	cmd.AddCommand(newExtensionValidateCommand())
	cmd.AddCommand(newExtensionDigestCommand())
	cmd.AddCommand(newExtensionTestCommand())
	cmd.AddCommand(newExtensionDocsCommand())
	cmd.AddCommand(newExtensionRecoveryListCommand())
	cmd.AddCommand(newExtensionRecoveryDisableCommand())
	cmd.AddCommand(newExtensionRecoveryDisableAllCommand())
	cmd.AddCommand(newPluginCommandCommand())
	return cmd
}

func newExtensionValidateCommand() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "validate [path]",
		Short: "Load and validate an extension package (resolves includes)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			abs, err := filepath.Abs(root)
			if err != nil {
				return err
			}
			info, err := os.Stat(abs)
			if err != nil {
				return err
			}
			if !info.IsDir() {
				return fmt.Errorf("%s is not a directory", abs)
			}
			manifest, err := extensionmanifest.LoadPackage(abs)
			if err != nil {
				return fmt.Errorf("invalid package at %s: %w", abs, err)
			}
			if asJSON {
				// 打印合并后的完整 Manifest，便于审查与调试 includes。
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(manifest)
			}
			printValidateSummary(cmd, abs, manifest)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Print merged manifest JSON")
	return cmd
}

func printValidateSummary(cmd *cobra.Command, root string, manifest extensionmanifest.Manifest) {
	cmd.Printf("OK  %s\n", root)
	cmd.Printf("  id:      %s\n", manifest.ID)
	cmd.Printf("  type:    %s\n", manifest.Type)
	cmd.Printf("  version: %s\n", manifest.Version)
	cmd.Printf("  contract: %s\n", extensionmanifest.ManifestContract(manifest))
	cmd.Printf("  name:    %s\n", manifest.Name)

	locales := make([]string, 0, len(manifest.Langs))
	for key := range manifest.Langs {
		locales = append(locales, key)
	}
	sort.Strings(locales)
	if len(locales) == 0 {
		cmd.Printf("  langs:   (none; using root identity defaults)\n")
	} else {
		cmd.Printf("  langs:   %s\n", strings.Join(locales, ", "))
	}

	cmd.Printf("  settings:       %d\n", len(manifest.Settings))
	cmd.Printf("  settings.ui:    mode=%s layout=%s tabs=%d actions=%d\n",
		manifest.SettingsDocument.UI.Mode,
		manifest.SettingsDocument.UI.Layout,
		len(manifest.SettingsDocument.UI.Tabs),
		len(manifest.SettingsDocument.Actions),
	)
	if component := manifest.SettingsDocument.UI.Component; component != nil {
		cmd.Printf("  settings.ui.component: id=%s apiVersion=%d entry=%s\n", component.ID, component.APIVersion, component.Entry)
	}
	cmd.Printf("  contributions:  %d\n", len(manifest.Contributions))
	cmd.Printf("  routes:         %d\n", len(manifest.Routes))
	cmd.Printf("  events:         %d\n", len(manifest.Events))
	cmd.Printf("  providers:      %d\n", len(manifest.Providers))
	cmd.Printf("  permissions:    %d\n", len(manifest.Permissions))
	if extensionmanifest.EffectiveManifestVersion(manifest) == extensionmanifest.ManifestVersionV3 {
		cmd.Printf("  v3.packageFiles: %d\n", len(manifest.PackageFiles))
		cmd.Printf("  v3.registries:   guards=%d schedules=%d components=%d templates=%d assets=%d services=%d commands=%d\n",
			len(manifest.Guards), len(manifest.Schedules), len(manifest.Components), len(manifest.Templates), len(manifest.Assets), len(manifest.Services), len(manifest.Commands))
		cmd.Printf("  v3.platform:     admin=%d queries=%d permissions=%d media=%d navigation=%d regions=%d dependencies=%d\n",
			len(manifest.AdminSurfaces), len(manifest.Queries), len(manifest.PermissionDefinitions), len(manifest.Media), len(manifest.Navigation), len(manifest.Regions), len(manifest.Dependencies))
	}
	if strings.TrimSpace(manifest.Backend.Entry) != "" {
		cmd.Printf("  backend:        %s (%s)\n", manifest.Backend.Entry, manifest.Backend.RPC)
	}
	if strings.TrimSpace(manifest.Admin.Entry) != "" || len(manifest.Admin.Pages) > 0 {
		cmd.Printf("  admin:          entry=%s pages=%d\n", manifest.Admin.Entry, len(manifest.Admin.Pages))
	}
}

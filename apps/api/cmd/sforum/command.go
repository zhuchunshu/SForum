package main

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

type makeOptions struct {
	Kind             string
	ID               string
	Name             string
	Description      string
	URL              string
	AuthorName       string
	AuthorURL        string
	AuthorEmail      string
	Out              string
	Builtin          bool
	NoInteraction    bool
	Backend          bool
	PrebuiltSettings bool
	ProviderSlot     string
	// Complex 生成 multi-file manifest（includes + langs 目录 + settings 分片示例）。
	Complex bool
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sforum",
		Short: "SForum developer console",
	}
	cmd.AddCommand(
		newMakeCommand("plugin"),
		newMakeCommand("theme"),
		newSeedCommand(),
		newSeedPerfCommand(),
		newExtensionCommand(),
		newDevCleanupOrphanPluginsCommand(),
	)
	return cmd
}

func newMakeCommand(kind string) *cobra.Command {
	opts := makeOptions{Kind: kind}
	commandName := "make:" + kind
	cmd := &cobra.Command{
		Use:   commandName,
		Short: fmt.Sprintf("Create an SForum %s scaffold", kind),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !opts.NoInteraction {
				if err := promptMakeOptions(&opts); err != nil {
					return err
				}
			}
			target, err := GenerateExtensionScaffold(opts)
			if err != nil {
				return err
			}
			cmd.Printf("Created %s scaffold at %s\n", kind, target)
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.ID, "id", "", "Extension id, such as acme.demo")
	cmd.Flags().StringVar(&opts.Name, "name", "", "Display name")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Short description")
	cmd.Flags().StringVar(&opts.URL, "url", "", "Official website URL")
	cmd.Flags().StringVar(&opts.AuthorName, "author-name", "", "Author name")
	cmd.Flags().StringVar(&opts.AuthorURL, "author-url", "", "Author website URL")
	cmd.Flags().StringVar(&opts.AuthorEmail, "author-email", "", "Author email")
	cmd.Flags().StringVar(&opts.Out, "out", "", "Output directory for this package")
	cmd.Flags().BoolVar(&opts.Builtin, "builtin", false, "Create under extensions/builtin instead of extensions/dev")
	cmd.Flags().BoolVar(&opts.NoInteraction, "no-interaction", false, "Disable interactive prompts")
	if kind == "plugin" {
		cmd.Flags().BoolVar(&opts.Backend, "backend", false, "Include a backend plugin stub")
		cmd.Flags().BoolVar(&opts.Complex, "complex", false, "Scaffold multi-file manifest (includes + langs + settings shards)")
		cmd.Flags().BoolVar(&opts.PrebuiltSettings, "prebuilt-settings", false, "Include an author-prebuilt Admin Micro-frontend API v1 settings component with Schema fallback")
		cmd.Flags().StringVar(&opts.ProviderSlot, "provider-slot", "", "Declare a provider slot and host-rendered provider_probe settings action (requires --backend)")
	} else {
		cmd.Flags().BoolVar(&opts.PrebuiltSettings, "prebuilt-settings", false, "Include an author-prebuilt Admin Micro-frontend API v1 settings component with Schema fallback")
	}
	return cmd
}

func promptMakeOptions(opts *makeOptions) error {
	if opts == nil {
		return errors.New("missing make options")
	}
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Extension ID").Description("Use a stable id such as acme.demo.").Value(&opts.ID),
			huh.NewInput().Title("Name").Value(&opts.Name),
			huh.NewInput().Title("Description").Value(&opts.Description),
			huh.NewInput().Title("Official URL").Value(&opts.URL),
			huh.NewInput().Title("Author name").Value(&opts.AuthorName),
			huh.NewInput().Title("Author URL").Value(&opts.AuthorURL),
			huh.NewInput().Title("Author email").Value(&opts.AuthorEmail),
		),
	)
	if opts.Kind == "plugin" {
		form = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Extension ID").Description("Use a stable id such as acme.demo.").Value(&opts.ID),
				huh.NewInput().Title("Name").Value(&opts.Name),
				huh.NewInput().Title("Description").Value(&opts.Description),
				huh.NewInput().Title("Official URL").Value(&opts.URL),
				huh.NewInput().Title("Author name").Value(&opts.AuthorName),
				huh.NewInput().Title("Author URL").Value(&opts.AuthorURL),
				huh.NewInput().Title("Author email").Value(&opts.AuthorEmail),
				huh.NewConfirm().Title("Include backend stub?").Value(&opts.Backend),
				huh.NewConfirm().Title("Multi-file complex scaffold?").Description("Uses includes, per-locale langs, and settings shards.").Value(&opts.Complex),
				huh.NewConfirm().Title("Prebuilt settings component?").Description("Author-built .mjs with Schema fallback; operators do not rebuild SForum.").Value(&opts.PrebuiltSettings),
				huh.NewInput().Title("Provider slot (optional)").Description("Adds a provider_probe action; requires backend stub.").Value(&opts.ProviderSlot),
			),
		)
	}
	return form.Run()
}

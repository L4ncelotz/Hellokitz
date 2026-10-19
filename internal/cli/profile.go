package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/L4ncelotz/Hellokitz/internal/profile"
)

// userDirOverride pins the user profile directory in tests. Production code
// resolves it from the OS user config directory.
var userDirOverride string

func defaultStore() profile.Store {
	dir := userDirOverride
	if dir == "" {
		if cfg, err := os.UserConfigDir(); err == nil {
			dir = filepath.Join(cfg, "hellokit", "profiles")
		}
	}
	return profile.Store{UserDir: dir, Builtins: profile.DefaultBuiltins()}
}

func newProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Inspect HelloKit profiles without changing the machine",
	}
	cmd.AddCommand(newProfileValidateCmd())
	cmd.AddCommand(newProfileListCmd())
	cmd.AddCommand(newProfileShowCmd())
	return cmd
}

func newProfileValidateCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a local profile file against schema v1",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := profile.LoadFile(file)
			if err != nil {
				return err
			}
			s := p.Summarize()
			_, err = fmt.Fprintf(cmd.OutOrStdout(),
				"Profile %q is valid: schema %d, os %s, arch %v, %d application(s), %d runtime(s), %d extension(s), %d setting(s).\n",
				p.Name, p.SchemaVersion, p.Platform.OS, p.Platform.Architectures,
				s.Applications, s.Runtimes, s.Extensions, s.Settings)
			return err
		},
	}

	cmd.Flags().StringVar(&file, "file", "", "Path to a local profile YAML file")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func newProfileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available local and bundled profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			entries, err := defaultStore().List()
			if err != nil {
				return err
			}
			shadowed := shadowedBuiltins(entries)
			claimants := userClaimants(entries)
			out := cmd.OutOrStdout()
			if len(entries) == 0 {
				_, err := fmt.Fprintln(out, "No profiles found.")
				return err
			}
			w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSOURCE\tSTATUS")
			for _, e := range entries {
				fmt.Fprintf(w, "%s\t%s\t%s\n", e.DisplayName(), e.Source, entryStatus(e, shadowed, claimants))
			}
			return w.Flush()
		},
	}
}

// shadowedBuiltins reports builtin names (case-insensitive) that a
// user-owned profile shadows.
func shadowedBuiltins(entries []profile.Entry) map[string]bool {
	local := make(map[string]bool)
	for _, e := range entries {
		if e.Source == profile.SourceLocal && e.Valid() {
			local[strings.ToLower(e.Name)] = true
		}
	}
	shadowed := make(map[string]bool)
	for _, e := range entries {
		if e.Source == profile.SourceBuiltin && local[strings.ToLower(e.Name)] {
			shadowed[strings.ToLower(e.Name)] = true
		}
	}
	return shadowed
}

// userClaimants maps case-insensitive profile names to their claiming files.
func userClaimants(entries []profile.Entry) map[string][]string {
	m := make(map[string][]string)
	for _, e := range entries {
		if e.Source == profile.SourceLocal && e.Valid() {
			k := strings.ToLower(e.Name)
			m[k] = append(m[k], e.File)
		}
	}
	return m
}

func entryStatus(e profile.Entry, shadowed map[string]bool, claimants map[string][]string) string {
	if !e.Valid() {
		first := strings.SplitN(e.Err.Error(), "\n", 2)[0]
		return "invalid: " + first
	}
	if e.Collision {
		var others []string
		for _, f := range claimants[strings.ToLower(e.Name)] {
			if f != e.File {
				others = append(others, f)
			}
		}
		return "collision: also claimed by " + strings.Join(others, ", ")
	}
	if e.Source == profile.SourceBuiltin && shadowed[strings.ToLower(e.Name)] {
		return "ok (shadowed by local profile)"
	}
	return "ok"
}

func newProfileShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Summarize a profile without changing the machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, e, err := defaultStore().Resolve(args[0])
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Name: %s\nSource: %s\n", p.Name, e.Source)
			if e.Source == profile.SourceLocal {
				fmt.Fprintf(out, "Path: %s\n", e.Path)
			}
			fmt.Fprintf(out, "Schema: %d\n", p.SchemaVersion)
			if p.Description != "" {
				fmt.Fprintf(out, "Description: %s\n", p.Description)
			}
			fmt.Fprintf(out, "Platform: %s (%s)\n", p.Platform.OS, strings.Join(p.Platform.Architectures, ", "))
			fmt.Fprintf(out, "Applications (%d):\n", len(p.Applications))
			for _, a := range p.Applications {
				if a.Version != "" {
					fmt.Fprintf(out, "  - %s (source: %s, version: %s)\n", a.ID, a.Source, a.Version)
				} else {
					fmt.Fprintf(out, "  - %s (source: %s)\n", a.ID, a.Source)
				}
			}
			fmt.Fprintf(out, "Runtimes (%d):\n", len(p.Runtimes))
			for _, r := range p.Runtimes {
				fmt.Fprintf(out, "  - %s %s (manager: %s)\n", r.Name, r.Version, r.Manager)
			}
			fmt.Fprintf(out, "Editor extensions (%d):\n", len(p.EditorExtensions))
			for _, x := range p.EditorExtensions {
				fmt.Fprintf(out, "  - %s: %s\n", x.Editor, x.ID)
			}
			fmt.Fprintf(out, "Settings (%d):\n", len(p.Settings))
			for _, s := range p.Settings {
				fmt.Fprintf(out, "  - %s %s = %s\n", s.Kind, s.Key, s.Value)
			}
			return nil
		},
	}
}

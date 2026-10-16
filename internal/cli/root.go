// Package cli defines the HelloKit command hierarchy.
package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCmd builds the hellokit root command with persistent flags and
// all subcommands attached. It is a constructor (no global state) so tests
// can instantiate isolated command trees.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "hellokit",
		Short: "One command. Your dev environment.",
		Long:  "One command. Your dev environment.\n\nHelloKit is an offline-first CLI for recreating a chosen developer environment on Windows.",
		// Do not dump full usage text on runtime errors; exit code + error
		// message is sufficient and keeps output parseable.
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().Bool("no-color", false, "Disable color output")
	root.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose diagnostic logging")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newProfileCmd())

	return root
}

// Execute runs the default command tree used by main.
func Execute() error {
	return NewRootCmd().Execute()
}

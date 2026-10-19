package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/L4ncelotz/Hellokitz/internal/plan"
	"github.com/L4ncelotz/Hellokitz/internal/profile"
)

func newPlanCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Preview the changes a profile implies without making any",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := profile.LoadFile(file)
			if err != nil {
				return err
			}
			// M1 has no detection adapters; the unknown inventory claims
			// nothing about the machine. Real states land with M2 adapters.
			printPlanReport(cmd.OutOrStdout(), plan.Build(p, plan.UnknownInventory{}))
			return nil
		},
	}

	cmd.Flags().StringVar(&file, "file", "", "Path to a local profile YAML file")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func printPlanReport(out io.Writer, pl plan.Plan) {
	fmt.Fprintf(out, "Plan for profile %q (%s):\n", pl.ProfileName, pl.Platform)
	for _, a := range pl.Actions {
		desired := ""
		if a.Desired != "" {
			desired = ", desired: " + a.Desired
		}
		fmt.Fprintf(out, "  [%s] %s (source: %s%s) — %s: %s\n",
			planActionLabel(a.State), a.Target, a.Source, desired, planStateWord(a.State), a.Reason)
	}
	fmt.Fprintln(out, "No changes have been made.")
}

func planActionLabel(state string) string {
	switch state {
	case plan.StateSatisfied:
		return "SKIP"
	case plan.StateMissing:
		return "INSTALL"
	case plan.StateMismatch:
		return "MODIFY"
	case plan.StateBlocked:
		return "BLOCKED"
	default:
		return "UNKNOWN"
	}
}

func planStateWord(state string) string {
	switch state {
	case plan.StateSatisfied:
		return "Already satisfied"
	case plan.StateMissing:
		return "Will install"
	case plan.StateMismatch:
		return "Needs attention"
	case plan.StateBlocked:
		return "Blocked"
	default:
		return "Cannot verify"
	}
}

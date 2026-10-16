package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"hellokit/internal/platform"
)

// doctorRunner executes capability probes. Tests replace it with a fake;
// production uses direct execution without a shell.
var doctorRunner platform.CommandRunner = platform.OSRunner{}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check host and tool readiness without changing the machine",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Diagnosis is infallible by design: every failure mode becomes a
			// check state, so a completed diagnosis always exits 0.
			report := platform.Diagnose(cmd.Context(), platform.Host(), doctorRunner)
			printDoctorReport(cmd.OutOrStdout(), report)
			return nil
		},
	}
}

func printDoctorReport(w io.Writer, r platform.Report) {
	fmt.Fprintf(w, "Host: %s\n", r.Host)
	for _, c := range r.Checks {
		fmt.Fprintf(w, "[%s] %s: %s\n", doctorStateLabel(c.State), c.Name, c.Detail)
	}
}

func doctorStateLabel(state string) string {
	switch state {
	case platform.StateOK:
		return "OK"
	case platform.StateMissing:
		return "MISSING"
	case platform.StateUnknown:
		return "UNKNOWN"
	case platform.StateUnsupported:
		return "UNSUPPORTED"
	default:
		return strings.ToUpper(state)
	}
}

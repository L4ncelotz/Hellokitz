package main

import (
	"fmt"
	"os"

	"hellokit/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		// Cobra silences its own error output (SilenceErrors); report once here.
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

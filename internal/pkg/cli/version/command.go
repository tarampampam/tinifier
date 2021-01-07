// Package version contains CLI `version` command implementation.
package version

import (
	"fmt"
	"os"
	"runtime"

	"github.com/tarampampam/tinifier/v3/internal/pkg/version"

	"github.com/spf13/cobra"
)

// NewCommand creates `version` command.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Display application version",
		Run: func(*cobra.Command, []string) {
			_, _ = fmt.Fprintf(os.Stdout, "app version:\t%s (%s)\n", version.Version(), runtime.Version())
		},
	}
}

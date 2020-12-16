package version

import (
	"fmt"
	"os"

	"github.com/tarampampam/tinifier/internal/pkg/version"

	"github.com/spf13/cobra"
)

// NewCommand creates `version` command.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Display application version",
		Run: func(*cobra.Command, []string) {
			_, _ = fmt.Fprintf(os.Stdout, "version: %s\n", version.Version())
		},
	}
}

// Package cli contains CLI command handlers.
package cli

import (
	"github.com/tarampampam/tinifier/v3/internal/pkg/cli/compress"
	"github.com/tarampampam/tinifier/v3/internal/pkg/cli/quota"
	"github.com/tarampampam/tinifier/v3/internal/pkg/cli/version"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NewCommand creates root command.
func NewCommand(log *zap.Logger, atomicLogLevel *zap.AtomicLevel, appName string) *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use: appName,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			if verbose {
				atomicLogLevel.SetLevel(zap.DebugLevel)
			}

			return nil
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	cmd.AddCommand(
		compress.NewCommand(log),
		quota.NewCommand(log),
		version.NewCommand(),
	)

	return cmd
}

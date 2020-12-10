package cli

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"tinifier/internal/pkg/cli/compress"
	"tinifier/internal/pkg/cli/quota"
	"tinifier/internal/pkg/cli/version"
)

// NewCommand creates root command.
func NewCommand(log *logrus.Logger, appName string) *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use: appName,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			if verbose {
				log.SetLevel(logrus.DebugLevel)
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.AddCommand(
		compress.NewCommand(log),
		quota.NewCommand(log),
		version.NewCommand(log),
	)

	return cmd
}

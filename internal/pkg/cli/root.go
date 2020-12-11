package cli

import (
	"github.com/tarampampam/tinifier/internal/pkg/cli/compress"
	"github.com/tarampampam/tinifier/internal/pkg/cli/quota"
	"github.com/tarampampam/tinifier/internal/pkg/cli/version"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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

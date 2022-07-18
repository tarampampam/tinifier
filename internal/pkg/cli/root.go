// Package cli contains CLI command handlers.
package cli

import (
	"github.com/spf13/cobra"
	"github.com/tarampampam/tinifier/v4/internal/pkg/cli/compress"
	"github.com/tarampampam/tinifier/v4/internal/pkg/cli/quota"
	versionCmd "github.com/tarampampam/tinifier/v4/internal/pkg/cli/version"
	"github.com/tarampampam/tinifier/v4/internal/pkg/logger"
	"github.com/tarampampam/tinifier/v4/internal/pkg/version"
)

// NewCommand creates root command.
func NewCommand(appName string) *cobra.Command {
	var (
		verbose bool
		debug   bool
	)

	// create "default" logger (will be overwritten later with customized)
	log, err := logger.New(false, false)
	if err != nil {
		panic(err) // will never occurs
	}

	cmd := &cobra.Command{
		Use: appName,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			_ = log.Sync() // sync previous logger instance

			customizedLog, e := logger.New(verbose, debug)
			if e != nil {
				return e
			}

			*log = *customizedLog // override "default" logger with customized

			return nil
		},
		PersistentPostRun: func(*cobra.Command, []string) {
			// error ignoring reasons:
			// - <https://github.com/uber-go/zap/issues/772>
			// - <https://github.com/uber-go/zap/issues/328>
			_ = log.Sync()
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	cmd.PersistentFlags().BoolVarP(&debug, "debug", "", false, "debug output")

	cmd.AddCommand(
		compress.NewCommand(log),
		quota.NewCommand(log),
		versionCmd.NewCommand(version.Version()),
	)

	return cmd
}

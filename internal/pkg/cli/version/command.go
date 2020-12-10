package version

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"tinifier/internal/pkg/version"
)

// NewCommand creates `version` command.
func NewCommand(log *logrus.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Display application version",
		Run: func(*cobra.Command, []string) {
			log.Info(version.Version())
		},
	}
}

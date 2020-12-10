package compress

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewCommand creates `compress` command.
func NewCommand(log *logrus.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "compress",
		Aliases: []string{"c"},
		Short:   "Compress images",
		RunE: func(*cobra.Command, []string) error {
			return nil
		},
	}
}

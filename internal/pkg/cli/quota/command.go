package quota

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewCommand creates `quota` command.
func NewCommand(log *logrus.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "quota",
		Aliases: []string{"q"},
		Short:   "Get currently used quota",
		RunE: func(*cobra.Command, []string) error {
			return nil
		},
	}
}

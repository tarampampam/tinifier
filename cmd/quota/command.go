package quota

import (
	"time"
	"tinifier/cmd/shared"
	"tinifier/tinypng"

	log "github.com/sirupsen/logrus"
)

type Command struct {
	shared.WithAPIKey
}

// Follows `flags.Commander` interface (required for commands handling).
func (*Command) Execute(_ []string) error { return nil }

// Handle `serve` command.
func (cmd *Command) Handle(log *log.Logger, _ []string) error {
	client := tinypng.NewClient(cmd.APIKey.String(), time.Second*5)

	count, err := client.GetCompressionCount()
	if err != nil {
		return err
	}

	log.
		WithField("key", cmd.APIKey.Masked()).
		WithField("quota", count).
		Info("Currently used quota")

	return nil
}

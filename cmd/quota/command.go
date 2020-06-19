package quota

import (
	"context"
	"fmt"
	"time"
	"tinifier/cmd/shared"
	"tinifier/tinypng"
)

const tinypngRequestTimeout time.Duration = time.Second * 5

// Command is a `quota` command.
type Command struct {
	shared.WithAPIKey
}

// Execute command.
func (cmd *Command) Execute(_ []string) error {
	client := tinypng.NewClient(cmd.APIKey.String(), tinypngRequestTimeout)

	count, err := client.GetCompressionCount(context.Background())
	if err != nil {
		return err
	}

	fmt.Printf("Currently used quota for key [%s] is %d\n", cmd.APIKey.Masked(), count)

	return nil
}

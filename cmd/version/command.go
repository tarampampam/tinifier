package version

import (
	"fmt"
	ver "tinifier/version"
)

// Command is a `version` command.
type Command struct{}

// Execute current command.
func (*Command) Execute(_ []string) (err error) {
	_, err = fmt.Printf("Version: %s\n", ver.Version())

	return
}

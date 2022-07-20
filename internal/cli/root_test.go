package cli_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tarampampam/tinifier/v4/internal/cli"
)

func TestNewApp(t *testing.T) {
	app := cli.NewApp()

	require.NotEmpty(t, app.Commands)
	require.NotEmpty(t, app.Flags)

	flagNames := make([]string, 0, len(app.Flags))

	for i := 0; i < len(app.Flags); i++ {
		flagNames = append(flagNames, app.Flags[i].Names()...)
	}

	assert.Contains(t, flagNames, "log-level")
}

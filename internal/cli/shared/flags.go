package shared

import (
	"github.com/urfave/cli/v2"

	"github.com/tarampampam/tinifier/v4/internal/env"
)

var APIKeyFlag = &cli.StringSliceFlag{
	Name:    "api-key",
	Aliases: []string{"k"},
	Usage:   "TinyPNG API key <https://tinypng.com/dashboard/api>",
	EnvVars: []string{env.TinyPngAPIKey.String()},
}

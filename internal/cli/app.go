// Package cli contains CLI command handlers.
package cli

import (
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/urfave/cli/v2"

	"github.com/tarampampam/tinifier/v4/internal/cli/compress"
	"github.com/tarampampam/tinifier/v4/internal/cli/quota"
	"github.com/tarampampam/tinifier/v4/internal/env"
	"github.com/tarampampam/tinifier/v4/internal/version"
)

// NewApp creates new console application.
func NewApp() *cli.App {
	return &cli.App{
		Usage: "CLI client for images compressing using tinypng.com API",
		Before: func(context *cli.Context) error {
			if _, exists := env.ForceColors.Lookup(); exists {
				text.EnableColors()
			} else if _, exists = env.NoColors.Lookup(); exists {
				text.DisableColors()
			} else if v, ok := env.Term.Lookup(); ok && v == "dumb" {
				text.DisableColors()
			}

			return nil
		},
		Commands: []*cli.Command{
			quota.NewCommand(),
			compress.NewCommand(),
		},
		Version: version.Version(),
	}
}

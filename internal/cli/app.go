// Package cli contains CLI command handlers.
package cli

import (
	"github.com/urfave/cli/v2"

	"github.com/tarampampam/tinifier/v4/internal/cli/compress"
	"github.com/tarampampam/tinifier/v4/internal/cli/quota"
	"github.com/tarampampam/tinifier/v4/internal/version"
)

// NewApp creates new console application.
func NewApp() *cli.App {
	return &cli.App{
		Usage: "CLI client for images compressing using tinypng.com API",
		Commands: []*cli.Command{
			quota.NewCommand(),
			compress.NewCommand(),
		},
		Version: version.Version(),
	}
}

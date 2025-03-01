package cli

import (
	"context"
	"errors"

	"gh.tarampamp.am/tinifier/v5/internal/cli/cmd"
	"gh.tarampamp.am/tinifier/v5/internal/version"
)

//go:generate go run ./generate/readme.go

type App struct {
	cmd cmd.Command
	opt struct{}
}

func NewApp(name string) *App {
	var app = App{
		cmd: cmd.Command{
			Name:        name,
			Description: "CLI client for images compressing using tinypng.com API.",
			Usage:       "[<options>] [<files-or-directories>]",
			Version:     version.Version(),
		},
	}

	app.cmd.Action = func(ctx context.Context, c *cmd.Command, args []string) error {
		return app.run(ctx)
	}

	return &app
}

// Run runs the application.
func (a *App) Run(ctx context.Context, args []string) error { return a.cmd.Run(ctx, args) }

// Help returns the help message.
func (a *App) Help() string { return a.cmd.Help() }

// run in the main logic of the application.
func (a *App) run(ctx context.Context) error {
	return errors.New("not implemented")
}

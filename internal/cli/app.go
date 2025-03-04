package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"gh.tarampamp.am/tinifier/v5/internal/cli/cmd"
	"gh.tarampamp.am/tinifier/v5/internal/config"
	"gh.tarampamp.am/tinifier/v5/internal/version"
)

//go:generate go run ./generate/readme.go

type App struct {
	cmd cmd.Command
	opt options
}

func NewApp(name string) *App { //nolint:funlen
	var app = App{
		cmd: cmd.Command{
			Name:        name,
			Description: "CLI client for images compressing using tinypng.com API.",
			Usage:       "[<options>] [<files-or-directories>]",
			Version:     version.Version(),
		},
		opt: newOptionsWithDefaults(),
	}

	var (
		configFile = cmd.Flag[string]{
			Names:   []string{"config-file", "c"},
			Usage:   "Path to the configuration file",
			EnvVars: []string{"CONFIG_FILE"},
			Default: filepath.Join(config.DefaultDirPath(), config.FileName),
		}
		apiKeys = cmd.Flag[string]{
			Names:   []string{"api-key", "k"},
			Usage:   "TinyPNG API keys <https://tinypng.com/dashboard/api> (separated by commas)",
			EnvVars: []string{"API_KEYS"},
		}
		fileExtensions = cmd.Flag[string]{
			Names:   []string{"ext", "e"},
			Usage:   "Extensions of files to compress (case insensitive, without leading dots, separated by commas)",
			EnvVars: []string{"FILE_EXTENSIONS"},
			Default: strings.Join(app.opt.FileExtensions, ","),
			Validator: func(c *cmd.Command, v string) error {
				if v == "" {
					return errors.New("extensions list cannot be empty")
				}

				return nil
			},
		}
		threatsCount = cmd.Flag[uint]{
			Names:   []string{"threads", "t"},
			Usage:   "Number of threads to use for compressing",
			EnvVars: []string{"THREADS"},
			Default: app.opt.ThreadsCount,
		}
		maxErrorsToStop = cmd.Flag[uint]{
			Names:   []string{"max-errors"},
			Usage:   "Maximum number of errors to stop the process (set 0 to disable)",
			EnvVars: []string{"MAX_ERRORS"},
			Default: app.opt.MaxErrorsToStop,
		}
		recursive = cmd.Flag[bool]{
			Names:   []string{"recursive", "r"},
			Usage:   "Search for files in listed directories recursively",
			EnvVars: []string{"RECURSIVE"},
			Default: app.opt.Recursive,
		}
		updateFileModDate = cmd.Flag[bool]{
			Names:   []string{"update-mod-date"},
			Usage:   "Update the modification date of the compressed files (otherwise, the original date will be preserved)",
			EnvVars: []string{"UPDATE_MOD_DATE"},
			Default: app.opt.UpdateFileModDate,
		}
	)

	app.cmd.Flags = []cmd.Flagger{
		&configFile,
		&apiKeys,
		&fileExtensions,
		&threatsCount,
		&maxErrorsToStop,
		&recursive,
		&updateFileModDate,
	}

	app.cmd.Action = func(ctx context.Context, c *cmd.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("no files or directories specified")
		}

		if err := app.opt.UpdateFromConfigFile(*configFile.Value); err != nil {
			return err
		}

		{ // override the options with the command-line flags
			if apiKeys.IsSet() && apiKeys.Value != nil {
				if clean := cleanStrings(*apiKeys.Value, ","); len(clean) > 0 {
					app.opt.ApiKeys = clean
				}
			}

			if fileExtensions.IsSet() && fileExtensions.Value != nil {
				if clean := cleanStrings(*fileExtensions.Value, ","); len(clean) > 0 {
					app.opt.FileExtensions = clean
				}
			}

			setIfFlagIsSet(&app.opt.ThreadsCount, threatsCount)
			setIfFlagIsSet(&app.opt.MaxErrorsToStop, maxErrorsToStop)
			setIfFlagIsSet(&app.opt.Recursive, recursive)
			setIfFlagIsSet(&app.opt.UpdateFileModDate, updateFileModDate)
		}

		if err := app.opt.Validate(); err != nil {
			return fmt.Errorf("invalid options: %w", err)
		}

		return app.run(ctx, args)
	}

	return &app
}

// setIfFlagIsSet sets the value from the flag to the option if the flag is set and the value is not nil.
func setIfFlagIsSet[T cmd.FlagType](target *T, source cmd.Flag[T]) {
	if target == nil || source.Value == nil || !source.IsSet() {
		return
	}

	*target = *source.Value
}

// cleanStrings splits the input string by the separator and removes empty strings and spaces.
func cleanStrings(in, sep string) []string {
	var out = strings.Split(in, sep)

	// remove spaces and empty strings
	for i := 0; i < len(out); i++ {
		out[i] = strings.TrimSpace(out[i])

		if out[i] == "" {
			out = append(out[:i], out[i+1:]...)
			i--
		}
	}

	return out
}

// Run runs the application.
func (a *App) Run(ctx context.Context, args []string) error { return a.cmd.Run(ctx, args) }

// Help returns the help message.
func (a *App) Help() string { return a.cmd.Help() }

// run in the main logic of the application.
func (a *App) run(ctx context.Context, paths []string) error {
	fmt.Println(paths)

	return errors.New("not implemented")
}

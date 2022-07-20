package compress

import (
	"context"
	"errors"
	"os"
	"runtime"
	"sort"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/tarampampam/tinifier/v4/internal/breaker"
	"github.com/tarampampam/tinifier/v4/internal/env"
	appFs "github.com/tarampampam/tinifier/v4/internal/fs"
	"github.com/tarampampam/tinifier/v4/internal/logger"
)

type command struct {
	log *logger.Logger
	c   *cli.Command
}

// NewCommand creates `compress` command.
func NewCommand(log *logger.Logger) *cli.Command {
	const (
		apiKeyFlagName          = "api-key"
		fileExtensionsFlagName  = "ext"
		threadsCountFlagName    = "threads"
		maxErrorsToStopFlagName = "max-errors"
		recursiveFlagName       = "recursive"
	)

	var cmd = command{log: log}

	cmd.c = &cli.Command{
		Name:      "compress",
		ArgsUsage: "<target-files-and-directories...>",
		Aliases:   []string{"c"},
		Usage:     "Compress images",
		Action: func(c *cli.Context) error {
			var (
				apiKeys         = c.StringSlice(apiKeyFlagName)
				fileExtensions  = c.StringSlice(fileExtensionsFlagName)
				threadsCount    = c.Uint(threadsCountFlagName)
				maxErrorsToStop = c.Uint(maxErrorsToStopFlagName)
				recursive       = c.Bool(recursiveFlagName)
				paths           = c.Args().Slice()
			)

			log.Debug("Run args",
				zap.Strings("apiKeys", apiKeys),
				zap.Strings("fileExtensions", fileExtensions),
				zap.Uint("threadsCount", threadsCount),
				zap.Uint("maxErrorsToStop", maxErrorsToStop),
				zap.Bool("recursive", recursive),
				zap.Strings("args", paths),
			)

			if len(paths) == 0 {
				return errors.New("no files or directories specified")
			}

			return cmd.Run(c.Context, paths, fileExtensions, recursive)
		},
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    apiKeyFlagName,
				Aliases: []string{"k"},
				Usage:   "TinyPNG API key <https://tinypng.com/dashboard/api>",
				EnvVars: []string{env.TinyPngAPIKey.String()},
			},
			&cli.StringSliceFlag{
				Name:    fileExtensionsFlagName,
				Aliases: []string{"e"},
				Usage:   "image file extensions (without leading dots)",
				Value:   cli.NewStringSlice("jpg", "JPG", "jpeg", "JPEG", "png", "PNG"),
				EnvVars: []string{}, // TODO implement
			},
			&cli.UintFlag{
				Name:    threadsCountFlagName,
				Aliases: []string{"t"},
				Usage:   "threads count",
				Value:   uint(runtime.NumCPU() * 8), //nolint:gomnd
				EnvVars: []string{env.ThreadsCount.String()},
			},
			&cli.UintFlag{
				Name:    maxErrorsToStopFlagName,
				Usage:   "maximum errors count to stop the process (set 0 to disable)",
				Value:   10,         //nolint:gomnd
				EnvVars: []string{}, // TODO implement
			},
			&cli.BoolFlag{
				Name:    recursiveFlagName,
				Aliases: []string{"r"},
				Usage:   "search for files in listed directories recursively",
				EnvVars: []string{}, // TODO implement
			},
		},
	}

	return cmd.c
}

// Run current command.
func (cmd *command) Run(pCtx context.Context, paths, fileExt []string, recursive bool) error {
	var (
		ctx, cancel = context.WithCancel(pCtx)  // main context creation
		oss         = breaker.NewOSSignals(ctx) // OS signals listener
	)

	oss.Subscribe(func(sig os.Signal) {
		cmd.log.Debug("Stopping by OS signal..", zap.String("signal", sig.String()))

		cancel()
	})

	defer func() {
		cancel()   // call cancellation function after all for "service" goroutines stopping
		oss.Stop() // stop system signals listening
	}()

	files, findErr := cmd.FindFiles(ctx, paths, fileExt, recursive)
	if findErr != nil {
		if errors.Is(findErr, context.Canceled) {
			return errors.New("images searching was canceled")
		}

		return findErr
	}

	if len(files) == 0 {
		return errors.New("nothing to compress (files not found)")
	}

	// cmd.log.Debug("Found files", zap.Strings("files", files))

	return nil
}

func (*command) FindFiles(ctx context.Context, where, filesExt []string, recursive bool) ([]string, error) {
	if len(where) == 0 || len(filesExt) == 0 { // fast terminator
		return []string{}, nil
	}

	var (
		spin      = spinner.New([]string{" ⣾ ", " ⣽ ", " ⣻ ", " ⢿ ", " ⡿ ", " ⣟ ", " ⣯ ", " ⣷ "}, time.Millisecond*70) //nolint:gomnd,lll
		startedAt = time.Now()
		prefix    = color.New(color.Bold).Sprint("Images searching")
	)

	if !color.NoColor {
		_ = spin.Color("green")
	}

	spin.PreUpdate = func(s *spinner.Spinner) {
		s.Prefix = prefix + " " + time.Since(startedAt).Round(time.Second).String()
	}

	spin.Start()
	defer spin.Stop()

	var found = make([]string, 0, len(where))

	if err := appFs.FindFiles(ctx, where, func(absPath string) {
		spin.Suffix = absPath
		found = append(found, absPath)
	}, appFs.WithRecursive(recursive), appFs.WithFilesExt(filesExt...)); err != nil {
		return nil, err
	}

	sort.Strings(found)

	return found, nil
}

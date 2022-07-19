package compress

import (
	"context"
	"errors"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/tarampampam/tinifier/v4/internal/breaker"
	"github.com/tarampampam/tinifier/v4/internal/env"
)

type command struct {
	log *zap.Logger
	c   *cli.Command
}

// NewCommand creates `compress` command.
func NewCommand(log *zap.Logger) *cli.Command {
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

	return nil
}

func (*command) FindFiles(ctx context.Context, where, filesExt []string, recursive bool) ([]string, error) {
	if len(where) == 0 || len(filesExt) == 0 { // fast terminator
		return []string{}, nil
	}

	for i := 0; i < len(where); i++ { // convert relative paths to absolute
		if !filepath.IsAbs(where[i]) {
			if abs, err := filepath.Abs(where[i]); err != nil {
				return nil, err
			} else {
				where[i] = abs
			}
		}
	}

	where = lo.Uniq[string](where) // remove duplicates
	sort.Strings(where)            // and sort

	var extMap = make(map[string]struct{}, len(filesExt))

	for _, ext := range filesExt { // burn the map
		extMap[ext] = struct{}{}
	}

	spin := spinner.New([]string{" ⣾ ", " ⣽ ", " ⣻ ", " ⢿ ", " ⡿ ", " ⣟ ", " ⣯ ", " ⣷ "}, time.Millisecond*70) //nolint:gomnd,lll
	spin.Prefix = "Images searching"

	if !color.NoColor {
		_ = spin.Color("green")
		spin.Prefix = color.New(color.Bold).Sprint(spin.Prefix)
	}

	spin.Start()
	defer spin.Stop()

	var unique = make(map[string]struct{}, len(where))

	for _, location := range where {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		locationStat, statErr := os.Stat(location)
		if statErr != nil {
			return nil, statErr
		}

		switch mode := locationStat.Mode(); {
		case mode.IsRegular(): // regular file (eg.: `./file.png`)
			spin.Suffix = location
			unique[location] = struct{}{} // ignore file extension checking for the single files

		case mode.IsDir(): // directory (eg.: `./path/to/images`)
			if recursive { //nolint:nestif // deep directory search
				if walkingErr := filepath.Walk(location, func(path string, info fs.FileInfo, err error) error {
					if ctxErr := ctx.Err(); ctxErr != nil {
						return ctxErr
					}

					if err == nil && info.Mode().IsRegular() {
						spin.Suffix = path

						if fileExt := filepath.Ext(info.Name()); len(fileExt) > 0 {
							if _, ok := extMap[fileExt[1:]]; ok {
								unique[path] = struct{}{}
							}
						}
					}

					return err
				}); walkingErr != nil {
					return nil, walkingErr
				}
			} else { // flat directory search
				files, readDirErr := ioutil.ReadDir(location)
				if readDirErr != nil {
					return nil, readDirErr
				}

				for _, file := range files {
					if ctxErr := ctx.Err(); ctxErr != nil {
						return nil, ctxErr
					}

					if file.Mode().IsRegular() {
						var path = filepath.Join(location, file.Name())

						spin.Suffix = path

						if fileExt := filepath.Ext(file.Name()); len(fileExt) > 0 {
							if _, ok := extMap[fileExt[1:]]; ok {
								unique[path] = struct{}{}
							}
						}
					}
				}
			}
		}
	}

	// convert map into slice
	result, i := make([]string, len(unique)), 0
	for path := range unique {
		result[i], i = path, i+1
	}

	sort.Strings(result)

	return result, nil
}

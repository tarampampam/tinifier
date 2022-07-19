package compress

import (
	"errors"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

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

			files, findErr := cmd.FindFiles(paths, fileExtensions, recursive)
			if findErr != nil {
				return findErr
			}

			if len(files) == 0 {
				return errors.New("nothing to compress (files not found)")
			}

			log.Debug("Found files", zap.Int("count", len(files)), zap.Strings("files", files))

			return cmd.Run()
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
				Value:   uint(runtime.NumCPU() * 8),
				EnvVars: []string{env.ThreadsCount.String()},
			},
			&cli.UintFlag{
				Name:    maxErrorsToStopFlagName,
				Usage:   "maximum errors count to stop the process (set 0 to disable)",
				Value:   10,
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

func (*command) FindFiles(where, filesExt []string, recursive bool) ([]string, error) {
	if len(where) == 0 || len(filesExt) == 0 { // fast terminator
		return []string{}, nil
	}

	var extMap = make(map[string]struct{}, len(filesExt))

	for _, ext := range filesExt { // burn the map
		extMap[ext] = struct{}{}
	}

	var unique = make(map[string]struct{}, len(where))

	for _, location := range where {
		locationStat, statErr := os.Stat(location)
		if statErr != nil {
			return nil, statErr
		}

		switch mode := locationStat.Mode(); {
		case mode.IsRegular():
			if absPath, absErr := filepath.Abs(location); absErr != nil {
				return nil, absErr
			} else if fileExt := filepath.Ext(locationStat.Name()); len(fileExt) > 0 {
				if _, ok := extMap[fileExt[1:]]; ok {
					unique[absPath] = struct{}{}
				}
			}

		case mode.IsDir():
			if recursive {
				if walkingErr := filepath.Walk(location, func(path string, info fs.FileInfo, err error) error {
					if info.Mode().IsRegular() {
						if absPath, absErr := filepath.Abs(path); absErr != nil {
							return absErr
						} else if fileExt := filepath.Ext(info.Name()); len(fileExt) > 0 {
							if _, ok := extMap[fileExt[1:]]; ok {
								unique[absPath] = struct{}{}
							}
						}
					}

					return err
				}); walkingErr != nil {
					return nil, walkingErr
				}
			} else {
				if files, err := ioutil.ReadDir(location); err != nil {
					return nil, err
				} else {
					for _, file := range files {
						if file.Mode().IsRegular() {
							if absPath, absErr := filepath.Abs(filepath.Join(location, file.Name())); absErr != nil {
								return nil, absErr
							} else if fileExt := filepath.Ext(file.Name()); len(fileExt) > 0 {
								if _, ok := extMap[fileExt[1:]]; ok {
									unique[absPath] = struct{}{}
								}
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

// Run current command.
func (cmd *command) Run() error {
	return nil
}

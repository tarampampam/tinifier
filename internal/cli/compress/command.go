package compress

import (
	"context"
	"os"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/tarampampam/tinifier/v4/internal/breaker"
	"github.com/tarampampam/tinifier/v4/internal/env"
	appFs "github.com/tarampampam/tinifier/v4/internal/fs"
	"github.com/tarampampam/tinifier/v4/internal/logger"
	"github.com/tarampampam/tinifier/v4/internal/retry"
	"github.com/tarampampam/tinifier/v4/internal/validate"
	"github.com/tarampampam/tinifier/v4/pkg/tinypng"
)

type command struct {
	log logger.Logger
	c   *cli.Command
}

// NewCommand creates `compress` command.
func NewCommand(log logger.Logger) *cli.Command { //nolint:funlen
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
				"apiKeys =", apiKeys,
				"fileExtensions =", fileExtensions,
				"threadsCount =", threadsCount,
				"maxErrorsToStop =", maxErrorsToStop,
				"recursive =", recursive,
				"args =", paths,
			)

			if threadsCount < 1 {
				return errors.New("threads count must be greater than 0")
			}

			if len(paths) == 0 {
				return errors.New("no files or directories specified")
			}

			if len(apiKeys) == 0 {
				return errors.New("no API keys specified")
			}

			return cmd.Run(c.Context, apiKeys, paths, fileExtensions, recursive, maxErrorsToStop, threadsCount)
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
func (cmd *command) Run(
	pCtx context.Context,
	apiKeys []string,
	paths []string,
	fileExt []string,
	recursive bool,
	maxErrorsToStop uint,
	threadsCount uint,
) error {
	var (
		ctx, cancel = context.WithCancel(pCtx)  // main context creation
		oss         = breaker.NewOSSignals(ctx) // OS signals listener
	)

	oss.Subscribe(func(sig os.Signal) {
		cmd.log.Debug("Stopping by OS signal..", "signal="+sig.String())

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

	var errorsChannel = make(chan error, 1) // TODO close

	go func() { // start the errors watcher
		var counter uint

		for {
			select {
			case <-ctx.Done():
				return

			case err := <-errorsChannel:
				cmd.log.Error("Error occurred", err.Error())
				counter++

				if counter >= maxErrorsToStop {
					cmd.log.Error("Maximum errors count reached, stopping...")
					cancel()

					return
				}
			}
		}
	}()

	var (
		pool         = newClientsPool(apiKeys)
		uploadsGuard = make(chan struct{}, threadsCount) // TODO close
	)

	const (
		uploadingRetryAttempts = 5
	)

	for _, filePath := range files {
		select {
		case uploadsGuard <- struct{}{}: // would block if guard channel is already filled

		case <-ctx.Done():
			return errors.New("compression was canceled")
		}

		go func(filePath string) {
			defer func() { /* release the guard */ <-uploadsGuard }()

			cmd.log.Debug("Opening file", filePath)

			file, openingErr := os.OpenFile(filePath, os.O_RDONLY, 0) // open file
			if openingErr != nil {
				cmd.log.Error("Error opening file", filePath, "err="+openingErr.Error())

				return
			}

			defer file.Close()

			if ok, err := validate.IsImage(file); err != nil { // validate (is image?)
				cmd.log.Error("Error validating file", filePath, "err="+err.Error())

				return
			} else if !ok {
				cmd.log.Error("File is not an image", filePath)

				return
			}

			var compressed *tinypng.Compressed

			if _, err := retry.Do(func(num uint) (err error) { // retry loop
				apiKey, client := pool.Get() // get client from pool
				if client == nil {
					cmd.log.Error("No one valid API key, working canceled")
					cancel()

					return errors.New("no one valid API key")
				}

				// TODO show spinner

				compressed, err = client.Compress(ctx, file) // compress the image
				if err != nil {
					cmd.log.Warn("Error compressing file",
						filePath, "err="+err.Error(), "attempt="+strconv.Itoa(int(num)),
					)

					if errors.Is(err, tinypng.ErrTooManyRequests) || errors.Is(err, tinypng.ErrUnauthorized) {
						pool.Remove(apiKey)
					}
				}

				return err // nil or error
			},
				retry.WithContext(ctx),
				retry.Attempts(uploadingRetryAttempts),
				retry.Delay(time.Millisecond*700),
			); err != nil {
				select {
				case <-ctx.Done():
				case errorsChannel <- errors.Wrapf(err, "Image (%s) uploading failed", filePath):
				}

				return
			}

			cmd.log.Debug("File compressed", filePath, compressed)
		}(filePath)
	}

	// cmd.log.Debug("Found files", zap.Strings("files", files))

	return nil
}

// FindFiles finds files in paths.
func (*command) FindFiles(ctx context.Context, where, filesExt []string, recursive bool) ([]string, error) {
	if len(where) == 0 || len(filesExt) == 0 { // fast terminator
		return []string{}, nil
	}

	var (
		spin = spinner.New(
			[]string{" ⣾ ", " ⣽ ", " ⣻ ", " ⢿ ", " ⡿ ", " ⣟ ", " ⣯ ", " ⣷ "},
			time.Millisecond*100, //nolint:gomnd
		)
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

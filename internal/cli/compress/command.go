package compress

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
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
		apiKeyFlagName            = "api-key"
		fileExtensionsFlagName    = "ext"
		threadsCountFlagName      = "threads"
		maxErrorsToStopFlagName   = "max-errors"
		recursiveFlagName         = "recursive"
		updateFileModDateFlagName = "update-mod-date"
	)

	var cmd = command{log: log}

	cmd.c = &cli.Command{
		Name:      "compress",
		ArgsUsage: "<target-files-and-directories...>",
		Aliases:   []string{"c"},
		Usage:     "Compress images",
		Action: func(c *cli.Context) error {
			var (
				apiKeys           = c.StringSlice(apiKeyFlagName)
				fileExtensions    = c.StringSlice(fileExtensionsFlagName)
				threadsCount      = c.Uint(threadsCountFlagName)
				maxErrorsToStop   = c.Uint(maxErrorsToStopFlagName)
				recursive         = c.Bool(recursiveFlagName)
				updateFileModDate = c.Bool(updateFileModDateFlagName)
				paths             = c.Args().Slice()
			)

			log.Debug("Run args",
				logger.With("api-keys", apiKeys),
				logger.With("file-extensions", fileExtensions),
				logger.With("threads-count", threadsCount),
				logger.With("max-errors-to-stop", maxErrorsToStop),
				logger.With("recursive", recursive),
				logger.With("update-mod-date", updateFileModDate),
				logger.With("args", paths),
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

			return cmd.Run(
				c.Context,
				apiKeys,
				paths,
				fileExtensions,
				recursive,
				updateFileModDate,
				maxErrorsToStop,
				threadsCount,
			)
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
			&cli.BoolFlag{
				Name:    updateFileModDateFlagName,
				Usage:   "update file modification date/time (otherwise previous modification date/time will be kept)",
				EnvVars: []string{}, // TODO implement
			},
		},
	}

	return cmd.c
}

// Run current command.
func (cmd *command) Run( //nolint:funlen,gocognit,gocyclo
	pCtx context.Context,
	apiKeys []string,
	paths []string,
	fileExt []string,
	recursive bool,
	updateFileModDate bool,
	maxErrorsToStop uint,
	threadsCount uint,
) error {
	var (
		ctx, cancel = context.WithCancel(pCtx)  // main context creation
		oss         = breaker.NewOSSignals(ctx) // OS signals listener
	)

	oss.Subscribe(func(sig os.Signal) {
		cmd.log.Debug("Stopping by OS signal..", logger.With("Signal", sig.String()))

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
				cmd.log.Error("Error occurred", logger.With("Error", err))
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
		wg           sync.WaitGroup
	)

	const (
		uploadingRetryAttempts   = 5
		downloadingRetryAttempts = 10

		ignoreCompressedFileSizeDelta = 32 // bytes
	)

	// docs: <https://github.com/pterm/pterm/tree/master/_examples/progressbar>
	precessProgress, _ := pterm.ProgressbarPrinter{
		Total:                     len(files),
		Title:                     "Images compressing",
		BarCharacter:              "█",
		LastCharacter:             "█",
		ElapsedTimeRoundingFactor: time.Second,
		BarStyle:                  &pterm.ThemeDefault.ProgressbarBarStyle,
		TitleStyle:                &pterm.ThemeDefault.ProgressbarTitleStyle,
		ShowTitle:                 true,
		ShowCount:                 true,
		ShowPercentage:            true,
		ShowElapsedTime:           true,
		RemoveWhenDone:            true,
		BarFiller:                 " ",
		Writer:                    os.Stdout,
	}.Start()
	defer func() { _, _ = precessProgress.Stop() }()

	for _, filePath := range files {
		select {
		case uploadsGuard <- struct{}{}: // would block if guard channel is already filled
			wg.Add(1)

		case <-ctx.Done():
			return errors.New("compression was canceled")
		}

		go func(filePath string) {
			defer func() {
				wg.Done()
				precessProgress.Increment()
				<-uploadsGuard // release the guard
			}()

			if ctx.Err() != nil { // check the context
				return
			}

			var startedAt = time.Now()

			origFileStat, statErr := os.Stat(filePath)
			if statErr != nil {
				cmd.log.Error("Error file statistics reading",
					logger.With("File", filePath),
					logger.With("Error", statErr),
				)

				return
			}

			var (
				compressed     *tinypng.Compressed
				errInvalidFile = errors.New("invalid file")
			)

			// STEP 1. Upload the file to TinyPNG
			if _, err := retry.Do(func(attemptNum uint) error { // uploading retry loop
				f, openingErr := os.OpenFile(filePath, os.O_RDONLY, 0) // open file
				if openingErr != nil {
					cmd.log.Error("Error opening file",
						logger.With("File", filePath),
						logger.With("Error", openingErr),
					)

					return openingErr
				}

				defer f.Close()

				if ok, err := validate.IsImage(f); err != nil { // validate (is image?)
					cmd.log.Error("Error validating file",
						logger.With("File", filePath),
						logger.With("Error", err),
					)

					return err
				} else if !ok {
					cmd.log.Error("File is not an image", logger.With("File", filePath))

					return errInvalidFile
				}

				apiKey, client := pool.Get() // get client from pool
				if client == nil {
					cmd.log.Error("No one valid API key, working canceled")
					cancel()

					return errors.New("no one valid API key")
				}

				var compErr error // compressing error

				compressed, compErr = client.Compress(ctx, f) // compress the image
				if compErr != nil {
					cmd.log.Warn("Error compressing file",
						logger.With("File", filePath),
						logger.With("Attempt", attemptNum),
						logger.With("Error", compErr),
					)

					if errors.Is(compErr, tinypng.ErrTooManyRequests) || errors.Is(compErr, tinypng.ErrUnauthorized) {
						pool.Remove(apiKey)
					}

					return compErr
				}

				return nil
			},
				retry.WithContext(ctx),
				retry.Attempts(uploadingRetryAttempts),
				retry.Delay(time.Millisecond*700), //nolint:gomnd
				retry.StopOnError(errInvalidFile),
			); err != nil {
				select {
				case <-ctx.Done():
				case errorsChannel <- errors.Wrapf(err, "Image (%s) uploading failed", filePath):
				}

				return
			}

			if ctx.Err() != nil { // check the context
				return
			}

			cmd.log.Debug("File compressed", logger.With("File", filePath))

			if uint64(origFileStat.Size()) <= compressed.Size()+ignoreCompressedFileSizeDelta {
				cmd.log.Info(fmt.Sprintf("File %s ignored (the size of the compressed file has not changed)", path.Base(filePath)),
					logger.With("File", filePath),
					logger.With("Elapsed time", time.Since(startedAt).Round(time.Second).String()),
					logger.With("Original file size", humanize.Bytes(uint64(origFileStat.Size()))),
					logger.With("Compressed file size", humanize.Bytes(compressed.Size())),
				)

				return
			}

			var tmpFilePath = filePath + ".tiny" // temporary file path

			defer func() {
				if _, err := os.Stat(tmpFilePath); err == nil { // check if temporary file exists
					if err = os.Remove(tmpFilePath); err != nil { // remove the temporary file
						cmd.log.Warn("Error removing temporary file",
							logger.With("File", tmpFilePath),
							logger.With("Error", err),
						)
					}
				}
			}()

			// STEP 2. Download the compressed file from TinyPNG to temporary file
			if _, err := retry.Do(func(uint) error {
				f, err := os.OpenFile(tmpFilePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE|os.O_SYNC, origFileStat.Mode())
				if err != nil {
					return err
				}

				defer f.Close()

				return compressed.Download(ctx, f)
			},
				retry.WithContext(ctx),
				retry.Attempts(downloadingRetryAttempts),
				retry.Delay(time.Second+time.Millisecond*500),
			); err != nil {
				select {
				case <-ctx.Done():
				case errorsChannel <- errors.Wrapf(err, "Image (%s) downloading failed", compressed.URL()):
				}

				return
			}

			if ctx.Err() != nil { // check the context
				return
			}

			// STEP 3. Replace original file content with compressed
			origFile, origFileErr := os.OpenFile(filePath, os.O_WRONLY, 0)
			if origFileErr != nil {
				cmd.log.Error("Error opening original file",
					logger.With("File", filePath),
					logger.With("Error", origFileErr),
				)

				return
			}

			defer origFile.Close()

			tmpFile, tmpFileErr := os.OpenFile(tmpFilePath, os.O_RDONLY, 0)
			if tmpFileErr != nil {
				cmd.log.Error("Error opening compressed file",
					logger.With("File", tmpFilePath),
					logger.With("Error", tmpFileErr),
				)

				return
			}

			defer tmpFile.Close()

			tmpFileStat, tmpFileStatErr := tmpFile.Stat()
			if tmpFileStatErr != nil {
				cmd.log.Error("Error getting compressed file stat",
					logger.With("File", tmpFilePath),
					logger.With("Error", tmpFileStatErr),
				)

				return
			}

			if _, err := io.Copy(origFile, tmpFile); err != nil { // copy compressed file content to original file
				cmd.log.Error("Error copying compressed file content to original file",
					logger.With("From", tmpFilePath),
					logger.With("To", filePath),
					logger.With("Error", err),
				)

				return
			}

			if !updateFileModDate { // restore original file modification date
				if err := os.Chtimes(filePath, origFileStat.ModTime(), origFileStat.ModTime()); err != nil {
					cmd.log.Error("Error changing file modification time",
						logger.With("File", filePath),
						logger.With("Error", err),
					)

					return
				}
			}

			var oldSize, newSize = float64(origFileStat.Size()), float64(tmpFileStat.Size())

			cmd.log.Success(fmt.Sprintf("File %s compressed", path.Base(filePath)),
				logger.With("File", filePath),
				logger.With("Elapsed time", time.Since(startedAt).Round(time.Second).String()),
				logger.With("Original file size", humanize.Bytes(uint64(oldSize))),
				logger.With("Compressed file size", humanize.Bytes(uint64(newSize))),
				logger.With("Saved space", fmt.Sprintf(
					"%s (%0.2f%%)",
					humanize.IBytes(uint64(oldSize-newSize)),
					((oldSize-newSize)/newSize)*100, //nolint:gomnd
				)),
			)
		}(filePath)
	}

	wg.Wait()
	_, _ = precessProgress.Stop() //nolint:wsl

	// cmd.log.Debug("Found files", zap.Strings("files", files))

	return nil
}

// FindFiles finds files in paths.
func (*command) FindFiles(ctx context.Context, where, filesExt []string, recursive bool) ([]string, error) {
	if len(where) == 0 || len(filesExt) == 0 { // fast terminator
		return []string{}, nil
	}

	spin, _ := pterm.SpinnerPrinter{ // docs: <https://github.com/pterm/pterm/tree/master/_examples/spinner>
		Sequence:            []string{" ⣾", " ⣽", " ⣻", " ⢿", " ⡿", " ⣟", " ⣯", " ⣷"},
		Delay:               time.Millisecond * 200, //nolint:gomnd
		Style:               &pterm.ThemeDefault.SpinnerStyle,
		TimerStyle:          &pterm.ThemeDefault.TimerStyle,
		MessageStyle:        &pterm.ThemeDefault.SpinnerTextStyle,
		RemoveWhenDone:      true,
		ShowTimer:           true,
		TimerRoundingFactor: time.Second,
		Writer:              os.Stdout,
	}.Start("Images searching")
	defer func() { _ = spin.Stop() }()

	var found = make([]string, 0, len(where))

	if err := appFs.FindFiles(ctx, where, func(absPath string) {
		found = append(found, absPath)

		spin.UpdateText(fmt.Sprintf("Found image: %s (%d total)", absPath, len(found)))
	}, appFs.WithRecursive(recursive), appFs.WithFilesExt(filesExt...)); err != nil {
		return nil, err
	}

	sort.Strings(found)

	return found, nil
}

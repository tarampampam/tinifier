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
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/tarampampam/tinifier/v4/internal/breaker"
	"github.com/tarampampam/tinifier/v4/internal/env"
	"github.com/tarampampam/tinifier/v4/internal/files"
	"github.com/tarampampam/tinifier/v4/internal/retry"
	"github.com/tarampampam/tinifier/v4/internal/validate"
	"github.com/tarampampam/tinifier/v4/pkg/tinypng"
)

type command struct {
	c *cli.Command
}

// NewCommand creates `compress` command.
func NewCommand() *cli.Command { //nolint:funlen
	const (
		apiKeyFlagName            = "api-key"
		fileExtensionsFlagName    = "ext"
		threadsCountFlagName      = "threads"
		maxErrorsToStopFlagName   = "max-errors"
		recursiveFlagName         = "recursive"
		updateFileModDateFlagName = "update-mod-date"
	)

	var cmd = command{}

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

			// log.Debug("Run args",
			// 	zap.Strings("api-keys", apiKeys),
			// 	zap.Strings("file-extensions", fileExtensions),
			// 	zap.Uint("threads-count", threadsCount),
			// 	zap.Uint("max-errors-to-stop", maxErrorsToStop),
			// 	zap.Bool("recursive", recursive),
			// 	zap.Bool("update-mod-date", updateFileModDate),
			// 	zap.Strings("args", paths),
			// )

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
				Value:   uint(runtime.NumCPU() * 6), //nolint:gomnd
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

type compressionStats struct {
	filePath       string
	fileType       string
	originalSize   uint64
	compressedSize uint64
}

var errNoClients = errors.New("no clients")

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

	oss.Subscribe(func(os.Signal) { cancel() })

	defer func() {
		cancel()   // call cancellation function after all for "service" goroutines stopping
		oss.Stop() // stop system signals listening
	}()

	filesList, findErr := cmd.FindFiles(ctx, paths, fileExt, recursive)
	if findErr != nil {
		if errors.Is(findErr, context.Canceled) {
			return errors.New("images searching was canceled")
		}

		return findErr
	}

	if len(filesList) == 0 {
		return errors.New("nothing to compress (files not found)")
	}

	var (
		errorsChannel = make(chan error, 1)
		statsChannel  = make(chan compressionStats, 1)
		statsBuf      = struct {
			history             []compressionStats
			totalOriginalSize   uint64
			totalCompressedSize uint64
		}{
			history: make([]compressionStats, 0, len(filesList)),
		}
	)

	var pw, totalTracker = progress.NewWriter(), progress.Tracker{
		Total:   int64(len(filesList)),
		Units:   unitsAsIs,
		Message: "› Overall progress",
	}

	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetStyle(progressStyleDefault)
	pw.Style().Visibility.Value = false
	pw.Style().Options.Separator = ": "
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.AppendTracker(&totalTracker)
	pw.SetSortBy(progress.SortByMessage)
	go pw.Render()

	go func(ch <-chan error) { // start the errors watcher
		var counter uint

		for {
			select {
			case <-ctx.Done():
				return

			case err, isOpened := <-ch:
				if !isOpened {
					return
				}

				pw.Log("Error occurred: %s", err.Error())
				counter++

				if counter >= maxErrorsToStop {
					pw.Log("Maximum errors count reached, stopping...")
					cancel()

					return
				}
			}
		}
	}(errorsChannel)

	go func(ch <-chan compressionStats) { // start the stats watcher
		for {
			select {
			case <-ctx.Done():
				return

			case stat, isOpened := <-ch:
				if !isOpened {
					return
				}

				statsBuf.history = append(statsBuf.history, stat)
				statsBuf.totalOriginalSize += stat.originalSize
				statsBuf.totalCompressedSize += stat.compressedSize
			}
		}
	}(statsChannel)

	var (
		pool         = newClientsPool(apiKeys)
		uploadsGuard = make(chan struct{}, threadsCount) // TODO close
		wg           sync.WaitGroup
	)

workersLoop:
	for _, filePath := range filesList {
		select {
		case uploadsGuard <- struct{}{}: // would block if guard channel is already filled
			wg.Add(1)

		case <-ctx.Done():
			break workersLoop
		}

		go func(filePath string) {
			defer func() { wg.Done(); <-uploadsGuard /* release the guard */ }()
			defer totalTracker.Increment(1)

			if ctx.Err() != nil { // check the context
				return
			}

			var fileName = path.Base(filePath)
			var tracker = progress.Tracker{Message: fileName, Total: 3, Units: unitsAsIs}

			pw.AppendTracker(&tracker)

			defer func() {
				if !tracker.IsDone() {
					tracker.MarkAsErrored()
				}
			}()

			origFileStat, statErr := os.Stat(filePath)
			if statErr != nil {
				select {
				case <-ctx.Done():
				case errorsChannel <- errors.Wrapf(statErr, "file (%s) statistics reading failed", filePath):
				}

				return // stop the process on error
			}

			// STEP 1. Upload the file to TinyPNG server
			tracker.SetValue(1)
			tracker.UpdateMessage(fmt.Sprintf("%s uploading (%s)", fileName, humanize.Bytes(uint64(origFileStat.Size()))))
			compressed, compErr := cmd.Upload(ctx, pool, filePath)
			if compErr != nil {
				switch {
				case errors.Is(compErr, errNoClients):
					pw.Log("No one valid API key, working canceled")
					cancel()

				case errors.Is(compErr, context.Canceled): // do nothing

				default:
					select {
					case <-ctx.Done():
					case errorsChannel <- errors.Wrapf(compErr, "image (%s) uploading failed", filePath):
					}
				}

				return // stop the process on uploading error
			}

			if ctx.Err() != nil { // check the context
				return
			}

			if size := uint64(origFileStat.Size()); size <= compressed.Size() {
				tracker.UpdateMessage(fmt.Sprintf("%s skipped (original size is less than compressed)", fileName))

				return
			}

			var tmpFilePath = filePath + ".tiny" // temporary file path

			defer func() {
				if _, err := os.Stat(tmpFilePath); err == nil { // check the temporary file existence
					if err = os.Remove(tmpFilePath); err != nil { // remove the temporary file
						pw.Log("Error removing temporary file %s (%s)",
							tmpFilePath,
							err.Error(),
						)
					}
				}
			}()

			// STEP 2. Download the compressed file from TinyPNG to temporary file
			tracker.SetValue(2)
			tracker.UpdateMessage(fmt.Sprintf("%s downloading (%s)", fileName, humanize.Bytes(compressed.Size())))
			if err := cmd.Download(ctx, compressed, tmpFilePath, origFileStat.Mode()); err != nil {
				select {
				case <-ctx.Done():
				case errorsChannel <- errors.Wrapf(err, "image (%s) downloading failed", filePath):
				}

				return // stop the process on error
			}

			if ctx.Err() != nil { // check the context
				return
			}

			tmpFileStat, statErr := os.Stat(tmpFilePath)
			if statErr != nil {
				select {
				case <-ctx.Done():
				case errorsChannel <- errors.Wrapf(statErr, "file (%s) statistics reading failed", tmpFilePath):
				}

				return // stop the process on error
			}

			// STEP 3. Replace original file content with compressed
			tracker.SetValue(3)
			if err := cmd.Replace(filePath, tmpFilePath, !updateFileModDate); err != nil {
				select {
				case <-ctx.Done():
				case errorsChannel <- errors.Wrapf(err, "content copying (%s -> %s) failed", tmpFilePath, filePath):
				}

				return // stop the process on error
			}

			tracker.UpdateMessage(fmt.Sprintf(
				"%s compressed (%s → %s)",
				fileName,
				humanize.Bytes(uint64(origFileStat.Size())),
				humanize.Bytes(compressed.Size()),
			))

			// var oldSize, newSize = float64(origFileStat.Size()), float64(tmpFileStat.Size())
			//
			// pw.Log(fmt.Sprintf("File %s compressed", pterm.Bold.Sprint(path.Base(filePath))),
			// 	zap.String("File", filePath),
			// 	zap.String("Elapsed time", time.Since(startedAt).Round(time.Second).String()),
			// 	zap.String("Original file size", humanize.Bytes(uint64(oldSize))),
			// 	zap.String("Compressed file size", humanize.Bytes(uint64(newSize))),
			// 	zap.String("Saved space", fmt.Sprintf(
			// 		"%s (%0.2f%%)",
			// 		humanize.IBytes(uint64(oldSize-newSize)),
			// 		((oldSize-newSize)/newSize)*100, //nolint:gomnd
			// 	)),
			// )

			select {
			case <-ctx.Done():
			case statsChannel <- compressionStats{
				filePath:       filePath,
				fileType:       compressed.Type(),
				originalSize:   uint64(origFileStat.Size()),
				compressedSize: uint64(tmpFileStat.Size()),
			}:
			}

			tracker.MarkAsDone()
		}(filePath)
	}

	wg.Wait()
	pw.Stop()

	close(uploadsGuard)
	close(errorsChannel)
	close(statsChannel)

	if len(statsBuf.history) > 0 {
		(&pterm.HeaderPrinter{
			TextStyle:       &pterm.Style{pterm.FgLightWhite, pterm.Bold},
			BackgroundStyle: &pterm.Style{pterm.BgBlue},
			Margin:          5, //nolint:gomnd
			FullWidth:       true,
			Writer:          os.Stdout,
		}).Println("Compression results")

		_ = pterm.DefaultBarChart.WithHorizontal().WithShowValue().WithBars(pterm.Bars{
			pterm.Bar{
				Label: fmt.Sprintf("Original files size (%s)", humanize.Bytes(statsBuf.totalOriginalSize)),
				Value: int(statsBuf.totalOriginalSize),
			},
			pterm.Bar{
				Label: fmt.Sprintf("Compressed files size (%s)", humanize.Bytes(statsBuf.totalCompressedSize)),
				Value: int(statsBuf.totalCompressedSize),
			},
		}).Render()
	} else {
		fmt.Println("No files compressed")
	}

	return nil
}

// Upload uploads the file to TinyPNG server.
func (*command) Upload(ctx context.Context, pool *clientsPool, filePath string) (*tinypng.Compressed, error) {
	const (
		retryAttempts = 5
		retryInterval = time.Millisecond * 700
	)

	var (
		compressed           *tinypng.Compressed
		errInvalidFileFormat = errors.New("invalid file format")
	)

	if limitExceeded, err := retry.Do(func(attemptNum uint) error { // uploading retry loop
		f, openingErr := os.OpenFile(filePath, os.O_RDONLY, 0) // open file
		if openingErr != nil {
			return openingErr
		}

		defer func() { _ = f.Close() }()

		if ok, err := validate.IsImage(f); err != nil { // validate (is image?)
			return err
		} else if !ok {
			return errInvalidFileFormat
		}

		apiKey, client := pool.Get() // get client from pool
		if client == nil || apiKey == "" {
			return errNoClients
		}

		if c, err := client.Compress(ctx, f); err != nil {
			if errors.Is(err, tinypng.ErrTooManyRequests) || errors.Is(err, tinypng.ErrUnauthorized) {
				pool.Remove(apiKey)
			}

			return err
		} else {
			compressed = c
		}

		return nil
	},
		retry.WithContext(ctx),
		retry.Attempts(retryAttempts),
		retry.Delay(retryInterval),
		retry.StopOnError(errInvalidFileFormat),
	); err != nil {
		return nil, err
	} else if limitExceeded {
		return nil, errors.New("too many attempts to compress (upload) the file " + path.Base(filePath))
	}

	return compressed, nil
}

// Download downloads the compressed file from TinyPNG server.
func (*command) Download(ctx context.Context, comp *tinypng.Compressed, filePath string, perm os.FileMode) error {
	const (
		retryAttempts = 10
		retryInterval = time.Second + time.Millisecond*500
	)

	if limitExceeded, err := retry.Do(func(uint) error {
		f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE|os.O_SYNC, perm)
		if err != nil {
			return err
		}

		defer func() { _ = f.Close() }()

		return comp.Download(ctx, f)
	}, retry.WithContext(ctx), retry.Attempts(retryAttempts), retry.Delay(retryInterval)); err != nil {
		return err
	} else if limitExceeded {
		return errors.New("too many attempts to download the file")
	}

	return nil
}

// Replace replaces the original file content with the compressed one (from temporary file).
func (*command) Replace(origFilePath, tmpFilePath string, keepOriginalFileModTime bool) error {
	origFile, origFileErr := os.OpenFile(origFilePath, os.O_WRONLY, 0)
	if origFileErr != nil {
		return origFileErr
	}

	defer func() { _ = origFile.Close() }()

	origFileStat, statErr := origFile.Stat()
	if statErr != nil {
		return statErr
	}

	tmpFile, tmpFileErr := os.OpenFile(tmpFilePath, os.O_RDONLY, 0)
	if tmpFileErr != nil {
		return tmpFileErr
	}

	defer func() { _ = tmpFile.Close() }()

	if _, err := io.Copy(origFile, tmpFile); err != nil { // copy compressed file content to original file
		return err
	}

	if keepOriginalFileModTime { // restore original file modification date
		// atime: time of last access (ls -lu),
		// mtime: time of last modification (ls -l)
		if err := os.Chtimes(origFilePath, time.Now(), origFileStat.ModTime()); err != nil {
			return err
		}
	}

	return nil
}

// FindFiles finds files in paths.
func (*command) FindFiles(ctx context.Context, where, filesExt []string, recursive bool) ([]string, error) {
	if len(where) == 0 || len(filesExt) == 0 { // fast terminator
		return []string{}, nil
	}

	var pw, tracker = progress.NewWriter(), progress.Tracker{Total: 0, Units: unitsAsIs}

	pw.SetStyle(progressStyleDefault)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.AppendTracker(&tracker)

	go pw.Render()
	defer pw.Stop()

	var found = make([]string, 0, len(where))

	if err := files.FindFiles(ctx, where, func(absPath string) {
		found = append(found, absPath)

		tracker.UpdateMessage(path.Base(absPath))
		tracker.SetValue(int64(len(found)))
	}, files.WithRecursive(recursive), files.WithFilesExt(filesExt...)); err != nil {
		tracker.MarkAsErrored()

		return nil, errors.Wrap(err, "wrong target path")
	}

	tracker.UpdateMessage("Images searching")
	tracker.MarkAsDone()

	sort.Strings(found)

	return found, nil
}

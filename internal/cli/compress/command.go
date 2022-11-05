package compress

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/tarampampam/tinifier/v4/internal/breaker"
	"github.com/tarampampam/tinifier/v4/internal/cli/shared"
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
		fileExtensionsFlagName    = "ext"
		threadsCountFlagName      = "threads"
		maxErrorsToStopFlagName   = "max-errors"
		recursiveFlagName         = "recursive"
		updateFileModDateFlagName = "update-mod-date"
		keepOriginalFileFlagName  = "keep-original-file"
	)

	var cmd = command{}

	cmd.c = &cli.Command{
		Name:      "compress",
		ArgsUsage: "<target-files-and-directories...>",
		Aliases:   []string{"c"},
		Usage:     "Compress images",
		Action: func(c *cli.Context) error {
			var (
				apiKeys           = c.StringSlice(shared.APIKeyFlag.Name)
				fileExtensions    = c.StringSlice(fileExtensionsFlagName)
				threadsCount      = c.Uint(threadsCountFlagName)
				maxErrorsToStop   = c.Uint(maxErrorsToStopFlagName)
				recursive         = c.Bool(recursiveFlagName)
				updateFileModDate = c.Bool(updateFileModDateFlagName)
				keepOriginalFile  = c.Bool(keepOriginalFileFlagName)
				paths             = c.Args().Slice()
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
				keepOriginalFile,
				maxErrorsToStop,
				threadsCount,
			)
		},
		Flags: []cli.Flag{
			shared.APIKeyFlag,
			&cli.StringSliceFlag{
				Name:    fileExtensionsFlagName,
				Aliases: []string{"e"},
				Usage:   "image file extensions (without leading dots)",
				Value:   cli.NewStringSlice("jpg", "JPG", "jpeg", "JPEG", "png", "PNG"),
				// EnvVars: []string{}, // TODO implement
			},
			&cli.UintFlag{
				Name:    threadsCountFlagName,
				Aliases: []string{"t"},
				Usage:   "threads count",
				Value:   uint(runtime.NumCPU() * 6), //nolint:gomnd
				EnvVars: []string{env.ThreadsCount.String()},
			},
			&cli.UintFlag{
				Name:  maxErrorsToStopFlagName,
				Usage: "maximum errors count to stop the process (set 0 to disable)",
				Value: 10, //nolint:gomnd
				// EnvVars: []string{}, // TODO implement
			},
			&cli.BoolFlag{
				Name:    recursiveFlagName,
				Aliases: []string{"r"},
				Usage:   "search for files in listed directories recursively",
				// EnvVars: []string{}, // TODO implement
			},
			&cli.BoolFlag{
				Name:  updateFileModDateFlagName,
				Usage: "change file modification date/time (otherwise, the original file modification date/time will be kept)",
				// EnvVars: []string{}, // TODO implement
			},
			&cli.BoolFlag{
				Name:  keepOriginalFileFlagName,
				Usage: "leave the original (uncompressed) file near the compressed one (with the .orig extension)",
				// EnvVars: []string{}, // TODO implement
			},
		},
	}

	return cmd.c
}

// Run current command.
func (cmd *command) Run( //nolint:funlen
	pCtx context.Context,
	apiKeys []string,
	paths []string,
	fileExt []string,
	recursive bool,
	updateFileModDate bool,
	keepOriginalFile bool,
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
		errorsWatcher                = make(ErrorsWatcher, 1)
		stats         StatsCollector = NewStatsStorage(len(filesList))
		pw                           = newProgressBar(len(filesList), true)
	)

	go pw.Render()

	go errorsWatcher.Watch(ctx, maxErrorsToStop, // start the errors watcher
		WithOnErrorHandler(func(err error) {
			pw.Log("Error occurred: %s", err.Error())
		}),
		WithLimitExceededHandler(func() {
			pw.Log("Maximum errors count reached, stopping...")

			cancel()
		}),
	)

	go stats.Watch(ctx) // start the stats watcher

	var (
		pool  = newClientsPool(apiKeys)
		guard = make(chan struct{}, threadsCount)
		wg    sync.WaitGroup
	)

workersLoop:
	for _, filePath := range filesList {
		select {
		case guard <- struct{}{}: // would block if guard channel is already filled
			wg.Add(1)

		case <-ctx.Done():
			break workersLoop
		}

		go func(filePath string) {
			defer func() { wg.Done(); <-guard /* release the guard */ }()

			if err := cmd.ProcessFile(ctx, pw, pool, stats, filePath, !updateFileModDate, keepOriginalFile); err != nil {
				switch {
				case errors.Is(err, errNoClients):
					pw.Log("No one valid API key, working canceled")
					cancel()

				case errors.Is(err, context.Canceled): // do nothing

				default:
					errorsWatcher.Push(ctx, err)
				}
			}
		}(filePath)
	}

	wg.Wait()

	pw.Style().Visibility.TrackerOverall = false // fix abandoned tracker on CTRL+C pressing
	pw.Stop()

	close(guard)
	close(errorsWatcher)
	stats.Close()

	tbl := table.NewWriter()
	tbl.SetStyle(table.StyleColoredBlackOnBlueWhite)
	tbl.AppendHeader(table.Row{"File Name", "Type", "Size Difference", "Saved"})

	for _, stat := range stats.History() {
		tbl.AppendRow(table.Row{
			path.Base(stat.FilePath), // File Name
			stat.FileType,            // Type
			fmt.Sprintf( // Size Difference
				"%s → %s",
				humanize.IBytes(stat.OriginalSize),
				humanize.IBytes(stat.CompressedSize),
			),
			fmt.Sprintf( // Saved
				"%s (%s)",
				humanize.IBytes(stat.OriginalSize-stat.CompressedSize),
				cmd.percentageDiff(float64(stat.CompressedSize), float64(stat.OriginalSize)),
			),
		})
	}

	tbl.AppendFooter(table.Row{"", "", fmt.Sprintf( // Overall size difference
		"%s → %s",
		humanize.IBytes(stats.TotalOriginalSize()),
		humanize.IBytes(stats.TotalCompressedSize()),
	)})

	tbl.AppendFooter(table.Row{"", "",
		fmt.Sprintf("Total saved (%d files)", stats.TotalFiles()),
		fmt.Sprintf("%s (%s)", // Saved
			humanize.IBytes(uint64(stats.TotalSavedBytes())),
			cmd.percentageDiff(float64(stats.TotalCompressedSize()), float64(stats.TotalOriginalSize())),
		),
	})

	if _, err := fmt.Fprintf(os.Stdout, "\n%s\n", tbl.Render()); err != nil {
		return err
	}

	return nil
}

func (cmd *command) ProcessFile( //nolint:funlen
	ctx context.Context,
	pw progress.Writer,
	pool *clientsPool,
	stats StatsCollector,
	filePath string,
	keepOriginalFileModTime, keepOriginalFile bool,
) error {
	if err := ctx.Err(); err != nil { // check the context
		return err
	}

	const (
		totalStepsCount = 3
		stepUpload      = iota
		stepDownload
		stepReplace
	)

	var (
		fileName = path.Base(filePath)
		tracker  = progress.Tracker{Message: fileName, Total: totalStepsCount, Units: unitsAsIs}
	)

	pw.AppendTracker(&tracker)

	defer func() {
		if !tracker.IsDone() {
			tracker.MarkAsErrored()
		}
	}()

	origFileStat, statErr := os.Stat(filePath)
	if statErr != nil {
		return errors.Wrapf(statErr, "file (%s) statistics reading failed", filePath)
	}

	// STEP 1. Upload the file to TinyPNG server
	tracker.SetValue(stepUpload)
	tracker.UpdateMessage(fmt.Sprintf("%s uploading (%s)", fileName, humanize.IBytes(uint64(origFileStat.Size()))))

	compressed, compErr := cmd.Upload(ctx, pool, filePath)
	if compErr != nil {
		return errors.Wrapf(compErr, "image (%s) uploading failed", filePath)
	}

	if err := ctx.Err(); err != nil { // check the context
		return err
	}

	if size := uint64(origFileStat.Size()); size <= compressed.Size() {
		tracker.UpdateMessage(fmt.Sprintf("%s skipped (original size less than compressed)", fileName))

		return nil
	}

	var tmpFilePath = filePath + ".tiny" // temporary file path

	defer func() {
		if _, err := os.Stat(tmpFilePath); err == nil { // check the temporary file existence
			if err = os.Remove(tmpFilePath); err != nil { // remove the temporary file
				pw.Log("Error removing temporary file %s (%s)", tmpFilePath, err.Error())
			}
		}
	}()

	// STEP 2. Download the compressed file from TinyPNG to temporary file
	tracker.SetValue(stepDownload)
	tracker.UpdateMessage(fmt.Sprintf("%s downloading (%s)", fileName, humanize.IBytes(compressed.Size())))

	if err := cmd.Download(ctx, compressed, tmpFilePath, origFileStat.Mode()); err != nil {
		return errors.Wrapf(err, "image (%s) downloading failed", filePath)
	}

	if err := ctx.Err(); err != nil { // check the context
		return err
	}

	tmpFileStat, statErr := os.Stat(tmpFilePath)
	if statErr != nil {
		return statErr
	}

	// STEP 3. Replace original file content with compressed
	tracker.SetValue(stepReplace)

	if err := cmd.Replace(filePath, tmpFilePath, keepOriginalFileModTime, keepOriginalFile); err != nil {
		return errors.Wrapf(err, "content copying (%s -> %s) failed", tmpFilePath, filePath)
	}

	tracker.UpdateMessage(fmt.Sprintf(
		"%s compressed (%s → %s)",
		fileName,
		humanize.IBytes(uint64(origFileStat.Size())),
		humanize.IBytes(uint64(tmpFileStat.Size())),
	))

	stats.Push(ctx, CompressionStat{
		FilePath:       filePath,
		FileType:       compressed.Type(),
		OriginalSize:   uint64(origFileStat.Size()),
		CompressedSize: uint64(tmpFileStat.Size()),
	})

	tracker.MarkAsDone()

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
func (*command) Replace(origFilePath, tmpFilePath string, keepOriginalFileModTime, keepOriginalFile bool) error {
	origFile, origFileErr := os.OpenFile(origFilePath, os.O_RDWR, 0)
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

	if keepOriginalFile { // make a copy of original file before overwriting
		origCopyFile, origCopyFileErr := os.OpenFile(
			origFilePath+".orig",
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			origFileStat.Mode().Perm(),
		)
		if origCopyFileErr != nil {
			return origCopyFileErr
		}

		if _, err := io.Copy(origCopyFile, origFile); err != nil {
			return err
		}

		if _, err := origFile.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}

	if err := origFile.Truncate(0); err != nil {
		return err
	}

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

	var pw, tracker = newProgressBar(1, false), progress.Tracker{Total: 0, Units: unitsAsIs}

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

// percentageDiff calculates difference between passed values in percentage representation.
func (*command) percentageDiff(from, to float64) string {
	return fmt.Sprintf("%0.2f%%", math.Abs(((from-to)/to)*100)) //nolint:gomnd
}

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gh.tarampamp.am/tinifier/v5/internal/cli/cmd"
	"gh.tarampamp.am/tinifier/v5/internal/config"
	"gh.tarampamp.am/tinifier/v5/internal/finder"
	"gh.tarampamp.am/tinifier/v5/internal/humanize"
	"gh.tarampamp.am/tinifier/v5/internal/retry"
	"gh.tarampamp.am/tinifier/v5/internal/version"
	"gh.tarampamp.am/tinifier/v5/pkg/tinypng"
)

//go:generate go run ./generate/readme.go

type App struct {
	cmd   cmd.Command
	opt   options
	logMu sync.Mutex
}

func NewApp(name string) *App { //nolint:funlen
	var app = App{
		cmd: cmd.Command{
			Name:        name,
			Description: "CLI tool for compressing images using the TinyPNG.",
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
			Usage:   "Extensions of files to compress (separated by commas)",
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
		retryAttempts = cmd.Flag[uint]{
			Names:   []string{"retry-attempts"},
			Usage:   "Number of retry attempts for upload/download/replace operations",
			EnvVars: []string{"RETRY_ATTEMPTS"},
			Default: app.opt.RetryAttempts,
		}
		delayBetweenRetries = cmd.Flag[time.Duration]{
			Names:   []string{"delay-between-retries"},
			Usage:   "Delay between retry attempts",
			EnvVars: []string{"DELAY_BETWEEN_RETRIES"},
			Default: app.opt.DelayBetweenRetries,
		}
		recursive = cmd.Flag[bool]{
			Names:   []string{"recursive", "r"},
			Usage:   "Search for files in listed directories recursively",
			EnvVars: []string{"RECURSIVE"},
			Default: app.opt.Recursive,
		}
		skipIfDiffLessThan = cmd.Flag[float64]{
			Names:   []string{"skip-if-diff-less"},
			Usage:   "Skip files if the diff between the original and compressed file sizes < N%",
			EnvVars: []string{"SKIP_IF_DIFF_LESS"},
			Default: app.opt.SkipIfDiffLessThan,
		}
		preserveTime = cmd.Flag[bool]{
			Names:   []string{"preserve-time", "p"},
			Usage:   "Preserve the original file modification date/time (including EXIF)",
			EnvVars: []string{"PRESERVE_TIME"},
			Default: app.opt.PreserveTime,
		}
		keepOriginalFile = cmd.Flag[bool]{
			Names:   []string{"keep-original-file"},
			Usage:   "Leave the original (uncompressed) file next to the compressed one (with the .orig extension)",
			EnvVars: []string{"KEEP_ORIGINAL_FILE"},
			Default: app.opt.KeepOriginalFile,
		}
	)

	app.cmd.Flags = []cmd.Flagger{
		&configFile,
		&apiKeys,
		&fileExtensions,
		&threatsCount,
		&maxErrorsToStop,
		&retryAttempts,
		&delayBetweenRetries,
		&recursive,
		&skipIfDiffLessThan,
		&preserveTime,
		&keepOriginalFile,
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
			setIfFlagIsSet(&app.opt.RetryAttempts, retryAttempts)
			setIfFlagIsSet(&app.opt.DelayBetweenRetries, delayBetweenRetries)
			setIfFlagIsSet(&app.opt.Recursive, recursive)
			setIfFlagIsSet(&app.opt.SkipIfDiffLessThan, skipIfDiffLessThan)
			setIfFlagIsSet(&app.opt.PreserveTime, preserveTime)
			setIfFlagIsSet(&app.opt.KeepOriginalFile, keepOriginalFile)
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

// Help returns the application's help message.
func (a *App) Help() string { return a.cmd.Help() }

// run executes the main logic of the application.
func (a *App) run(pCtx context.Context, paths []string) error { //nolint:gocognit,funlen,gocyclo
	var ctx, cancel = context.WithCancel(pCtx)
	defer cancel() // canceling the context stops the process

	var iterCtx, cancelIter = context.WithCancel(ctx)
	defer cancelIter() // stopping the iterator

	var (
		filesSeq    = finder.Files(iterCtx, paths, a.opt.Recursive, finder.FilterByExt(false, a.opt.FileExtensions...))
		totalAmount atomic.Uint64
	)

	// count total files in the background to prevent blocking the main process
	go func(count uint64) {
		for range filesSeq { // iterator respects the context, so no extra checks are needed
			count++
		}

		totalAmount.Store(count)
	}(0)

	var (
		errs       = make(chan error, max(1, a.opt.ThreadsCount))
		errsClosed = make(chan struct{})
	)

	// process errors in the background and stop execution if necessary
	go func() {
		defer close(errsClosed)

		var (
			counter uint
			once    sync.Once
		)

		for err := range errs {
			counter++

			if !errors.Is(err, context.Canceled) {
				a.errorf("[%d%s Error] %s",
					counter,
					func() string {
						if maxErr := a.opt.MaxErrorsToStop; maxErr > 0 {
							return fmt.Sprintf("/%d", maxErr)
						}

						return ""
					}(),
					err,
				)
			}

			if maxErr := a.opt.MaxErrorsToStop; maxErr > 0 && counter >= maxErr {
				once.Do(func() {
					a.errorf("Maximum number of errors reached, stopping the process")

					cancelIter()
				})
			}
		}
	}()

	var (
		pool        = tinypng.NewClientsPool(a.opt.ApiKeys)
		guard       = make(chan struct{}, max(1, a.opt.ThreadsCount))
		stats       fileStats
		wg          sync.WaitGroup // ensures all jobs are complete before exiting
		fileCounter uint64

		once sync.Once
	)

	for path := range filesSeq {
		once.Do(func() {
			a.logf(
				"Compression process has started (%s). Please be patient...",
				strings.Join([]string{
					fmt.Sprintf("keys = %d", len(a.opt.ApiKeys)),
					fmt.Sprintf("threads = %d", a.opt.ThreadsCount),
					fmt.Sprintf("time preservation = %t", a.opt.PreserveTime),
				}, ", "),
			)
		})

		func() { guard <- struct{}{}; wg.Add(1) }() // acquire a concurrency slot

		fileCounter++

		go func(fileCounter uint64, path string) {
			defer func() { <-guard; wg.Done() }()

			var filename = filepath.Base(path)

			stat, statErr := os.Stat(path)
			if statErr != nil {
				errs <- fmt.Errorf("failed to get the file info (%s): %w", filename, statErr)

				return
			}

			var fStat = fileStat{
				Path:     path,
				OrigSize: uint64(stat.Size()), //nolint:gosec
			}

			var comp *tinypng.Compressed

			for { // attempt file upload with retries if necessary
				client, revoke, clientFound := pool.Get()
				if !clientFound || client == nil { // no clients available in the pool
					errs <- errors.New("no valid API keys available")
					cancelIter() //nolint:wsl

					return
				}

				var cErr error

				comp, cErr = a.uploadFile(ctx, path, client)
				if cErr != nil {
					if errors.Is(cErr, tinypng.ErrUnauthorized) || errors.Is(cErr, tinypng.ErrTooManyRequests) {
						revoke() // revoke the client if it's unauthorized or rate-limited

						continue // try to get a new client and retry uploading the file
					}

					errs <- fmt.Errorf("failed to upload (%s): %w", filename, cErr)

					return
				}

				break // exit the loop if the file was uploaded successfully
			}

			fStat.CompSize = comp.Size
			fStat.Type = comp.Type

			// proceed only if compressed file meets criteria:
			// - compressed file size is not 0
			// - compressed file size is less than the original one
			// - the difference between the original and compressed file sizes is greater than N%
			if comp.Size == 0 ||
				int64(comp.Size) >= stat.Size() || //nolint:gosec
				((float64(stat.Size())-float64(comp.Size))/float64(comp.Size))*100 < a.opt.SkipIfDiffLessThan {
				fStat.Skipped = true
				stats.Add(fStat)

				return
			}

			var tmpFilePath = path + ".tiny"

			defer func() { // remove the temporary file if it exists
				if _, tmpStatErr := os.Stat(tmpFilePath); tmpStatErr == nil {
					_ = os.Remove(tmpFilePath)
				}
			}()

			// download the compressed file and save it to the temporary file
			if err := a.downloadCompressed(ctx, comp, tmpFilePath); err != nil {
				errs <- fmt.Errorf("failed to download the compressed file (%s): %w", filename, err)

				return
			}

			if err := a.replaceFiles(ctx, path, tmpFilePath); err != nil {
				errs <- fmt.Errorf("failed to replace (%s): %w", filename, err)

				return
			}

			a.logf(
				"%s File %s compressed (%s → %s / %s, %s)",
				func() string {
					if total := totalAmount.Load(); total > 0 {
						width := len(strconv.FormatUint(total, 10))

						return fmt.Sprintf("[%0*d/%d]", width, fileCounter, total)
					}

					return fmt.Sprintf("[%d/⏳]", fileCounter)
				}(),
				filename,
				humanize.Bytes(stat.Size()),
				humanize.Bytes(comp.Size),
				humanize.BytesDiff(comp.Size, stat.Size()),
				humanize.PercentageDiff(comp.Size, stat.Size()),
			)

			stats.Add(fStat)
		}(fileCounter, path)
	}

	wg.Wait()    // wait for all jobs to complete
	close(guard) // close the guard channel
	close(errs)  // close the errors channel to exit the waiting loop
	<-errsClosed // wait for the errors channel to be closed

	if table := stats.Table(); table != "" {
		a.logf("\n%s", table)
	}

	return ctx.Err()
}

// Step 1 is uploadFile - it uploads the file to the tinypng.com.
func (a *App) uploadFile(ctx context.Context, path string, c *tinypng.Client) (res *tinypng.Compressed, _ error) {
	return res, retry.Try(
		ctx,
		a.opt.RetryAttempts,
		func(context.Context, uint) error {
			f, err := os.OpenFile(path, os.O_RDONLY, 0)
			if err != nil {
				return err
			}

			defer func() { _ = f.Close() }()

			res, err = c.Compress(ctx, f)
			if err != nil {
				return err
			}

			return nil
		},
		retry.WithDelayBetweenAttempts(a.opt.DelayBetweenRetries),
		retry.WithStopOnError(tinypng.ErrUnauthorized, tinypng.ErrTooManyRequests),
	)
}

// Step 2 is downloadCompressed - it downloads the compressed file from the tinypng.com and saves it the provided path.
func (a *App) downloadCompressed(
	ctx context.Context,
	comp *tinypng.Compressed,
	path string,
) error {
	return retry.Try(
		ctx,
		a.opt.RetryAttempts,
		func(context.Context, uint) error {
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o644) //nolint:mnd
			if err != nil {
				return err
			}

			defer func() { _ = f.Close() }()

			var opts []tinypng.DownloadOption

			if a.opt.PreserveTime {
				opts = append(opts, tinypng.WithDownloadPreserveCreation())
			}

			return comp.Download(ctx, f, opts...)
		},
		retry.WithDelayBetweenAttempts(a.opt.DelayBetweenRetries),
		retry.WithStopOnError(tinypng.ErrUnauthorized, tinypng.ErrTooManyRequests),
	)
}

// Step 3 is replaceFiles - it replaces the original file content with the compressed one.
func (a *App) replaceFiles(ctx context.Context, origPath, compPath string) error {
	return retry.Try(
		ctx,
		a.opt.RetryAttempts,
		func(context.Context, uint) error {
			origStat, err := os.Stat(origPath)
			if err != nil {
				return err
			}

			comp, err := os.OpenFile(compPath, os.O_RDONLY, 0)
			if err != nil {
				return err
			}

			defer func() { _ = comp.Close() }()

			// make a copy of original file before replacing it, if needed
			if a.opt.KeepOriginalFile {
				orig, oErr := os.OpenFile(origPath, os.O_RDONLY, 0)
				if oErr != nil {
					return oErr
				}

				defer func() { _ = orig.Close() }()

				origCopy, oErr := os.OpenFile(orig.Name()+".orig", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, origStat.Mode().Perm())
				if oErr != nil {
					return fmt.Errorf("failed to create file for copy of the original file: %w", oErr)
				}

				defer func() { _ = origCopy.Close() }()

				if _, err = io.Copy(origCopy, orig); err != nil {
					_ = os.Remove(origCopy.Name()) // remove the copy if failed to write

					return err
				}

				_, _ = orig.Close(), origCopy.Close()
			}

			orig, err := os.OpenFile(origPath, os.O_WRONLY|os.O_TRUNC, 0)
			if err != nil {
				return err
			}

			defer func() { _ = orig.Close() }()

			if _, copyErr := io.Copy(orig, comp); copyErr != nil {
				return copyErr
			}

			_, _ = comp.Close(), orig.Close()

			if a.opt.PreserveTime {
				// restore original file modification date
				// atime: time of last access (ls -lu)
				// mtime: time of last modification (ls -l)
				_ = os.Chtimes(origPath, origStat.ModTime(), origStat.ModTime())
			}

			return nil
		},
		retry.WithDelayBetweenAttempts(a.opt.DelayBetweenRetries),
	)
}

func (a *App) logf(format string, args ...any) {
	a.logMu.Lock()
	defer a.logMu.Unlock()

	_, _ = fmt.Fprintf(os.Stdout, format+"\n", args...)
}

func (a *App) errorf(format string, args ...any) {
	a.logMu.Lock()
	defer a.logMu.Unlock()

	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
}

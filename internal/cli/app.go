package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gh.tarampamp.am/tinifier/v5/internal/cli/cmd"
	"gh.tarampamp.am/tinifier/v5/internal/config"
	"gh.tarampamp.am/tinifier/v5/internal/finder"
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
func (a *App) run(pCtx context.Context, paths []string) error { //nolint:gocognit,funlen,gocyclo
	var ctx, cancel = context.WithCancel(pCtx)
	defer cancel() // calling this function will cancel the context and stop the process

	var iterCtx, cancelIter = context.WithCancel(ctx)
	defer cancelIter() // calling this one will stop the iterator

	var (
		filesSeq    = finder.Files(iterCtx, paths, a.opt.Recursive, finder.FilterByExt(false, a.opt.FileExtensions...))
		totalAmount atomic.Uint64
	)

	// calculate the total number of files to process in background to avoid blocking the main process
	go func() {
		for range filesSeq { // due to iterator respect the context, we can iterate over it without any additional checks
			totalAmount.Add(1)
		}
	}()

	var (
		pool  = tinypng.NewClientsPool(a.opt.ApiKeys)
		guard = make(chan struct{}, max(1, a.opt.ThreadsCount))
		errs  = make(chan error, cap(guard))
	)

	// run background goroutine to process errors and stop the process if needed
	go func() {
		var counter uint

		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-errs:
				if !ok {
					return
				}

				counter++

				if a.opt.MaxErrorsToStop > 0 && counter >= a.opt.MaxErrorsToStop {
					a.logf("Maximum number of errors reached, stopping the process\n")

					cancelIter()
				}
			}
		}
	}()

	var wg sync.WaitGroup // wait group to wait for all jobs to complete

	for path := range filesSeq {
		func() { guard <- struct{}{}; wg.Add(1) }() // acquire a concurrency slot

		go func(path string) {
			defer func() { <-guard; wg.Done() }()

			a.logf("Processing %s\n", path)

			var comp *tinypng.Compressed

			for { // this loop is used to retry uploading the file if the client is revoked
				client, revoke, clientFound := pool.Get()
				if !clientFound || client == nil { // no clients available in the pool
					cancelIter()
					a.logf("No API keys available, stopping the process\n")

					return
				}

				var cErr error

				comp, cErr = a.uploadFile(ctx, path, client)
				if cErr != nil {
					if errors.Is(cErr, tinypng.ErrUnauthorized) || errors.Is(cErr, tinypng.ErrTooManyRequests) {
						revoke() // revoke the client if it's unauthorized or rate-limited

						continue // try to get a new client and retry uploading the file
					}

					a.logf("Failed to upload %s: %s\n", path, cErr)
					errs <- cErr

					return
				}

				break // exit the loop if the file was uploaded successfully
			}

			// check if the compressed file is not larger than the original one
			if compSize := int64(comp.Size); compSize != 0 { //nolint:gosec
				if stat, err := os.Stat(path); err == nil {
					if compSize > stat.Size() {
						a.logf("Compressed file is larger than the original one: %s\n", path)

						return
					}
				}
			}

			var tmpFilePath = path + ".tiny"

			defer func() { // remove the temporary file if it exists
				if _, err := os.Stat(tmpFilePath); err == nil {
					_ = os.Remove(tmpFilePath)
				}
			}()

			// download the compressed file and save it to the temporary file
			if err := a.downloadCompressed(ctx, comp, tmpFilePath); err != nil {
				a.logf("Failed to download the compressed file for %s: %s\n", path, err)
				errs <- err

				return
			}

			if err := a.replaceFiles(ctx, path, tmpFilePath); err != nil {
				a.logf("Failed to replace the original file with the compressed one: %s\n", err)
				errs <- err

				return
			}
		}(path)
	}

	wg.Wait()    // wait for all jobs to complete
	close(guard) // close the guard channel
	close(errs)  // close the errors channel

	return ctx.Err()
}

const (
	retryAttempts        uint = 3           // TODO: make it configurable?
	delayBetweenAttempts      = time.Second // TODO: make it configurable?
)

// Step 1 is uploadFile - it uploads the file to the tinypng.com.
func (a *App) uploadFile(ctx context.Context, path string, c *tinypng.Client) (res *tinypng.Compressed, _ error) {
	return res, retry.Try(
		ctx,
		retryAttempts,
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
		retry.WithDelayBetweenAttempts(delayBetweenAttempts),
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
		retryAttempts,
		func(context.Context, uint) error {
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o644) //nolint:mnd
			if err != nil {
				return err
			}

			defer func() { _ = f.Close() }()

			var opts []tinypng.DownloadOption

			if !a.opt.UpdateFileModDate {
				opts = append(opts, tinypng.WithDownloadPreserveCreation())
			}

			return comp.Download(ctx, f, opts...)
		},
		retry.WithDelayBetweenAttempts(delayBetweenAttempts),
		retry.WithStopOnError(tinypng.ErrUnauthorized, tinypng.ErrTooManyRequests),
	)
}

// Step 3 is replaceFiles - it replaces the original file content with the compressed one.
func (a *App) replaceFiles(ctx context.Context, origPath, compPath string) error {
	return retry.Try(ctx, retryAttempts, func(context.Context, uint) error {
		origStat, err := os.Stat(origPath)
		if err != nil {
			return err
		}

		comp, err := os.OpenFile(compPath, os.O_RDONLY, 0)
		if err != nil {
			return err
		}

		defer func() { _ = comp.Close() }()

		orig, err := os.OpenFile(origPath, os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil {
			return err
		}

		defer func() { _ = orig.Close() }()

		if _, copyErr := io.Copy(orig, comp); copyErr != nil {
			return copyErr
		}

		_, _ = comp.Close(), orig.Close()

		if !a.opt.UpdateFileModDate {
			// restore original file modification date
			// atime: time of last access (ls -lu)
			// mtime: time of last modification (ls -l)
			_ = os.Chtimes(origPath, origStat.ModTime(), origStat.ModTime())
		}

		return nil
	}, retry.WithDelayBetweenAttempts(delayBetweenAttempts))
}

func (a *App) logf(format string, args ...any) {
	a.logMu.Lock()
	defer a.logMu.Unlock()

	_, _ = fmt.Fprintf(os.Stdout, format, args...)
}

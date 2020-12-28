// Package compress contains CLI `compress` command implementation.
package compress

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tarampampam/tinifier/v3/internal/pkg/breaker"
	"github.com/tarampampam/tinifier/v3/internal/pkg/files"
	"github.com/tarampampam/tinifier/v3/internal/pkg/keys"
	"github.com/tarampampam/tinifier/v3/internal/pkg/pool"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	apiKeyEnvName         = "TINYPNG_API_KEY"
	apiKeyMinLength uint8 = 8
)

// NewCommand creates `compress` command.
func NewCommand(log *zap.Logger) *cobra.Command { //nolint:funlen
	var (
		apiKeys         []string
		fileExtensions  []string
		threadsCount    uint8
		maxErrorsToStop uint32
		recursive       bool
	)

	cmd := &cobra.Command{
		Use:     "compress <target-files-and-directories...>",
		Aliases: []string{"c"},
		Short:   "Compress images",
		Args:    cobra.MinimumNArgs(1),
		PreRunE: func(*cobra.Command, []string) error {
			if len(fileExtensions) < 1 {
				return errors.New("empty file extensions list")
			}

			if threadsCount < 1 {
				return errors.New("wrong threads value")
			}

			if len(apiKeys) == 0 {
				if envAPIKey, exists := os.LookupEnv(apiKeyEnvName); exists {
					apiKeys = append(apiKeys, envAPIKey)
				} else {
					return errors.New("API key was not provided")
				}
			}

			for i := 0; i < len(apiKeys); i++ {
				if uint8(len(apiKeys[i])) <= apiKeyMinLength {
					return fmt.Errorf("API key (%s) is too short", apiKeys[i])
				}
			}

			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			sources := files.FilterMissed(args)
			if len(sources) < 1 {
				return errors.New("nothing to compress (check your target lists)")
			}

			locator, err := files.NewLocator(sources, fileExtensions)
			if err != nil {
				return err // should be never occurs
			}

			targets, err := locator.Find(recursive)
			if err != nil {
				return err
			}

			if len(targets) < 1 {
				return errors.New("nothing to compress (double check your target and file extension lists)")
			}

			sort.Strings(targets)

			return execute(log, targets, apiKeys, threadsCount, maxErrorsToStop)
		},
	}

	cmd.Flags().StringSliceVarP(
		&apiKeys,   // var
		"api-key",  // name
		"k",        // short
		[]string{}, // default
		fmt.Sprintf(
			"TinyPNG API key <https://tinypng.com/dashboard/api> (multiple keys are allowed) [$%s]",
			apiKeyEnvName,
		),
	)

	cmd.Flags().StringSliceVarP(
		&fileExtensions, // var
		"ext",           // name
		"e",             // short
		[]string{"jpg", "JPG", "jpeg", "JPEG", "png", "PNG"}, // default
		"image file extensions (without leading dots)",
	)

	cmd.Flags().Uint8VarP(
		&threadsCount,           // var
		"threads",               // name
		"t",                     // short
		uint8(runtime.NumCPU()), // default
		"threads count",
	)

	cmd.Flags().Uint32VarP(
		&maxErrorsToStop, // var
		"max-errors",     // name
		"",               // short
		10,               // default
		"maximum errors count to stop the process, set 0 to disable",
	)

	cmd.Flags().BoolVarP(
		&recursive,  // var
		"recursive", // name
		"r",         // short
		false,       // default
		"search for files in listed directories recursively",
	)

	return cmd
}

// execute current command (compress passed files).
func execute( //nolint:funlen
	log *zap.Logger,
	targets []string,
	apiKeys []string,
	threadsCount uint8,
	maxErrorsToStop uint32,
) error {
	var (
		execErr     error
		execErrOnce sync.Once

		ctx, cancel = context.WithCancel(context.Background()) // main context creation
		startedAt   = time.Now()                               // save "started at" timestamp
		oss         = breaker.NewOSSignals(ctx)                // OS signals listener
	)

	oss.Subscribe(func(sig os.Signal) {
		log.Warn("Stopping by OS signal..", zap.String("signal", sig.String()))

		execErrOnce.Do(func() { execErr = errors.New("stopped by OS signal") })

		cancel() // we must to stop by OS signal
	})

	defer func() {
		cancel()   // call cancellation function after all for "service" goroutines stopping
		oss.Stop() // stop system signals listening
	}()

	keeper := keys.NewKeeper()
	if err := keeper.Add(apiKeys...); err != nil {
		return err
	}

	p := pool.NewPool(ctx, newWorker(
		log,
		&keeper,
		5,                    //nolint:gomnd
		time.Millisecond*700, //nolint:gomnd
	))

	var (
		errorsCounter uint32 // counter (atomic usage only)
		results       = p.Run(targets, threadsCount)
		reader        = NewResultsReader(os.Stdout) // results reader (pretty results writer)
	)

	for {
		result, isOpened := <-results
		if !isOpened {
			break
		}

		if err := result.Err; err != nil {
			if errors.Is(err, errNoAvailableAPIKey) {
				execErrOnce.Do(func() { execErr = errors.New("no one valid API key, working canceled") })

				cancel()
			}

			log.Error("Compression failed",
				zap.String("error", err.Error()),
				zap.String("file", result.Task.FilePath),
			)

			if count := atomic.AddUint32(&errorsCounter, 1); maxErrorsToStop > 0 && count >= maxErrorsToStop {
				log.Error(fmt.Sprintf("Too many (%d) errors occurred, stopping the process", count))

				execErrOnce.Do(func() { execErr = errors.New("too many errors occurred") })

				cancel() // too many errors occurred, we must to stop the process
			}
		} else {
			log.Debug(fmt.Sprintf("File \"%s\" compressed successful", result.FilePath),
				zap.Uint64("old size", result.OriginalSize),
				zap.Uint64("new size", result.CompressedSize),
			)

			reader.Append(result)
		}
	}

	reader.Draw()

	log.Info(fmt.Sprintf("Completed in %s", time.Since(startedAt)))

	return execErr
}

package compress

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/tarampampam/tinifier/internal/pkg/breaker"
	"github.com/tarampampam/tinifier/internal/pkg/files"
	"github.com/tarampampam/tinifier/internal/pkg/keys"
	"github.com/tarampampam/tinifier/internal/pkg/pipeline"
	"github.com/tarampampam/tinifier/internal/pkg/threadsafe"

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
		maxAPIKeyErrors uint32
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

			if maxAPIKeyErrors < 1 {
				return errors.New("wrong maximum API key errors value")
			}

			if len(apiKeys) == 0 {
				if envAPIKey := strings.Trim(os.Getenv(apiKeyEnvName), " "); envAPIKey != "" {
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

			return execute(log, targets, apiKeys, threadsCount, maxAPIKeyErrors, maxErrorsToStop)
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
		&fileExtensions,                                      // var
		"ext",                                                // name
		"e",                                                  // short
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

	cmd.Flags().Uint32VarP(
		&maxAPIKeyErrors, // var
		"max-key-errors", // name
		"",               // short
		3,                // default
		"maximum API key errors (compression retries) to disable the key",
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
	maxAPIKeyErrors uint32,
	maxErrorsToStop uint32,
) error {
	log.Debug("Running",
		zap.Strings("api keys", apiKeys),
		zap.Uint8("threads count", threadsCount),
		zap.Uint32("max api key errors", maxAPIKeyErrors),
		zap.Uint32("max errors", maxErrorsToStop),
		zap.Int("targets count", len(targets)),
	)

	keysKeeper := keys.NewKeeper(int(maxAPIKeyErrors))
	if err := keysKeeper.Add(apiKeys...); err != nil {
		return err
	}

	var (
		execError threadsafe.ErrorBag // for thread-safe "last error" returning TODO(jetexe) use channel len(1) instead this

		ctx, cancel = context.WithCancel(context.Background()) // main context creation
		startedAt   = time.Now()                               // save "started at" timestamp
		oss         = breaker.NewOSSignals(ctx)                // OS signals listener
	)

	oss.Subscribe(func(sig os.Signal) {
		log.Warn("Stopping by OS signal..", zap.String("signal", sig.String()))
		execError.Wrap(errors.New("stopped by OS signal"))

		cancel() // we must to stop by OS signal
	})

	defer func() {
		cancel()   // call cancellation function after all for "service" goroutines stopping
		oss.Stop() // stop system signals listening
	}()

	var (
		tasksCounter, errorsCounter uint32 // counters (atomic usage only)

		comp   = newCompressor(ctx, log, &keysKeeper)
		reader = newResultsReader(os.Stdout) // results reader (pretty results writer)
	)

	onError := func(err pipeline.TaskError) { // task errors handler
		count := atomic.AddUint32(&errorsCounter, 1)

		log.Error("Compression failed",
			zap.String("error", err.Error.Error()),
			zap.String("file", err.Task.FilePath),
		)

		if maxErrorsToStop > 0 && count >= maxErrorsToStop {
			log.Error(fmt.Sprintf("Too many (%d) errors occurred, stopping the process", count))
			execError.Set(errors.New("too many errors occurred"))

			cancel() // too many errors occurred, we must to stop the process
		}
	}

	onResult := func(res pipeline.TaskResult) { // task results handler
		log.Debug(fmt.Sprintf("File \"%s\" compressed successful", res.FilePath),
			zap.Uint64("old size", res.OriginalSize),
			zap.Uint64("new size", res.CompressedSize),
			zap.Uint64("used quota", res.UsedQuota),
		)

		reader.Append(res)
	}

	pipe := pipeline.NewPipeline(ctx, filePathsToTasks(targets), comp, onResult, onError)

	pipe.PreWorkerRun = func(task pipeline.Task) { // attach custom pre-worker-run handler
		log.Info(fmt.Sprintf(
			"[%d of %d] Compressing file \"%s\"",
			atomic.AddUint32(&tasksCounter, 1), len(targets), task.FilePath,
		))
	}

	<-pipe.Run(threadsCount) // wait until all jobs is done
	reader.Draw()            // draw results table

	log.Info(fmt.Sprintf("Completed in %s", time.Since(startedAt)))

	return execError.Get()
}

// filePathsToTasks converts slice with file paths into pipeline.Task's slice.
func filePathsToTasks(in []string) []pipeline.Task {
	result := make([]pipeline.Task, 0, len(in))

	for i := 0; i < len(in); i++ {
		result = append(result, pipeline.Task{FilePath: in[i]})
	}

	return result
}

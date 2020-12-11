package compress

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/tarampampam/tinifier/internal/pkg/files"
	"github.com/tarampampam/tinifier/pkg/tinypng"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	apiKeyEnvName            = "TINYPNG_API_KEY"
	apiKeyMinLength    uint8 = 8
	httpRequestTimeout       = time.Second * 80
)

type executeProperties struct {
	targets         []string
	apiKey          string
	threadsCount    uint8
	maxErrorsToStop uint32
}

// NewCommand creates `compress` command.
func NewCommand(log *logrus.Logger) *cobra.Command { //nolint:funlen
	var (
		apiKey          string
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

			if apiKey == "" {
				if envAPIKey := strings.Trim(os.Getenv(apiKeyEnvName), " "); envAPIKey != "" {
					apiKey = envAPIKey
				} else {
					return errors.New("API key was not provided")
				}
			}

			if uint8(len(apiKey)) <= apiKeyMinLength {
				return fmt.Errorf("API key (%s) is too short", apiKey)
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

			return execute(log, executeProperties{
				targets:         targets,
				apiKey:          apiKey,
				threadsCount:    threadsCount,
				maxErrorsToStop: maxErrorsToStop,
			})
		},
	}

	cmd.Flags().StringVarP(
		&apiKey,   // var
		"api-key", // name
		"k",       // short
		"",        // default
		fmt.Sprintf("TinyPNG API key <https://tinypng.com/dashboard/api> [$%s]", apiKeyEnvName),
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

type (
	task struct {
		filePath string
	}

	taskResult struct {
		fileType            string
		filePath            string
		originalSizeBytes   uint64
		compressedSizeBytes uint64
	}

	taskError struct {
		task task
		err  error
	}
)

// execute current command.
func execute(log *logrus.Logger, props executeProperties) (lastError error) { //nolint:funlen,gocognit
	log.WithFields(logrus.Fields{
		"api key":       props.apiKey,
		"threads count": props.threadsCount,
		"max errors":    props.maxErrorsToStop,
		// "targets":    props.targets,
		"targets count": len(props.targets),
	}).Debug("Running")

	// make a channel for system signals and "subscribe" for some of them
	signalsCh := make(chan os.Signal, 1)
	signal.Notify(signalsCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// main context creation
	ctx, cancel := context.WithCancel(context.Background())

	// cancel context on OS signal in separate goroutine
	go func(signalsCh <-chan os.Signal) {
		select {
		case sig := <-signalsCh:
			log.WithField("signal", sig).Warn("Stopping by OS signal..")

			lastError = errors.New("stopped by OS signal")

			cancel()

		case <-ctx.Done():
			break
		}
	}(signalsCh)

	var (
		startedAt = time.Now()
		channels  = struct {
			tasks   chan task
			results chan taskResult
			errors  chan taskError
		}{
			tasks:   make(chan task, props.threadsCount),
			results: make(chan taskResult, props.threadsCount),
			errors:  make(chan taskError),
		}
	)

	go func(errorsCh <-chan taskError, errorsLimit uint32) {
		var errorsCounter uint32

		for {
			compErr, isOpened := <-errorsCh
			if !isOpened { // channel is closed AND empty
				return
			}

			log.WithError(compErr.err).
				WithField("file", compErr.task.filePath).
				Error("Compression failed")

			if errorsLimit > 0 && errorsCounter >= errorsLimit {
				log.Errorf("Too many (%d) errors occurred, stopping the process", errorsCounter)

				cancel() // too many errors occurred, we must to stop the process
			}

			errorsCounter++
		}
	}(channels.errors, props.maxErrorsToStop)

	// fill-up tasks channel using single separate goroutine
	go func(tasksCh chan<- task) {
		defer close(tasksCh) // important

		for _, filePath := range props.targets {
			select {
			case <-ctx.Done():
				return

			default:
				tasksCh <- task{filePath: filePath}
			}
		}
	}(channels.tasks)

	var (
		workersWg    sync.WaitGroup
		tasksCounter uint32 // atomic usage
		tiny         = tinypng.NewClient(tinypng.ClientConfig{
			APIKey:         props.apiKey,
			RequestTimeout: httpRequestTimeout,
		})
	)

	// run workers (using many goroutines)
	for i := uint8(0); i < props.threadsCount; i++ {
		workersWg.Add(1)

		go func(tasksCh <-chan task, resultsCh chan<- taskResult, errorsCh chan<- taskError, tasksTotal int) {
			defer workersWg.Done()

			for {
				select {
				case <-ctx.Done():
					return

				default:
					t, isOpened := <-tasksCh // read task
					if !isOpened {
						return
					}

					log.Infof(
						"[%d of %d] Compressing file \"%s\"",
						atomic.AddUint32(&tasksCounter, 1), tasksTotal, t.filePath,
					)

					result, err := compressFile(ctx, tiny, t)

					if err != nil {
						errorsCh <- taskError{task: t, err: err}
					} else {
						resultsCh <- result
					}
				}
			}
		}(channels.tasks, channels.results, channels.errors, len(props.targets))
	}

	var resultsWg sync.WaitGroup

	resultsWg.Add(1)
	// read results using single separate goroutine
	go func() {
		defer resultsWg.Done()

		reader := newResultsReader()

		reader.Read(channels.results) // blocked here
		reader.Draw()
	}()

	workersWg.Wait()        // wait for workers completed state
	close(channels.results) // close results channel ("results reader" will stops when channel will be empty)
	close(channels.errors)
	signal.Stop(signalsCh) // stop os signals listening

	log.Infof("Completed in %s", time.Since(startedAt))

	resultsWg.Wait() // wait for "results reader" exiting
	cancel()         // cancel context anyway (important)

	return lastError
}

//nolint
func compressFile(ctx context.Context, tiny *tinypng.Client, task task) (taskResult, error) {
	// TODO write code (read file, compress, overwrite file)
	// defer time.Sleep(time.Millisecond * 1000)
	return taskResult{
		fileType:            "image/png",
		filePath:            task.filePath,
		originalSizeBytes:   111,
		compressedSizeBytes: 122,
	}, nil // errors.New("foo err")
}

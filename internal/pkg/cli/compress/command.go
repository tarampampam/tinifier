package compress

import (
	"context"
	"errors"
	"fmt"
	"io"
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

			err = execute(log, executeProperties{ // FIXME
				targets:         targets,
				apiKey:          apiKey,
				threadsCount:    threadsCount,
				maxErrorsToStop: maxErrorsToStop,
			})

			log.Warn(err)

			return err

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

// execute current command (compress passed files).
//
// Goroutines working schema:
//
//                         |---------|       results     |----------------|
//                         | |---------|   ------------> | results reader |
// |-----------|   tasks   |-| |---------|               |----------------|
// | scheduler |  -------->  |-| |---------|
// |-----------|               |-| |---------|  errors   |----------------|
//                               |-| workers | --------> | errors watcher |
//                                 |---------|           |----------------|
//
func execute(log *logrus.Logger, props executeProperties) error { //nolint:funlen,gocognit,gocyclo
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

	defer func() {
		signal.Stop(signalsCh)
		close(signalsCh)
	}()

	// main context creation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startedAt := time.Now() // save "started at" timestamp

	// cancel context on OS signal in separate goroutine
	go func(signalsCh <-chan os.Signal) {
		select {
		case sig, opened := <-signalsCh:
			if opened && sig != nil {
				log.WithField("signal", sig).Warn("Stopping by OS signal..")

				cancel()
			}

		case <-ctx.Done():
			break
		}
	}(signalsCh)

	tasksCh := make(chan task, props.threadsCount) // channel for compression tasks

	// fill-up tasks channel (schedule) using single separate goroutine
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
	}(tasksCh)

	errorsCh := make(chan taskError) // channel for compression task errors

	// start "errors watcher" in separate goroutine
	go func(errorsCh <-chan taskError, errorsLimit uint32) {
		var errorsCounter uint32

		for {
			taskErr, isOpened := <-errorsCh
			if !isOpened { // channel is closed AND empty
				return
			}

			log.WithError(taskErr.err).
				WithField("file", taskErr.task.filePath).
				Error("Compression failed")

			if errorsLimit > 0 && errorsCounter >= errorsLimit {
				log.Errorf("Too many (%d) errors occurred, stopping the process", errorsCounter)

				cancel() // too many errors occurred, we must to stop the process
			}

			errorsCounter++
		}
	}(errorsCh, props.maxErrorsToStop)

	var (
		workersWg    sync.WaitGroup
		tasksCounter uint32 // atomic usage only
		resultsCh    = make(chan taskResult, props.threadsCount)
	)

	// run workers (using many goroutines)
	for i := uint8(0); i < props.threadsCount; i++ {
		workersWg.Add(1)

		go func(tasksCh <-chan task, resultsCh chan<- taskResult, errorsCh chan<- taskError, total int, key string) {
			defer workersWg.Done()

			tiny := tinypng.NewClient(tinypng.ClientConfig{
				APIKey:         key,
				RequestTimeout: httpRequestTimeout,
			})

			for {
				select {
				case <-ctx.Done():
					return

				case t, isOpened := <-tasksCh: // read task
					if !isOpened {
						return
					}

					log.Infof(
						"[%d of %d] Compressing file \"%s\"",
						atomic.AddUint32(&tasksCounter, 1), total, t.filePath,
					)

					result, err := compressFile(ctx, tiny, t)

					if err != nil {
						errorsCh <- taskError{task: t, err: err}
					} else {
						resultsCh <- *result
					}
				}
			}
		}(tasksCh, resultsCh, errorsCh, len(props.targets), props.apiKey)
	}

	var resultsWg sync.WaitGroup

	resultsWg.Add(1)
	// read results using single separate goroutine
	go func(resultsCh <-chan taskResult) {
		reader := newResultsReader()

		defer func() {
			reader.Draw()
			resultsWg.Done()
		}()

		for {
			result, isOpened := <-resultsCh
			if !isOpened { // channel is closed AND empty
				return
			}

			log.WithFields(logrus.Fields{
				"old size": result.originalSizeBytes,
				"new size": result.compressedSizeBytes,
			}).Debugf("File \"%s\" compressed successful", result.filePath)

			reader.Append(result)
		}
	}(resultsCh)

	workersWg.Wait() // wait for workers completed state
	close(resultsCh) // close results channel ("results reader" will stops when channel will be empty)
	close(errorsCh)  // close errors channel
	resultsWg.Wait() // wait for "results reader" exiting

	// FIXME exit code on context canceling

	log.Infof("Completed in %s", time.Since(startedAt))

	return nil
}

// compressFile reads file from passed task, compress them using tinypng client, and overwrite original file with
// compressed image content.
func compressFile(ctx context.Context, tiny *tinypng.Client, task task) (*taskResult, error) {
	fileRead, err := os.OpenFile(task.filePath, os.O_RDONLY, 0) // open file for reading
	if err != nil {
		return nil, err
	}

	stat, err := fileRead.Stat()
	if err != nil {
		fileRead.Close() // do not forget to close file

		return nil, err
	}

	resp, err := tiny.Compress(ctx, fileRead)

	fileRead.Close() // file was compressed (successful or not), and must be closed

	if err != nil {
		return nil, err
	}

	defer resp.Compressed.Close()

	fileWrite, err := os.OpenFile(task.filePath, os.O_WRONLY|os.O_TRUNC, stat.Mode()) // open file for writing
	if err != nil {
		return nil, err
	}

	defer fileWrite.Close()

	_, err = io.Copy(fileWrite, resp.Compressed)
	if err != nil {
		return nil, err
	}

	return &taskResult{
		fileType:            resp.Output.Type,
		filePath:            task.filePath,
		originalSizeBytes:   resp.Input.Size,
		compressedSizeBytes: resp.Output.Size,
	}, nil
}

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
	"syscall"
	"time"

	"github.com/tarampampam/tinifier/internal/pkg/files"
	"github.com/tarampampam/tinifier/pkg/tinypng"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	apiKeyEnvName                    = "TINYPNG_API_KEY"
	apiKeyMinLength    uint8         = 8
	httpRequestTimeout time.Duration = time.Second * 80
)

type executeProperties struct {
	targets      []string
	apiKey       string
	threadsCount uint
}

// NewCommand creates `compress` command.
func NewCommand(log *logrus.Logger) *cobra.Command { //nolint:funlen
	var (
		apiKey         string
		fileExtensions []string
		threadsCount   uint
		recursive      bool
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
				targets:      targets,
				apiKey:       apiKey,
				threadsCount: threadsCount,
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

	cmd.Flags().UintVarP(
		&threadsCount,          // var
		"threads",              // name
		"t",                    // short
		uint(runtime.NumCPU()), // default
		"threads count",
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
	fileCompressionTask struct {
		filePath string
	}

	fileCompressionResult struct {
		error               error
		fileType            string
		filePath            string
		originalSizeBytes   uint64
		compressedSizeBytes uint64
	}

	compressionStatistic struct {
		mu sync.RWMutex

		originalBytes   uint64
		compressedBytes uint64
		savedBytes      int64
		totalFiles      uint32
	}
)

// execute current command.
func execute(log *logrus.Logger, props executeProperties) error { //nolint:funlen
	log.WithFields(logrus.Fields{
		"api key":       props.apiKey,
		"threads count": props.threadsCount,
		"targets":       props.targets,
		"targets count": len(props.targets),
	}).Debug("Running")

	log.Infof("To compress: %d files", len(props.targets))

	// make a channel for system signals and "subscribe" for some of them
	signalsCh := make(chan os.Signal, 1)
	signal.Notify(signalsCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// main context creation
	ctx, cancel := context.WithCancel(context.Background())

	// cancel context on OS signal in separate goroutine
	go func(ctx context.Context, signalsCh <-chan os.Signal) {
		select {
		case sig := <-signalsCh:
			log.WithField("signal", sig).Warn("Stopping by OS signal..")
			cancel()

		case <-ctx.Done():
			break
		}
	}(ctx, signalsCh)

	var (
		tasksCh   = make(chan fileCompressionTask, props.threadsCount)
		resultsCh = make(chan fileCompressionResult, props.threadsCount)
	)

	// fill-up tasks channel using single separate goroutine
	go func(ctx context.Context, tasksCh chan<- fileCompressionTask) {
		defer close(tasksCh) // important

		for _, filePath := range props.targets {
			select {
			case <-ctx.Done():
				return

			default:
				tasksCh <- fileCompressionTask{filePath: filePath}
			}
		}
	}(ctx, tasksCh)

	var (
		workersWg = new(sync.WaitGroup)
		tiny      = tinypng.NewClient(tinypng.ClientConfig{
			APIKey:         props.apiKey,
			RequestTimeout: httpRequestTimeout,
		})
	)

	// run workers (using many goroutines)
	for i := uint(0); i < props.threadsCount; i++ {
		workersWg.Add(1)

		go func(ctx context.Context, tasksCh <-chan fileCompressionTask, resultsCh chan<- fileCompressionResult) {
			defer workersWg.Done()

			for {
				select {
				case <-ctx.Done():
					return

				default:
					task, isOpened := <-tasksCh // read task
					if !isOpened {
						return
					}

					resultsCh <- compressFile(ctx, tiny, task)
				}
			}
		}(ctx, tasksCh, resultsCh)
	}

	var (
		resultsWg = new(sync.WaitGroup)
		stats     = compressionStatistic{mu: sync.RWMutex{}}
	)

	resultsWg.Add(1)
	// read results using single separate goroutine
	go func(resultsCh <-chan fileCompressionResult, stats *compressionStatistic) {
		defer resultsWg.Done()

		for {
			result, isOpened := <-resultsCh // read result
			if !isOpened {
				return
			}

			stats.mu.Lock()
			stats.originalBytes += result.originalSizeBytes
			stats.compressedBytes += result.compressedSizeBytes
			stats.savedBytes += int64(result.originalSizeBytes - result.compressedSizeBytes)
			stats.totalFiles++
			stats.mu.Unlock()
			// TODO write code (show progress bar, log, etc)
		}
	}(resultsCh, &stats)

	workersWg.Wait()
	close(resultsCh)
	resultsWg.Wait()

	return printResults(&stats)
}

func compressFile(ctx context.Context, tiny *tinypng.Client, task fileCompressionTask) fileCompressionResult {
	// TODO write code (read file, compress, overwrite file)
	return fileCompressionResult{}
}

func printResults(stats *compressionStatistic) error {
	// TODO write code (show stats in a table)
	return nil
}

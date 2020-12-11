package quota

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tarampampam/tinifier/pkg/tinypng"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	apiKeyEnvName                    = "TINYPNG_API_KEY"
	apiKeyMinLength    uint8         = 8
	httpRequestTimeout time.Duration = time.Second * 5
)

// NewCommand creates `quota` command.
func NewCommand(log *logrus.Logger) *cobra.Command {
	var APIKey string

	cmd := &cobra.Command{
		Use:     "quota",
		Aliases: []string{"q"},
		Short:   "Get currently used quota",
		PreRunE: func(*cobra.Command, []string) error {
			if APIKey == "" {
				if envAPIKey := strings.Trim(os.Getenv(apiKeyEnvName), " "); envAPIKey != "" {
					APIKey = envAPIKey
				} else {
					return errors.New("API key was not provided")
				}
			}

			if uint8(len(APIKey)) <= apiKeyMinLength {
				return fmt.Errorf("API key (%s) is too short", APIKey)
			}

			return nil
		},
		RunE: func(*cobra.Command, []string) error {
			return execute(log, APIKey)
		},
	}

	cmd.Flags().StringVarP(
		&APIKey,   // var
		"api-key", // name
		"k",       // short
		"",        // default
		fmt.Sprintf("TinyPNG API key <https://tinypng.com/dashboard/api> [$%s]", apiKeyEnvName),
	)

	return cmd
}

type result struct { // FIXME
	count uint64
	err   error
}

// execute current command.
func execute(log *logrus.Logger, apiKey string) (lastError error) {
	log.WithField("api key", apiKey).Debug("Running")

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

			lastError = errors.New("stopped by OS signal")

			cancel()

		case <-ctx.Done():
			break
		}
	}(ctx, signalsCh)

	// client creation
	client := tinypng.NewClient(tinypng.ClientConfig{
		APIKey:         apiKey,
		RequestTimeout: httpRequestTimeout,
	})

	// result channel
	resultCh := make(chan result)

	// execute client call in separate goroutine
	go func(out chan<- result) {
		count, err := client.GetCompressionCount(ctx)
		if err != nil {
			out <- result{err: err}

			return
		}

		out <- result{count: count}
	}(resultCh)

	// ...and wait for result
	res := <-resultCh

	close(resultCh)
	cancel()
	signal.Stop(signalsCh)
	close(signalsCh)

	if res.err != nil {
		return res.err
	}

	fmt.Printf("Used quota is: %d\n", res.count)

	return lastError
}

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
	apiKeyEnvName      string = "TINYPNG_API_KEY"
	apiKeyMinLength    uint8  = 8
	httpRequestTimeout        = time.Second * 5
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

// execute current command.
func execute(log *logrus.Logger, apiKey string) error { //nolint:funlen
	log.WithField("api key", apiKey).Debug("Running")

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

	countCh, errCh := make(chan uint64), make(chan error)

	// execute client call in separate goroutine
	go func(countCh chan<- uint64, errCh chan<- error, apiKey string) {
		defer func() {
			close(countCh)
			close(errCh)
		}()

		client := tinypng.NewClient(tinypng.ClientConfig{
			APIKey:         apiKey,
			RequestTimeout: httpRequestTimeout,
		})

		count, err := client.GetCompressionCount(ctx)
		if err != nil {
			errCh <- err

			return
		}

		countCh <- count
	}(countCh, errCh, apiKey)

	// and wait for results (or context canceling)
	select {
	case count := <-countCh:
		fmt.Printf("Used quota is: %d\n", count)

	case err := <-errCh:
		return err

	case <-ctx.Done():
		return errors.New("working canceled")
	}

	return nil
}

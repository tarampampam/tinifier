// Package quota contains CLI `quota` command implementation.
package quota

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tarampampam/tinifier/v3/internal/pkg/breaker"
	"github.com/tarampampam/tinifier/v3/internal/pkg/env"
	"github.com/tarampampam/tinifier/v3/pkg/tinypng"
	"go.uber.org/zap"
)

// NewCommand creates `quota` command.
func NewCommand(log *zap.Logger) *cobra.Command {
	var APIKey string

	cmd := &cobra.Command{
		Use:     "quota",
		Aliases: []string{"q"},
		Short:   "Get currently used quota",
		PreRunE: func(*cobra.Command, []string) error {
			if APIKey == "" {
				if envAPIKey, exists := env.TinyPngAPIKey.Lookup(); exists {
					APIKey = envAPIKey
				} else {
					return errors.New("API key was not provided")
				}
			}

			const apiKeyMinLength = 8

			if len(APIKey) <= apiKeyMinLength {
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
		fmt.Sprintf("TinyPNG API key <https://tinypng.com/dashboard/api> [$%s]", env.TinyPngAPIKey.String()),
	)

	return cmd
}

// execute current command.
func execute(log *zap.Logger, apiKey string) error {
	log.Debug("Running", zap.String("api key", apiKey))

	var (
		ctx, cancel = context.WithCancel(context.Background()) // main context creation
		oss         = breaker.NewOSSignals(ctx)                // OS signals listener
	)

	oss.Subscribe(func(sig os.Signal) {
		log.Warn("Stopping by OS signal..", zap.String("signal", sig.String()))

		cancel()
	})

	defer func() {
		cancel()   // call cancellation function after all for "service" goroutines stopping
		oss.Stop() // stop system signals listening
	}()

	countCh, errCh := make(chan uint64), make(chan error)

	const httpRequestTimeout = time.Second * 5

	// execute client call in separate goroutine
	go func(countCh chan<- uint64, errCh chan<- error, apiKey string) {
		defer func() { close(countCh); close(errCh) }()

		client := tinypng.NewClient(apiKey, tinypng.WithContext(ctx))

		count, err := client.CompressionCount(httpRequestTimeout)
		if err != nil {
			errCh <- err

			return
		}

		countCh <- count
	}(countCh, errCh, apiKey)

	// and wait for results (or context canceling)
	select {
	case count := <-countCh:
		_, _ = fmt.Fprintf(os.Stdout, "Used quota is: %d\n", count)

	case err := <-errCh:
		return err

	case <-ctx.Done():
		return errors.New("working canceled")
	}

	return nil
}

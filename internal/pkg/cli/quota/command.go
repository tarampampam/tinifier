package quota

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tarampampam/tinifier/internal/pkg/breaker"
	"github.com/tarampampam/tinifier/pkg/tinypng"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	apiKeyEnvName      string = "TINYPNG_API_KEY"
	apiKeyMinLength    uint8  = 8
	httpRequestTimeout        = time.Second * 5
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
				if envAPIKey := strings.Trim(os.Getenv(apiKeyEnvName), " "); envAPIKey != "" { // TODO(jetexe) os.LookupEnv
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
func execute(log *zap.Logger, apiKey string) error { //nolint:funlen
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
		fmt.Fprintf(os.Stdout, "Used quota is: %d\n", count)

	case err := <-errCh:
		return err

	case <-ctx.Done():
		return errors.New("working canceled")
	}

	return nil
}

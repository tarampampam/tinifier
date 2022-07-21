// Package quota contains CLI `quota` command implementation.
package quota

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/tarampampam/tinifier/v4/internal/breaker"
	"github.com/tarampampam/tinifier/v4/internal/env"
	"github.com/tarampampam/tinifier/v4/internal/logger"
	"github.com/tarampampam/tinifier/v4/pkg/tinypng"
)

// NewCommand creates `quota` command.
func NewCommand(log logger.Logger) *cli.Command {
	const (
		apiKeyFlagName  = "api-key"
		apiKeyMinLength = 8
	)

	return &cli.Command{
		Name:    "quota",
		Aliases: []string{"q"},
		Usage:   "Get currently used quota",
		Action: func(c *cli.Context) error {
			var apiKey = c.String(apiKeyFlagName)

			if len(apiKey) <= apiKeyMinLength {
				return fmt.Errorf("API key (%s) is too short", apiKey)
			}

			log.Debug("Running", "api_key="+apiKey)

			var (
				ctx, cancel = context.WithCancel(c.Context) // main context creation
				oss         = breaker.NewOSSignals(ctx)     // OS signals listener
			)

			oss.Subscribe(func(sig os.Signal) {
				log.Warn("Stopping by OS signal..", "signal="+sig.String())

				cancel()
			})

			defer func() {
				cancel()   // call cancellation function after all for "service" goroutines stopping
				oss.Stop() // stop system signals listening
			}()

			if count, err := tinypng.NewClient(apiKey).UsedQuota(ctx); err != nil {
				return err
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "Used quota is: %d\n", count)
			}

			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    apiKeyFlagName,
				Aliases: []string{"k"},
				Usage:   "TinyPNG API key <https://tinypng.com/dashboard/api>",
				EnvVars: []string{env.TinyPngAPIKey.String()},
			},
		},
	}
}

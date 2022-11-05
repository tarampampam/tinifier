// Package quota contains CLI `quota` command implementation.
package quota

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/tarampampam/tinifier/v4/internal/breaker"
	"github.com/tarampampam/tinifier/v4/internal/cli/shared"
	"github.com/tarampampam/tinifier/v4/pkg/tinypng"
)

// NewCommand creates `quota` command.
func NewCommand() *cli.Command { //nolint:funlen
	const (
		apiKeyMinLength = 8
	)

	return &cli.Command{
		Name:    "quota",
		Aliases: []string{"q"},
		Usage:   "Get currently used quota",
		Action: func(c *cli.Context) error {
			var apiKeys = c.StringSlice(shared.APIKeyFlag.Name)

			if len(apiKeys) == 0 {
				return errors.New("API key(s) was not provided")
			}

			var (
				ctx, cancel = context.WithCancel(c.Context) // main context creation
				oss         = breaker.NewOSSignals(ctx)     // OS signals listener
			)

			oss.Subscribe(func(os.Signal) { cancel() })

			defer func() {
				cancel()   // call cancellation function after all for "service" goroutines stopping
				oss.Stop() // stop system signals listening
			}()

			var (
				wg       sync.WaitGroup
				errColor = text.Colors{text.FgRed, text.Bold}
			)

			for _, key := range apiKeys {
				if len(key) <= apiKeyMinLength {
					_, _ = fmt.Fprint(os.Stderr, errColor.Sprintf("API key (%s) is too short\n", key))

					continue
				}

				wg.Add(1)

				go func(key string) {
					defer wg.Done()

					if ctx.Err() != nil { // check if context was canceled
						return
					}

					if count, err := tinypng.NewClient(key).UsedQuota(ctx); err != nil {
						_, _ = fmt.Fprint(os.Stderr, errColor.Sprintf("Key %s error: %s\n", maskString(key), err))

						return
					} else {
						var color = text.FgRed

						switch {
						case count <= 300: //nolint:gomnd
							color = text.FgGreen

						case count <= 400: //nolint:gomnd
							color = text.FgYellow
						}

						_, _ = fmt.Fprintf(os.Stdout,
							"Used quota (key %s) is: %s\n",
							text.Colors{text.FgHiBlue}.Sprint(maskString(key)),
							text.Colors{color, text.Bold}.Sprintf("%d", count),
						)
					}
				}(key)
			}

			wg.Wait()

			return nil
		},
		Flags: []cli.Flag{
			shared.APIKeyFlag,
		},
	}
}

// maskString returns masked string with same length as input string.
func maskString(s string) string {
	var length = utf8.RuneCountInString(s)

	if length <= 8 { //nolint:gomnd
		return s
	}

	return s[:4] + strings.Repeat("*", length-8) + s[length-4:] //nolint:gomnd
}

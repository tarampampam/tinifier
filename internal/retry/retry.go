// Package retry allows to use retry mechanism (call some function until it does not return non-error result or
// max attempts count is not exceeded).
package retry

import (
	"context"
	"math"
	"time"

	"github.com/pkg/errors"
)

type Option func(*config)

// WithContext overrides default context (by default `context.Background()` is used).
func WithContext(ctx context.Context) Option { return func(c *config) { c.ctx = ctx } }

// Attempts sets the maximum allowed retries count (set 0 to disable the limitation at all).
func Attempts(attempts uint) Option { return func(c *config) { c.maxAttempts = attempts } }

// Delay overrides default retry delay.
func Delay(delay time.Duration) Option { return func(c *config) { c.delay = delay } }

// StopOnError allows stopping the retry loop when one of the defined errors returned from the calling function.
func StopOnError(e ...error) Option {
	return func(c *config) { c.stopOnErrors = append(c.stopOnErrors, e...) }
}

type config struct {
	maxAttempts  uint
	delay        time.Duration
	ctx          context.Context
	stopOnErrors []error
}

const (
	noDelay = time.Duration(0)

	defaultAttemptsCount uint = 3       // default maxAttempts count
	defaultAttemptDelay       = noDelay // default delay between maxAttempts
)

// Do execute passed function and repeat it until non-error returned or maximum retries maxAttempts count in not
// exceeded, attempts counter starts from 1.
//
// The default attempts count is 3 (without delay between attempts).
func Do(fn func(attemptNum uint) error, options ...Option) (limitExceeded bool, lastErr error) {
	cfg := config{
		ctx:         context.Background(),
		delay:       defaultAttemptDelay,
		maxAttempts: defaultAttemptsCount,
	}

	// apply passed options to the configuration
	for _, option := range options {
		option(&cfg)
	}

	// enables infinite retry loop if maxAttempts count is 0
	if cfg.maxAttempts == 0 {
		cfg.maxAttempts = math.MaxUint // in fact, we have a limit, but very-very large
	}

	var timer *time.Timer // timer is used for delay between maxAttempts without thread blocking

	defer func() {
		if timer != nil {
			timer.Stop() // stop the timer on exit (in efficiency reasons)
		}
	}()

	for attemptNum := uint(1); attemptNum <= cfg.maxAttempts; attemptNum++ {
		limitExceeded = attemptNum >= cfg.maxAttempts

		// check for context cancellation
		if err := cfg.ctx.Err(); err != nil {
			lastErr = err

			return
		}

		// execute passed function
		lastErr = fn(attemptNum)

		// if function executed without any error - stop the loop
		if lastErr == nil {
			return
		}

		// otherwise, if "errors to stop" is defined and one of them is occurred - stop the loop
		if len(cfg.stopOnErrors) > 0 {
			for i := 0; i < len(cfg.stopOnErrors); i++ {
				if errors.Is(lastErr, cfg.stopOnErrors[i]) { // checking using errors.Is(...) is important
					return
				}
			}
		}

		if cfg.delay > noDelay {
			// create (and start immediately) or reset the timer
			if timer == nil {
				timer = time.NewTimer(cfg.delay)
			} else {
				timer.Reset(cfg.delay)
			}

			// and blocks until context is done or timer is expired
			select {
			case <-cfg.ctx.Done():
				lastErr = cfg.ctx.Err()

				return

			case <-timer.C:
			}
		}
	}

	return limitExceeded, lastErr
}

package retry

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type (
	options struct {
		DelayBetweenAttempts time.Duration // duration to wait between retry attempts
		StopOnError          []error       // list of errors that should stop the retry loop immediately
	}

	// Option represents a functional option for configuring retry behavior.
	Option func(*options)
)

// Apply applies the given options to the current options struct and returns the modified options.
func (o options) Apply(opts ...Option) options {
	for _, opt := range opts {
		opt(&o)
	}

	return o
}

// WithDelayBetweenAttempts sets the delay duration between consecutive retry attempts.
func WithDelayBetweenAttempts(d time.Duration) Option {
	return func(o *options) { o.DelayBetweenAttempts = d }
}

// WithStopOnError sets the list of errors that should immediately stop the retry loop when encountered.
func WithStopOnError(e ...error) Option {
	return func(o *options) { o.StopOnError = append(o.StopOnError, e...) }
}

// ErrRetryAttemptsExceeded is returned when the function exceeds the allowed number of retry attempts.
var ErrRetryAttemptsExceeded = errors.New("retry attempts exceeded")

// Try executes the given function `fn` until it succeeds, the maximum number of attempts is reached,
// or a stop condition is met.
//
// If `fn` returns an error that matches any error in StopOnError, the retry loop stops immediately.
// If the maximum number of attempts is reached, the function returns ErrRetryAttemptsExceeded, wrapped
// with the last encountered error (you need to use [errors.Is] to check for this error).
// If the context is canceled, the function returns the context error.
// If WithDelayBetweenAttempts is set, the function waits for the specified duration before retrying.
//
// It returns nil if `fn` succeeds within the allowed attempts, or an error if the function ultimately fails,
// either due to reaching the maximum attempts or encountering a stop condition.
func Try(
	ctx context.Context,
	attempts uint, // the maximum number of times to retry before giving up
	fn func(_ context.Context, attempt uint) error, // the function to execute, current attempt number starts from 1
	opts ...Option, // optional settings for configuring retry behavior
) error {
	var (
		o       = options{}.Apply(opts...)
		attempt uint
		timer   *time.Timer
	)

	// initialize the timer if a delay between attempts is specified
	if o.DelayBetweenAttempts > time.Duration(0) {
		timer = time.NewTimer(o.DelayBetweenAttempts)
		defer timer.Stop()
	}

	for {
		// check if the context was canceled before attempting the function
		if err := ctx.Err(); err != nil {
			return err
		}

		attempt++

		// execute the function and check for errors
		if err := fn(ctx, attempt); err != nil {
			// if the maximum number of attempts is reached, return ErrRetryAttemptsExceeded
			if attempt >= attempts {
				return fmt.Errorf("%w: %w", ErrRetryAttemptsExceeded, err)
			}

			// if a delay is specified, wait for the next attempt unless the context is canceled
			if timer != nil {
				timer.Reset(o.DelayBetweenAttempts)

				select {
				case <-ctx.Done():
					return ctx.Err() // return immediately if context is canceled
				case <-timer.C:
				}
			}

			// check if the error matches any in StopOnError; if so, exit immediately
			for _, errToStop := range o.StopOnError {
				if errors.Is(err, errToStop) {
					return err
				}
			}

			continue
		}

		// return nil if `fn` succeeds
		return nil
	}
}

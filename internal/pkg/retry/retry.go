package retry

import (
	"context"
	"errors"
	"time"
)

type Option func(*config)

// WithContext overrides default context (by default `context.Background()` is used).
func WithContext(ctx context.Context) Option { return func(c *config) { c.ctx = ctx } }

// WithAttempts overrides default attempts count.
func WithAttempts(attempts uint) Option { return func(c *config) { c.attempts = attempts } }

// WithDelay overrides default retry delay.
func WithDelay(delay time.Duration) Option { return func(c *config) { c.delay = delay } }

// WithLastErrorReturning allows to return last execution error (instead ErrToManyAttempts or context error).
func WithLastErrorReturning() Option { return func(c *config) { c.returnLastError = true } }

// WithRetryStoppingErrors allows to thor the retry loop when defined error returned from calling function.
func WithRetryStoppingErrors(e ...error) Option { return func(c *config) { c.loopStoppingErrors = e } }

type config struct {
	attempts           uint
	delay              time.Duration
	ctx                context.Context
	returnLastError    bool
	loopStoppingErrors []error
}

const (
	defaultAttemptsCount uint = 3                // default attempts count
	defaultAttemptDelay       = time.Millisecond // default delay between attempts
)

var (
	ErrToManyAttempts = errors.New("too many attempts") // Too many retry attempts exceeded.
	ErrNoAttempts     = errors.New("no attempts")       // No retry attempts.
	ErrRetryStopped   = errors.New("retry attempts was stopped")
)

// Do executes passed function and repeat it until non-error returned or maximum retries attempts count in not exceeded.
// Attempts counter starts from 1.
func Do(fn func(attemptNum uint) error, options ...Option) error { //nolint:funlen,gocyclo
	cfg := config{
		ctx:      context.Background(),
		delay:    defaultAttemptDelay,
		attempts: defaultAttemptsCount,
	}

	for _, option := range options {
		option(&cfg)
	}

	if cfg.attempts <= 0 {
		return ErrNoAttempts
	}

	var (
		timer       *time.Timer
		attemptErr  error
		loopStopped bool
	)

	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

loop:
	for attemptNum := uint(1); attemptNum <= cfg.attempts; attemptNum++ {
		if err := cfg.ctx.Err(); err != nil {
			return err
		}

		attemptErr = fn(attemptNum)
		if attemptErr == nil {
			return nil
		} else if cfg.loopStoppingErrors != nil {
			for i := 0; i < len(cfg.loopStoppingErrors); i++ {
				if errors.Is(attemptErr, cfg.loopStoppingErrors[i]) {
					loopStopped = true

					break loop
				}
			}
		}

		if timer == nil {
			timer = time.NewTimer(cfg.delay)
		} else {
			timer.Reset(cfg.delay)
		}

		select {
		case <-cfg.ctx.Done():
			return cfg.ctx.Err()

		case <-timer.C:
		}
	}

	if cfg.returnLastError && attemptErr != nil {
		return attemptErr
	} else if loopStopped {
		return ErrRetryStopped
	}

	return ErrToManyAttempts
}

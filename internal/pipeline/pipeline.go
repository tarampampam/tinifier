package pipeline

import (
	"context"
	"errors"
	"iter"
	"sync"
)

// ErrTooManyErrors is returned when the pipeline encounters too many errors, exceeding the configured threshold.
var ErrTooManyErrors = errors.New("too many errors")

// options holds configuration settings for the pipeline.
type (
	options struct {
		MaxParallel     int  // maximum number of inputs to process concurrently
		RetryAttempts   int  // number of retry attempts per step on failure
		MaxErrorsToStop uint // error threshold that triggers pipeline cancellation (0 means no limit)
	}

	// Option is a function type that modifies pipeline options.
	Option func(*options)
)

// Apply applies the given options to the options struct and returns a new options set.
func (o options) Apply(opts []Option) options {
	for _, opt := range opts {
		opt(&o)
	}

	return o
}

// WithMaxParallel sets the maximum number of inputs to process concurrently.
func WithMaxParallel(v int) Option { return func(o *options) { o.MaxParallel = v } }

// WithRetryAttempts sets the number of retry attempts per step on failure.
func WithRetryAttempts(v int) Option { return func(o *options) { o.RetryAttempts = v } }

// WithMaxErrorsToStop sets the error threshold that triggers pipeline cancellation.
func WithMaxErrorsToStop(v uint) Option { return func(o *options) { o.MaxErrorsToStop = v } }

// ThreeSteps executes a three-step pipeline for processing a sequence of inputs.
// It takes a parent context, three processing steps, an input sequence, and optional configuration options.
// Returns a sequence of results and an error channel.
func ThreeSteps[T1, T2, T3, T4 any]( //nolint:gocognit,gocyclo,funlen
	pCtx context.Context, // parent context for the pipeline
	step1 func(context.Context, T1) (*T2, error), // first processing step
	step2 func(context.Context, T2) (*T3, error), // second processing step
	step3 func(context.Context, T3) (*T4, error), // third processing step
	inputs iter.Seq[T1], // slice of input values to process
	opts ...Option, // optional pipeline configuration
) (iter.Seq2[T4, error], <-chan error) {
	switch {
	case inputs == nil:
		errCh := make(chan error)
		close(errCh)

		return nil, errCh
	case pCtx == nil:
		return nil, newErrChan(errors.New("ctx must not be nil"))
	case step1 == nil || step2 == nil || step3 == nil:
		return nil, newErrChan(errors.New("all steps must not be nil"))
	}

	// create a new context that allows pipeline cancellation
	ctx, cancel := context.WithCancel(pCtx)

	// result struct holds processed values or errors
	type result[T any] struct {
		Value T
		Err   error
	}

	var (
		opt     = options{}.Apply(opts) // apply pipeline options
		results = make(chan result[T4]) // channel for collecting processed results
	)

	go func() {
		defer close(results)

		// limit concurrency based on configuration
		guard := make(chan struct{}, max(1, opt.MaxParallel))
		defer close(guard)

		var wg sync.WaitGroup

	loop:
		for input := range inputs {
			select {
			case <-ctx.Done(): // stop processing new inputs if context is canceled
				break loop
			case guard <- struct{}{}: // acquire a concurrency slot
				wg.Add(1)
			}

			// job (a set of steps) is executed in a separate goroutine
			go func(input T1) {
				defer func() {
					select {
					case <-ctx.Done(): // stop processing if context is canceled
					case <-guard: // release the concurrency slot
					}
					wg.Done() // mark this job as complete
				}()

				// execute step 1 with retries
				s1, s1err := retry(ctx, input, opt.RetryAttempts, step1)
				if s1err != nil {
					results <- result[T4]{Err: s1err}

					return
				}

				// execute step 2 with retries
				s2, s2err := retry(ctx, fromPtr(s1), opt.RetryAttempts, step2)
				if s2err != nil {
					results <- result[T4]{Err: s2err}

					return
				}

				// execute step 3 with retries
				s3, s3err := retry(ctx, fromPtr(s2), opt.RetryAttempts, step3)
				if s3err != nil {
					results <- result[T4]{Err: s3err}

					return
				}

				results <- result[T4]{Value: fromPtr(s3)}
			}(input)
		}

		wg.Wait() // wait for all jobs to complete
	}()

	var errCh = make(chan error, 1)

	// return a sequence of results and an error channel
	return func(yield func(T4, error) bool) {
		defer cancel()
		defer close(errCh)

		var errCount uint

		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()

				return
			case res, channelOpen := <-results:
				if !channelOpen { // stop if all results are processed
					return
				}

				// track errors and check against maxErrorsToStop threshold
				if opt.MaxErrorsToStop > 0 && res.Err != nil {
					errCount++

					if errCount >= opt.MaxErrorsToStop {
						cancel() // cancel all ongoing jobs
						errCh <- ErrTooManyErrors

						return
					}
				}

				// yield processed results
				if !yield(res.Value, res.Err) {
					return
				}
			}
		}
	}, errCh
}

// newErrChan creates a new error channel with the given error. The channel is closed immediately.
func newErrChan(err error) <-chan error {
	ch := make(chan error, 1)
	ch <- err
	close(ch)

	return ch
}

// fromPtr safely dereferences a pointer, returning the zero value if the pointer is nil.
func fromPtr[T any](v *T) T {
	if v == nil {
		return *new(T)
	}

	return *v
}

// retry attempts to execute a function up to retryAttempts+1 times until it succeeds.
// It returns the result of the successful attempt or the last error encountered.
func retry[TIn, TOut any](
	ctx context.Context,
	in TIn,
	retryAttempts int,
	fn func(context.Context, TIn) (*TOut, error),
) (result *TOut, lastErr error) {
	for i := 0; i <= retryAttempts; i++ {
		// check if context was canceled before attempting execution
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		result, lastErr = fn(ctx, in)
		if lastErr == nil {
			return result, nil // success, no need to retry
		}
	}

	return nil, lastErr // return the last error after all attempts
}

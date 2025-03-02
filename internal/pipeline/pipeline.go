package pipeline

import (
	"context"
	"errors"
	"sync"
)

var ErrTooManyErrors = errors.New("too many errors")

type (
	options struct {
		MaxParallel     int  // maximum number of inputs to process concurrently
		RetryAttempts   int  // number of retry attempts per step on failure
		MaxErrorsToStop uint // error threshold that triggers pipeline cancellation (0 means no limit)
	}

	// Option allows setting options for the pipeline.
	Option func(*options)
)

// Apply the given options to the options struct and return a new options set.
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

// Result represents the outcome of a pipeline step, containing either a value or an error.
type Result[T any] struct {
	Value T
	Err   error
}

// ThreeSteps executes a three-step pipeline for processing a slice of inputs.
//
// The pipeline processes each input through three sequential steps, with steps running in parallel
// across different inputs, up to maxParallel concurrent executions (zero means limit equal count of inputs).
//
// Each step can retry up to retryAttempts times in case of failure. If the total number of errors
// exceeds maxErrorsToStop, the pipeline is canceled and returns ErrTooManyErrors.
//
// The function returns a slice of [Result], but the order of results is not guaranteed. In case of context
// cancellation, the function returns immediately with the current results and [context.Canceled].
//
// Important note: each step should respect the context and return as soon as possible when the context is
// canceled to avoid blocking the pipeline.
func ThreeSteps[T1, T2, T3, T4 any]( //nolint:gocognit,gocyclo,funlen
	pCtx context.Context, // parent context for the pipeline
	step1 func(context.Context, T1) (*T2, error),
	step2 func(context.Context, T2) (*T3, error),
	step3 func(context.Context, T3) (*T4, error),
	inputs []T1, // slice of input values to process
	opts ...Option, // optional pipeline configuration
) ([]Result[T4], error) {
	switch {
	case len(inputs) == 0:
		return nil, nil
	case pCtx == nil:
		return nil, errors.New("ctx must not be nil")
	case step1 == nil || step2 == nil || step3 == nil:
		return nil, errors.New("all steps must not be nil")
	}

	// create a new context for the pipeline that can be canceled
	ctx, cancel := context.WithCancel(pCtx)
	defer cancel()

	var (
		opt       = options{}.Apply(opts)
		jobResult = make(chan Result[T4]) // channel for collecting pipeline results
	)

	go func() {
		defer close(jobResult)

		// limit concurrency to either maxParallel or number of inputs, whichever is smaller
		guard := make(chan struct{}, min(max(1, opt.MaxParallel), len(inputs)))
		defer close(guard)

		var wg sync.WaitGroup

	loop:
		for _, input := range inputs {
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
					jobResult <- Result[T4]{Err: s1err}

					return
				}

				// execute step 2 with retries
				s2, s2err := retry(ctx, fromPtr(s1), opt.RetryAttempts, step2)
				if s2err != nil {
					jobResult <- Result[T4]{Err: s2err}

					return
				}

				// execute step 3 with retries
				s3, s3err := retry(ctx, fromPtr(s2), opt.RetryAttempts, step3)
				if s3err != nil {
					jobResult <- Result[T4]{Err: s3err}

					return
				}

				jobResult <- Result[T4]{Value: fromPtr(s3)}
			}(input)
		}

		wg.Wait() // wait for all jobs to complete
	}()

	// collect results and handle error tracking
	return func() (_ []Result[T4], err error) {
		var (
			errCount uint
			out      = make([]Result[T4], 0, len(inputs))
		)

	loop:
		for { //nolint:gosimple
			select {
			case result, channelOpen := <-jobResult:
				if !channelOpen { // channel closed, all jobs completed
					break loop
				}

				// track errors and check against maxErrorsToStop threshold
				if opt.MaxErrorsToStop > 0 && result.Err != nil {
					errCount++

					if errCount >= opt.MaxErrorsToStop {
						cancel() // cancel all ongoing jobs
						err = ErrTooManyErrors

						break loop
					}
				}

				out = append(out, result)
			}
		}

		// if the error is not set yet (avoid overwriting ErrTooManyErrors)
		if err == nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				err = ctxErr
			}
		}

		return out, err
	}()
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

package pipeline

import (
	"context"
	"errors"
	"sync"
)

var ErrTooManyErrors = errors.New("too many errors")

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
// The function returns a slice of Results with the same order as the inputs.
func ThreeSteps[T1, T2, T3, T4 any]( //nolint:gocognit,gocyclo,funlen
	pCtx context.Context, // parent context for the pipeline
	step1 func(context.Context, T1) (*T2, error),
	step2 func(context.Context, T2) (*T3, error),
	step3 func(context.Context, T3) (*T4, error),
	inputs []T1,          // slice of input values to process
	maxParallel int,      // maximum number of inputs to process concurrently
	retryAttempts int,    // number of retry attempts per step on failure
	maxErrorsToStop uint, // error threshold that triggers pipeline cancellation (0 means no limit)
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

	jobResult := make(chan Result[T4]) // channel for collecting pipeline results

	go func() {
		defer close(jobResult)

		// limit concurrency to either maxParallel or number of inputs, whichever is smaller
		guard := make(chan struct{}, min(max(1, maxParallel), len(inputs)))
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

			go func(input T1) {
				defer func() {
					<-guard   // release the concurrency slot
					wg.Done() // mark this job as complete
				}()

				// execute step 1 with retries
				s1, s1err := retry(ctx, input, retryAttempts, step1)
				if s1err != nil {
					jobResult <- Result[T4]{Err: s1err}

					return
				}

				// execute step 2 with retries
				s2, s2err := retry(ctx, fromPtr(s1), retryAttempts, step2)
				if s2err != nil {
					jobResult <- Result[T4]{Err: s2err}

					return
				}

				// execute step 3 with retries
				s3, s3err := retry(ctx, fromPtr(s2), retryAttempts, step3)
				if s3err != nil {
					jobResult <- Result[T4]{Err: s3err}

					return
				}

				jobResult <- Result[T4]{Value: fromPtr(s3)}
			}(input)
		}

		wg.Wait() // wait for all jobs to complete
	}()

	var pipelineError error

	// collect results and handle error tracking
	results := func() []Result[T4] {
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
				if maxErrorsToStop > 0 && result.Err != nil {
					errCount++

					if errCount >= maxErrorsToStop {
						cancel() // cancel all ongoing jobs
						pipelineError = ErrTooManyErrors

						break loop
					}
				}

				out = append(out, result)
			}
		}

		return out
	}()

	return results, pipelineError
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

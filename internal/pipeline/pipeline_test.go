package pipeline_test

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"gh.tarampamp.am/tinifier/v5/internal/pipeline"
)

func TestThreeSteps(t *testing.T) {
	t.Parallel()

	t.Run("common case", func(t *testing.T) {
		t.Parallel()

		result, err := pipeline.ThreeSteps(
			t.Context(),
			func(_ context.Context, in string) (*string, error) { return toPtr(in + " foo"), nil },
			func(_ context.Context, in string) (*string, error) { return toPtr(in + " bar"), nil },
			func(_ context.Context, in string) (*string, error) { return toPtr(in + " baz"), nil },
			[]string{"uno", "dos", "tres"},
			pipeline.WithMaxParallel(2),
			pipeline.WithRetryAttempts(0),
			pipeline.WithMaxErrorsToStop(1),
		)

		assertNoError(t, err)

		assertEqual(t, 3, len(result))
		assertSliceContains(t, result, pipeline.Result[string]{Value: "uno foo bar baz"})
		assertSliceContains(t, result, pipeline.Result[string]{Value: "dos foo bar baz"})
		assertSliceContains(t, result, pipeline.Result[string]{Value: "tres foo bar baz"})
	})

	t.Run("retry attempts", func(t *testing.T) {
		t.Parallel()

		var step1run, step2run, step3run atomic.Int32

		const failTimes = 50

		result, err := pipeline.ThreeSteps(
			t.Context(),
			func(_ context.Context, in int) (*int, error) {
				if step1run.Add(1) > failTimes {
					return toPtr(in + 1), nil
				}

				return nil, errors.New("step1 error")
			},
			func(_ context.Context, in int) (*int, error) {
				if step2run.Add(1) > failTimes {
					return toPtr(in + 1), nil
				}

				return nil, errors.New("step2 error")
			},
			func(_ context.Context, in int) (*int, error) {
				if step3run.Add(1) > failTimes {
					return toPtr(in + 1), nil
				}

				return nil, errors.New("step3 error")
			},
			[]int{1, 2, 3},
			pipeline.WithRetryAttempts(failTimes),
		)

		assertNoError(t, err)

		assertEqual(t, 3, len(result))
		assertSliceContains(t, result, pipeline.Result[int]{Value: 4})
		assertSliceContains(t, result, pipeline.Result[int]{Value: 5})
		assertSliceContains(t, result, pipeline.Result[int]{Value: 6})

		assertEqual(t, failTimes, step1run.Load()-3)
		assertEqual(t, failTimes, step2run.Load()-3)
		assertEqual(t, failTimes, step3run.Load()-3)
	})

	t.Run("max errors to stop", func(t *testing.T) {
		t.Parallel()

		var (
			step3run atomic.Int32
			step3err = errors.New("step3 error")
		)

		result, err := pipeline.ThreeSteps(
			t.Context(),
			func(_ context.Context, in int) (*int, error) {
				return toPtr(in + 1), nil
			},
			func(_ context.Context, in int) (*int, error) {
				return toPtr(in + 1), nil
			},
			func(_ context.Context, in int) (*int, error) {
				step3run.Add(1)

				return nil, step3err
			},
			[]int{1, 2, 3},
			pipeline.WithMaxErrorsToStop(2),
		)

		assertErrorIs(t, pipeline.ErrTooManyErrors, err)
		assertEqual(t, 1, len(result))
		assertSliceContains(t, result, pipeline.Result[int]{Err: step3err})
		assertBiggerOrEqual(t, 2, int(step3run.Load())) // 2 or 3
	})

	t.Run("context cancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		var ster1err = errors.New("step1 error")

		go func() {
			runtime.Gosched()
			<-time.After(time.Millisecond)
			cancel()
		}()

		result, err := pipeline.ThreeSteps(
			ctx,
			func(ctx context.Context, in int) (*int, error) {
				select {
				case <-ctx.Done():
				case <-time.After(time.Hour):
				}

				return nil, ster1err
			},
			func(ctx context.Context, in int) (*int, error) {
				select {
				case <-ctx.Done():
				case <-time.After(time.Hour):
				}

				return nil, errors.New("step2 error")
			},
			func(ctx context.Context, in int) (*int, error) {
				select {
				case <-ctx.Done():
				case <-time.After(time.Hour):
				}

				return nil, errors.New("step3 error")
			},
			[]int{1, 2, 3},
		)

		assertErrorIs(t, context.Canceled, err)
		assertEqual(t, 1, len(result))
		assertSliceContains(t, result, pipeline.Result[int]{Err: ster1err})
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()

		res, err := pipeline.ThreeSteps(
			t.Context(),
			func(context.Context, int) (_ *int, _ error) { return },
			func(context.Context, int) (_ *int, _ error) { return },
			func(context.Context, int) (_ *int, _ error) { return },
			[]int{}, // <-- important
		)

		assertSlicesEqual(t, nil, res)
		assertEqual(t, nil, err)
	})

	t.Run("args validation", func(t *testing.T) {
		t.Parallel()

		var (
			step1 = func(context.Context, int) (_ *int, _ error) { return }
			step2 = func(context.Context, int) (_ *int, _ error) { return }
			step3 = func(context.Context, int) (_ *int, _ error) { return }
		)

		t.Run("nil context", func(t *testing.T) {
			t.Parallel()

			_, err := pipeline.ThreeSteps(
				nil, //nolint:staticcheck // <-- important
				step1,
				step2,
				step3,
				[]int{1},
			)

			assertEqual(t, "ctx must not be nil", err.Error())
		})

		t.Run("nil step1", func(t *testing.T) {
			t.Parallel()

			_, err := pipeline.ThreeSteps(
				t.Context(),
				nil, // <-- important
				step2,
				step3,
				[]int{1},
			)

			assertEqual(t, "all steps must not be nil", err.Error())
		})

		t.Run("nil step2", func(t *testing.T) {
			t.Parallel()

			_, err := pipeline.ThreeSteps(
				t.Context(),
				step1,
				nil, // <-- important
				step3,
				[]int{1},
			)

			assertEqual(t, "all steps must not be nil", err.Error())
		})

		t.Run("nil step3", func(t *testing.T) {
			t.Parallel()

			_, err := pipeline.ThreeSteps[int, int, int, int](
				t.Context(),
				step1,
				step2,
				nil, // <-- important
				[]int{1},
			)

			assertEqual(t, "all steps must not be nil", err.Error())
		})
	})
}

func toPtr[T any](v T) *T { return &v }

func assertEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()

	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func assertBiggerOrEqual(t *testing.T, expected, actual int) {
	t.Helper()

	if actual < expected {
		t.Fatalf("expected %d to be bigger or equal to %d", actual, expected)
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertErrorIs(t *testing.T, expected, actual error) {
	t.Helper()

	if !errors.Is(actual, expected) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func assertSliceContains[T comparable](t *testing.T, slice []T, elem T) {
	t.Helper()

	for _, e := range slice {
		if e == elem {
			return
		}
	}

	t.Fatalf("slice %v does not contain %v", slice, elem)
}

func assertSlicesEqual[T comparable](t *testing.T, expected, actual []T) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}

	for i := range expected {
		if expected[i] != actual[i] {
			t.Fatalf("expected %v, got %v", expected, actual)
		}
	}
}

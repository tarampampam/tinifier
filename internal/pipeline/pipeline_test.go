package pipeline_test

import (
	"context"
	"errors"
	"iter"
	"reflect"
	"runtime"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"gh.tarampamp.am/tinifier/v5/internal/pipeline"
)

func TestThreeSteps(t *testing.T) {
	t.Parallel()

	t.Run("common case", func(t *testing.T) {
		t.Parallel()

		seq, errCh := pipeline.ThreeSteps(
			t.Context(),
			func(_ context.Context, in string) (*string, error) { return toPtr(in + " foo"), nil },
			func(_ context.Context, in string) (*string, error) { return toPtr(in + " bar"), nil },
			func(_ context.Context, in string) (*string, error) { return toPtr(in + " baz"), nil },
			slices.Values([]string{"uno", "dos", "tres"}),
			pipeline.WithMaxParallel(2),
			pipeline.WithRetryAttempts(0),
			pipeline.WithMaxErrorsToStop(1),
		)

		resultMap := seq2ToMap(seq)

		assertNoError(t, <-errCh)
		assertEqual(t, 3, len(resultMap))

		const uno, dos, tres = "uno foo bar baz", "dos foo bar baz", "tres foo bar baz"

		assertMapHasKey(t, resultMap, uno)
		assertNil(t, resultMap[uno])
		assertMapHasKey(t, resultMap, dos)
		assertNil(t, resultMap[dos])
		assertMapHasKey(t, resultMap, tres)
		assertNil(t, resultMap[tres])
	})

	t.Run("retry attempts", func(t *testing.T) {
		t.Parallel()

		var step1run, step2run, step3run atomic.Int32

		const failTimes = 50

		seq, errCh := pipeline.ThreeSteps(
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
			slices.Values([]int{1, 2, 3}),
			pipeline.WithRetryAttempts(failTimes),
		)

		resultMap := seq2ToMap(seq)

		assertNoError(t, <-errCh)
		assertEqual(t, 3, len(resultMap))

		assertMapHasKey(t, resultMap, 4)
		assertNil(t, resultMap[4])
		assertMapHasKey(t, resultMap, 5)
		assertNil(t, resultMap[5])
		assertMapHasKey(t, resultMap, 6)
		assertNil(t, resultMap[6])

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

		seq, errCh := pipeline.ThreeSteps(
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
			slices.Values([]int{1, 2, 3}),
			pipeline.WithMaxErrorsToStop(2),
		)

		resultMap := seq2ToMap(seq)

		assertEqual(t, 1, len(resultMap))
		assertErrorIs(t, pipeline.ErrTooManyErrors, <-errCh)

		assertEqual(t, 1, len(resultMap))
		assertMapHasKey(t, resultMap, 0)
		assertEqual(t, step3err, resultMap[0])
		assertBiggerOrEqual(t, 2, int(step3run.Load())) // 2 or 3
	})

	t.Run("context cancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go func() {
			runtime.Gosched()
			<-time.After(time.Millisecond)
			cancel()
		}()

		seq, errCh := pipeline.ThreeSteps(
			ctx,
			func(ctx context.Context, in int) (*int, error) {
				select {
				case <-ctx.Done():
				case <-time.After(time.Hour):
				}

				return nil, errors.New("step1 error")
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
			slices.Values([]int{1, 2, 3}),
		)

		resultMap := seq2ToMap(seq)

		assertErrorIs(t, context.Canceled, <-errCh)
		assertEqual(t, 0, len(resultMap))
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()

		seq, errCh := pipeline.ThreeSteps(
			t.Context(),
			func(context.Context, int) (_ *int, _ error) { return },
			func(context.Context, int) (_ *int, _ error) { return },
			func(context.Context, int) (_ *int, _ error) { return },
			slices.Values([]int{}), // <-- important
		)

		resultMap := seq2ToMap(seq)

		assertEqual(t, 0, len(resultMap))
		assertNil(t, <-errCh)
	})

	t.Run("args validation", func(t *testing.T) {
		t.Parallel()

		var (
			step1 = func(context.Context, int) (_ *int, _ error) { return }
			step2 = func(context.Context, int) (_ *int, _ error) { return }
			step3 = func(context.Context, int) (_ *int, _ error) { return }
		)

		t.Run("nil input", func(t *testing.T) {
			t.Parallel()

			seq, errCh := pipeline.ThreeSteps(
				t.Context(),
				step1,
				step2,
				step3,
				nil, // <-- important
			)

			assertNil(t, seq)
			assertNil(t, <-errCh)
		})

		t.Run("nil context", func(t *testing.T) {
			t.Parallel()

			_, errCh := pipeline.ThreeSteps(
				nil, //nolint:staticcheck // <-- important
				step1,
				step2,
				step3,
				slices.Values([]int{1}),
			)

			assertEqual(t, "ctx must not be nil", (<-errCh).Error())
		})

		t.Run("nil step1", func(t *testing.T) {
			t.Parallel()

			_, errCh := pipeline.ThreeSteps(
				t.Context(),
				nil, // <-- important
				step2,
				step3,
				slices.Values([]int{1}),
			)

			assertEqual(t, "all steps must not be nil", (<-errCh).Error())
		})

		t.Run("nil step2", func(t *testing.T) {
			t.Parallel()

			_, errCh := pipeline.ThreeSteps(
				t.Context(),
				step1,
				nil, // <-- important
				step3,
				slices.Values([]int{1}),
			)

			assertEqual(t, "all steps must not be nil", (<-errCh).Error())
		})

		t.Run("nil step3", func(t *testing.T) {
			t.Parallel()

			_, errCh := pipeline.ThreeSteps[int, int, int, int](
				t.Context(),
				step1,
				step2,
				nil, // <-- important
				slices.Values([]int{1}),
			)

			assertEqual(t, "all steps must not be nil", (<-errCh).Error())
		})
	})
}

func toPtr[T any](v T) *T { return &v }

func seq2ToMap[T comparable, U any](seq iter.Seq2[T, U]) map[T]U {
	m := make(map[T]U)

	for k, v := range seq {
		m[k] = v
	}

	return m
}

func assertMapHasKey[T comparable, U any](t *testing.T, m map[T]U, key T) {
	t.Helper()

	if _, ok := m[key]; !ok {
		t.Fatalf("map %v does not have key %v", m, key)
	}
}

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

func assertNil(t *testing.T, v any) {
	t.Helper()

	if ref := reflect.ValueOf(v); ref.Kind() == reflect.Ptr && !ref.IsNil() {
		t.Fatalf("expected nil, got %v", v)
	}
}

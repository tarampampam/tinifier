package pipeline_test

import (
	"context"
	"testing"

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
			2,
			0,
			1,
		)

		assertNoError(t, err)

		assertEqual(t, 3, len(result))
		assertSliceContains(t, result, pipeline.Result[string]{Value: "uno foo bar baz"})
		assertSliceContains(t, result, pipeline.Result[string]{Value: "dos foo bar baz"})
		assertSliceContains(t, result, pipeline.Result[string]{Value: "tres foo bar baz"})
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()

		res, err := pipeline.ThreeSteps(
			t.Context(),
			func(context.Context, int) (_ *int, _ error) { return },
			func(context.Context, int) (_ *int, _ error) { return },
			func(context.Context, int) (_ *int, _ error) { return },
			[]int{}, // <-- important
			1,
			1,
			1,
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
				nil, // <-- important
				step1,
				step2,
				step3,
				[]int{1},
				1,
				1,
				1,
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
				1,
				1,
				1,
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
				1,
				1,
				1,
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
				1,
				1,
				1,
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

func assertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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

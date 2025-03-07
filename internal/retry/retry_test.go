package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gh.tarampamp.am/tinifier/v5/internal/retry"
)

func TestTry(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var err = retry.Try(t.Context(), 3, func(_ context.Context, attempt uint) error {
			assertEqual(t, uint(1), attempt)

			return nil
		})

		assertNoError(t, err)
	})

	t.Run("retry attempts exceeded", func(t *testing.T) {
		t.Parallel()

		var (
			counter    uint
			attemptErr = errors.New("attempt error")
		)

		var err = retry.Try(t.Context(), 3, func(_ context.Context, attempt uint) error {
			counter++
			assertEqual(t, counter, attempt)

			return attemptErr
		})

		assertEqual(t, 3, counter)
		assertErrorIs(t, err, attemptErr)
		assertErrorIs(t, err, retry.ErrRetryAttemptsExceeded)
	})

	t.Run("delay between attempts", func(t *testing.T) {
		t.Parallel()

		var now = time.Now()

		const (
			delay     = 10 * time.Millisecond
			attempts  = 3
			finalTime = delay * time.Duration(attempts-1)
		)

		var err = retry.Try(
			t.Context(),
			attempts,
			func(_ context.Context, _ uint) error { return errors.New("error") },
			retry.WithDelayBetweenAttempts(delay),
		)

		assertErrorIs(t, err, retry.ErrRetryAttemptsExceeded)
		assertEqual(t, true, time.Since(now).Round(time.Millisecond) >= finalTime)
	})

	t.Run("err to stop", func(t *testing.T) {
		t.Parallel()

		var (
			counter   uint
			errToStop = errors.New("err to stop")
		)

		var err = retry.Try(
			t.Context(),
			3,
			func(_ context.Context, _ uint) error { counter++; return errToStop },
			retry.WithStopOnError(errToStop),
		)

		assertEqual(t, 1, counter)
		assertErrorIs(t, err, errToStop)
	})

	t.Run("on canceled context", func(t *testing.T) {
		t.Parallel()

		var (
			ctx, cancel = context.WithCancel(t.Context())
			counter     uint
		)

		cancel()

		var err = retry.Try(
			ctx,
			3,
			func(_ context.Context, _ uint) error { counter++; return errors.New("error") },
		)

		assertEqual(t, 0, counter)
		assertErrorIs(t, err, context.Canceled)
	})

	t.Run("on context cancellation during delay", func(t *testing.T) {
		t.Parallel()

		var (
			ctx, cancel = context.WithCancel(t.Context())
			counter     uint
		)

		go func() {
			time.Sleep(time.Millisecond)
			cancel()
		}()

		var err = retry.Try(
			ctx,
			3,
			func(_ context.Context, _ uint) error { counter++; return errors.New("error") },
			retry.WithDelayBetweenAttempts(time.Hour),
		)

		assertEqual(t, 1, counter)
		assertErrorIs(t, err, context.Canceled)
	})
}

func assertEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()

	if expected != actual {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertErrorIs(t *testing.T, err error, expected error) {
	t.Helper()

	if !errors.Is(err, expected) {
		t.Fatalf("expected error: %v, got: %v", expected, err)
	}
}

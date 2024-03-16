package retry_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"gh.tarampamp.am/tinifier/v4/internal/retry"
)

func ExampleDo() {
	var ctx = context.Background()

	limitExceeded, err := retry.Do(func(attemptNum uint) error {
		fmt.Println(attemptNum) // the function accepts attempt number as an argument

		// some unstable function call should be here. for example - calling the remote API, or something else
		if attemptNum < 3 {
			return errors.New("foo error")
		}

		fmt.Println("success")

		return nil // you should return nil if retrying is not needed
	}, retry.WithContext(ctx), retry.Attempts(10), retry.Delay(time.Millisecond*10))

	if err != nil {
		panic(err)
	}

	fmt.Println("Limit exceeded:", limitExceeded)

	// Output:
	// 1
	// 2
	// 3
	// success
	// Limit exceeded: false
}

func TestDo_DefaultsAlwaysError(t *testing.T) {
	var (
		execCounter uint
		fnErr       = errors.New("foo error")
	)

	limitExceeded, err := retry.Do(func(attemptNum uint) error {
		execCounter++
		assert.Equal(t, execCounter, attemptNum)

		return fnErr
	})

	assert.Equal(t, uint(3), execCounter)
	assert.True(t, limitExceeded)
	assert.ErrorIs(t, err, fnErr)
}

func TestDo_DefaultsOnce(t *testing.T) {
	var execCounter uint

	limitExceeded, err := retry.Do(func(attemptNum uint) error {
		execCounter++
		assert.Equal(t, execCounter, attemptNum)

		return nil
	})

	assert.NoError(t, err)
	assert.False(t, limitExceeded)
	assert.Equal(t, uint(1), execCounter)
}

func TestWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())

	cancel() // <-- important

	var execCounter uint

	limitExceeded, err := retry.Do(func(attemptNum uint) error {
		execCounter++
		assert.Equal(t, execCounter, attemptNum)

		return nil
	}, retry.WithContext(ctx))

	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, limitExceeded)
	assert.Equal(t, uint(0), execCounter)
}

func TestWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())

	var execCounter uint

	limitExceeded, err := retry.Do(func(attemptNum uint) error {
		assert.Equal(t, uint(1), attemptNum)

		execCounter++

		cancel() // <-- important

		return errors.New("foo error")
	}, retry.WithContext(ctx))

	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, limitExceeded)
	assert.Equal(t, uint(1), execCounter)
}

func TestDo_WithAttempts(t *testing.T) {
	var (
		execCounter uint
		fnErr       = errors.New("foo error")
	)

	limitExceeded, err := retry.Do(func(attemptNum uint) error {
		execCounter++
		assert.Equal(t, execCounter, attemptNum)

		return fnErr
	}, retry.Attempts(33), retry.Delay(time.Nanosecond))

	assert.Equal(t, uint(33), execCounter)
	assert.True(t, limitExceeded)
	assert.ErrorIs(t, err, fnErr)
}

func TestDo_WithDelay(t *testing.T) {
	var execCounter uint

	startedAt := time.Now()

	limitExceeded, err := retry.Do(func(attemptNum uint) error {
		if execCounter > 0 {
			return nil
		}

		execCounter++
		assert.Equal(t, execCounter, attemptNum)

		return errors.New("foo error")
	}, retry.Delay(time.Millisecond*123))

	assert.WithinDuration(t, time.Now(), startedAt, time.Millisecond*123+time.Millisecond*25)
	assert.Equal(t, uint(1), execCounter)
	assert.False(t, limitExceeded)
	assert.NoError(t, err)
}

func TestDo_WithRetryStoppingErrors(t *testing.T) {
	var (
		execCounter uint
		errToStop   = errors.New("hell yeah")
	)

	limitExceeded, err := retry.Do(func(uint) error {
		execCounter++

		if execCounter >= 3 {
			return errToStop
		}

		return errors.New("don't stop the planet")
	}, retry.Attempts(20), retry.StopOnError(errToStop))

	assert.ErrorIs(t, err, errToStop)
	assert.False(t, limitExceeded)
	assert.Equal(t, uint(3), execCounter)
}

func TestDo_WithRetryStoppingErrorsWithWrappedError(t *testing.T) {
	var (
		execCounter uint
		errToStop   = errors.New("hell yeah")
	)

	limitExceeded, err := retry.Do(func(uint) error {
		execCounter++

		return errors.Wrap(errToStop, "don't stop the planet")
	}, retry.Attempts(20), retry.StopOnError(errToStop))

	assert.EqualError(t, err, "don't stop the planet: hell yeah")
	assert.ErrorIs(t, err, errToStop)
	assert.False(t, limitExceeded)
	assert.Equal(t, uint(1), execCounter)
}

func TestDo_WithRetryStoppingErrorsWithLastError(t *testing.T) {
	var (
		execCounter uint
		errToStop   = errors.New("hell yeah")
	)

	limitExceeded, err := retry.Do(func(uint) error {
		execCounter++

		return errToStop
	}, retry.Attempts(20), retry.StopOnError(errToStop))

	assert.ErrorIs(t, err, errToStop)
	assert.False(t, limitExceeded)
	assert.Equal(t, uint(1), execCounter)
}

func TestDo_InfiniteAttemptsLoop(t *testing.T) {
	var (
		execCounter uint
		errContinue = errors.New("hell yeah")
	)

	limitExceeded, err := retry.Do(func(uint) error {
		execCounter++

		if execCounter < 9_999_999 { // some very large value
			return errContinue
		}

		return nil
	}, retry.Attempts(0))

	assert.NoError(t, err)
	assert.False(t, limitExceeded)
	assert.EqualValues(t, 9_999_999, execCounter)
}

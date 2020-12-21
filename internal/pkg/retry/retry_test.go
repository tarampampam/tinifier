package retry

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ExampleDo() {
	var (
		content []byte
		ctx     = context.Background()
	)

	err := Do(func(uint) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://httpbin.org/anything", nil)
		if err != nil {
			return err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()

		content, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return nil
	}, WithContext(ctx), WithAttempts(10), WithDelay(time.Second), WithLastErrorReturning())

	if err != nil {
		panic(err)
	}

	fmt.Printf("Response: %s", string(content))
}

func TestDoDefaultsAlwaysError(t *testing.T) {
	var execCounter uint

	assert.EqualError(t, Do(func(attemptNum uint) error {
		execCounter++
		assert.Equal(t, execCounter, attemptNum)

		return errors.New("foo error")
	}), ErrToManyAttempts.Error())

	assert.Equal(t, uint(3), execCounter)
}

func TestDoDefaultsOnce(t *testing.T) {
	var execCounter uint

	assert.NoError(t, Do(func(attemptNum uint) error {
		execCounter++
		assert.Equal(t, execCounter, attemptNum)

		return nil
	}))

	assert.Equal(t, uint(1), execCounter)
}

func TestWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())

	cancel() // <-- important

	var execCounter uint

	assert.EqualError(t, Do(func(attemptNum uint) error {
		execCounter++
		assert.Equal(t, execCounter, attemptNum)

		return nil
	}, WithContext(ctx)), context.Canceled.Error())

	assert.Equal(t, uint(0), execCounter)
}

func TestWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())

	var execCounter uint

	assert.EqualError(t, Do(func(attemptNum uint) error {
		assert.Equal(t, uint(1), attemptNum)

		execCounter++
		cancel() // <-- important

		return errors.New("foo error")
	}, WithContext(ctx)), context.Canceled.Error())

	assert.Equal(t, uint(1), execCounter)
}

func TestDoWithAttempts(t *testing.T) {
	var execCounter uint

	assert.EqualError(t, Do(func(attemptNum uint) error {
		execCounter++
		assert.Equal(t, execCounter, attemptNum)

		return errors.New("foo error")
	}, WithAttempts(33), WithDelay(time.Nanosecond)), ErrToManyAttempts.Error())

	assert.Equal(t, uint(33), execCounter)
}

func TestDoWithZeroAttempts(t *testing.T) {
	var execCounter uint

	assert.EqualError(t, Do(func(attemptNum uint) error {
		execCounter++
		assert.Equal(t, execCounter, attemptNum)

		return errors.New("foo error")
	}, WithAttempts(0)), ErrNoAttempts.Error())

	assert.Equal(t, uint(0), execCounter)
}

func TestDoWithLastErrorReturning(t *testing.T) {
	var execCounter uint

	assert.EqualError(t, Do(func(attemptNum uint) error {
		execCounter++
		assert.Equal(t, execCounter, attemptNum)

		return errors.New("foo error")
	}, WithAttempts(2), WithLastErrorReturning()), "foo error")

	assert.Equal(t, uint(2), execCounter)
}

func TestDoWithDelay(t *testing.T) {
	var execCounter uint

	startedAt := time.Now().UnixNano()

	assert.NoError(t, Do(func(attemptNum uint) error {
		if execCounter > 0 {
			return nil
		}

		execCounter++
		assert.Equal(t, execCounter, attemptNum)

		return errors.New("foo error")
	}, WithDelay(time.Millisecond*123)))

	endedAt := time.Now().UnixNano()

	assert.InDelta(t, 123, (endedAt-startedAt)/1000000, 5)
	assert.Equal(t, uint(1), execCounter)
}

package uikit_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tarampampam/tinifier/v4/internal/uikit"
)

func TestNoOut(t *testing.T) {
	var output = uikit.NoOut()

	assert.NotNil(t, output)
	assert.Same(t, output, uikit.NoOut())
	assert.EqualValues(t, "noop", output.String())

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			l, err := output.Write([]byte{1, 2, 3})

			assert.NoError(t, err)
			assert.Equal(t, 3, l)
		}()
	}

	wg.Wait()
}

func TestStdOut(t *testing.T) {
	var output = uikit.StdOut()

	assert.NotNil(t, output)
	assert.Same(t, output, uikit.StdOut())
	assert.EqualValues(t, "stdout", output.String())

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			l, err := output.Write([]byte{})

			assert.NoError(t, err)
			assert.Equal(t, 0, l)
		}()
	}

	wg.Wait()
}

func TestStdErr(t *testing.T) {
	var output = uikit.StdErr()

	assert.NotNil(t, output)
	assert.Same(t, output, uikit.StdErr())
	assert.EqualValues(t, "stderr", output.String())

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			l, err := output.Write([]byte{})

			assert.NoError(t, err)
			assert.Equal(t, 0, l)
		}()
	}

	wg.Wait()
}

func TestBufOut(t *testing.T) {
	var output = uikit.BufOut()

	assert.NotNil(t, output)
	assert.NotSame(t, output, uikit.BufOut()) // always new instance
	assert.EqualValues(t, "buffer", output.String())

	output.Grow(1_000)

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			l, err := output.Write([]byte{1, 2, 3})

			assert.NoError(t, err)
			assert.Equal(t, 3, l)
		}()
	}

	wg.Wait()

	assert.EqualValues(t, 100*3, output.Len())
	assert.Len(t, output.AsBytes(), 100*3)
	assert.EqualValues(t, []byte{1, 2, 3, 1, 2, 3, 1}, output.AsBytes()[:7])

	output.Reset()

	assert.EqualValues(t, 0, output.Len())
	assert.Len(t, output.AsBytes(), 0)
	assert.EqualValues(t, []byte{}, output.AsBytes())

	_, _ = output.Write([]byte("hello"))

	assert.EqualValues(t, "hello", output.AsString())
}

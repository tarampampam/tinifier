package ui_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tarampampam/tinifier/v4/internal/ui"
)

func TestOutputMutate(t *testing.T) {
	for name, tt := range map[string]struct {
		giveOut ui.Output
	}{
		"stdout": {ui.StdOut()},
		"stderr": {ui.StdErr()},
		"noop":   {ui.NoOut()},
		"buffer": {ui.BufOut()},
	} {
		t.Run(name, func(t *testing.T) {
			var (
				origData   = []byte{1, 2, 3}
				mutatorRan bool
			)

			cancelMutation := tt.giveOut.Mutate(func(data *[]byte) {
				defer func() { mutatorRan = true }()

				assert.EqualValues(t, origData, *data)

				*data = []byte{} // reset data
			})

			defer cancelMutation()

			assert.False(t, mutatorRan)

			wrote, err := tt.giveOut.Write(origData)

			assert.True(t, mutatorRan)
			assert.Equal(t, 0, wrote)
			assert.NoError(t, err)
		})
	}
}

func TestOutputMutateWithBuffer(t *testing.T) {
	var out = ui.BufOut()

	var cancelM1 = out.Mutate(func(data *[]byte) {
		*data = append(*data, []byte{1, 2, 3}...)
	})

	l, err := out.Write([]byte{7})

	assert.EqualValues(t, 4, l)
	assert.NoError(t, err)
	assert.EqualValues(t, []byte{7, 1, 2, 3}, out.AsBytes())

	out.Reset()

	l, err = out.Write([]byte{7})

	assert.EqualValues(t, 4, l) // not changed
	assert.NoError(t, err)
	assert.EqualValues(t, []byte{7, 1, 2, 3}, out.AsBytes())

	out.Reset()

	cancelM1() // call the cancellation func now
	cancelM1()
	cancelM1()

	l, err = out.Write([]byte{7})

	assert.EqualValues(t, 1, l) // without mutation
	assert.NoError(t, err)
	assert.EqualValues(t, []byte{7}, out.AsBytes())
}

func TestNoOut(t *testing.T) {
	var output = ui.NoOut()

	assert.NotNil(t, output)
	assert.Same(t, output, ui.NoOut())
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
	var output = ui.StdOut()

	assert.NotNil(t, output)
	assert.Same(t, output, ui.StdOut())
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
	var output = ui.StdErr()

	assert.NotNil(t, output)
	assert.Same(t, output, ui.StdErr())
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
	var output = ui.BufOut()

	assert.NotNil(t, output)
	assert.NotSame(t, output, ui.BufOut()) // always new instance
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

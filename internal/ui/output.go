package ui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/mattn/go-colorable"
)

type (
	WritingMutator interface {
		// Mutate adds the given mutator function into pre-write hooks. The cancellation function calling removes it.
		// Mutators' execution order immutability is NOT guaranteed.
		Mutate(mutateDataFn) (cancel func())
	}

	// Output is a writer that can be used to write to stdout or stderr.
	Output interface {
		io.Writer
		fmt.Stringer
		WritingMutator
	}

	// BufferedOutput is a buffered output.
	BufferedOutput interface {
		Output

		Grow(n int)       // Grow grows the buffer to the given size.
		Reset()           // Reset resets the output buffer to be empty.
		AsBytes() []byte  // AsBytes returns a buffer as a slice of bytes.
		AsString() string // AsString returns the contents of the buffer as a string.
		Len() int         // Len returns the number of bytes in the output buffer.
	}
)

// the following variables are used as a singletons for stdout/stderr/noop (like a context.Background).
var (
	stdOut Output = &outWithLocker{dest: colorable.NewColorable(os.Stdout)} //nolint:gochecknoglobals
	stdErr Output = &outWithLocker{dest: colorable.NewColorable(os.Stderr)} //nolint:gochecknoglobals
	noOut  Output = &outWithLocker{dest: io.Discard}                        //nolint:gochecknoglobals
	_      Output = new(bufOutWithLocker)
)

// StdOut returns a stout pipe writer.
func StdOut() Output { return stdOut }

// StdErr returns a stderr pipe writer.
func StdErr() Output { return stdErr }

// NoOut returns a no-op writer.
func NoOut() Output { return noOut }

// BufOut returns the new buffered output (common use case - unit-tests).
func BufOut() BufferedOutput { return new(bufOutWithLocker) }

// outWithLocker is a wrapper for io.Writer that locks the mutex.
type outWithLocker struct {
	m    sync.Mutex
	dest io.Writer
	mut  outputMutator
}

// Mutate adds the given mutator function into pre-write hooks. The cancellation function calling removes it.
func (o *outWithLocker) Mutate(fn mutateDataFn) (cancel func()) { return o.mut.Add(fn) }

// Write writes the given bytes to the output.
func (o *outWithLocker) Write(p []byte) (n int, err error) {
	o.mut.RunAll(&p)

	if o.dest != io.Discard {
		o.m.Lock()
		n, err = o.dest.Write(p)
		o.m.Unlock()
	} else {
		n = len(p)
	}

	return
}

// String returns a string representation of the output (name only).
func (o *outWithLocker) String() string {
	switch {
	case o.dest == os.Stdout:
		return "stdout"
	case o.dest == os.Stderr:
		return "stderr"
	case o.dest == io.Discard:
		return "noop"
	}

	return "unknown"
}

// bufOutWithLocker is a buffered output.
type bufOutWithLocker struct {
	m   sync.Mutex
	buf bytes.Buffer
	mut outputMutator
}

// Grow grows the output buffer to the given size.
func (o *bufOutWithLocker) Grow(n int) { o.m.Lock(); o.buf.Grow(n); o.m.Unlock() }

// Reset resets the output buffer to be empty.
func (o *bufOutWithLocker) Reset() { o.m.Lock(); o.buf.Reset(); o.m.Unlock() }

// AsBytes returns a buffer as a slice of bytes.
func (o *bufOutWithLocker) AsBytes() []byte { return o.buf.Bytes() }

// AsString returns the contents of the buffer as a string.
func (o *bufOutWithLocker) AsString() string { return o.buf.String() }

// Len returns the number of bytes in the output buffer.
func (o *bufOutWithLocker) Len() int { return o.buf.Len() }

func (o *bufOutWithLocker) Mutate(fn mutateDataFn) (cancel func()) { return o.mut.Add(fn) }

// Write writes the given bytes to the output.
func (o *bufOutWithLocker) Write(p []byte) (n int, err error) {
	o.mut.RunAll(&p)

	o.m.Lock()
	n, err = o.buf.Write(p)
	o.m.Unlock()

	return
}

// String returns a string representation of the output (name only).
func (o *bufOutWithLocker) String() string { return "buffer" }

type (
	// outputMutator allows to mutate the output data before writing it.
	outputMutator struct {
		counter uint32 // atomic usage only
		m       sync.Mutex
		mut     map[uint32]mutateDataFn
	}

	// mutateDataFn is a function that mutates the data.
	mutateDataFn func(data *[]byte)
)

// Add adds the given mutator func into the collection. The cancellation function calling removes it. Cancellation
// function can be called multiple times.
func (om *outputMutator) Add(fn mutateDataFn) (cancel func()) {
	om.m.Lock()
	defer om.m.Unlock()

	if om.mut == nil { // lazy init
		om.mut = make(map[uint32]mutateDataFn)
	}

	var (
		id   = atomic.AddUint32(&om.counter, 1)
		once sync.Once
	)

	om.mut[id] = fn

	return func() {
		once.Do(func() {
			om.m.Lock()
			delete(om.mut, id)
			om.m.Unlock()
		})
	}
}

// RunAll runs all the mutators on the given data.
func (om *outputMutator) RunAll(data *[]byte) {
	om.m.Lock()
	defer om.m.Unlock()

	for _, fn := range om.mut {
		om.m.Unlock()
		fn(data)
		om.m.Lock()
	}
}

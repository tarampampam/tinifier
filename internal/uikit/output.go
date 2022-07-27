package uikit

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
)

type (
	// Output is a writer that can be used to write to stdout or stderr.
	Output interface {
		io.Writer
		fmt.Stringer
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
	stdOut Output = &outWithLocker{dest: os.Stdout}
	stdErr Output = &outWithLocker{dest: os.Stderr}
	noOut  Output = &outWithLocker{dest: io.Discard}
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
}

// Write writes the given bytes to the output.
func (o *outWithLocker) Write(p []byte) (n int, err error) {
	if o.dest != io.Discard {
		o.m.Lock()
		defer o.m.Unlock()
	}

	n, err = o.dest.Write(p)

	return
}

// String returns a string representation of the output.
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
}

// Grow grows the output buffer to the given size.
func (o *bufOutWithLocker) Grow(n int) { o.buf.Grow(n) }

// Reset resets the output buffer to be empty.
func (o *bufOutWithLocker) Reset() { o.buf.Reset() }

// AsBytes returns a buffer as a slice of bytes.
func (o *bufOutWithLocker) AsBytes() []byte { return o.buf.Bytes() }

// AsString returns the contents of the buffer as a string.
func (o *bufOutWithLocker) AsString() string { return o.buf.String() }

// Len returns the number of bytes in the output buffer.
func (o *bufOutWithLocker) Len() int { return o.buf.Len() }

// Write writes the given bytes to the output.
func (o *bufOutWithLocker) Write(p []byte) (n int, err error) {
	o.m.Lock()
	n, err = o.buf.Write(p)
	o.m.Unlock()

	return
}

// String returns a string representation of the output.
func (o *bufOutWithLocker) String() string { return "buffer" }

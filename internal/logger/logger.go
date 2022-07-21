// Package logger contains functions for a working with application logging.
package logger

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/fatih/color"
)

type Logger interface {
	// Debug logs a message at DebugLevel.
	Debug(msg string, v ...any)

	// Info logs a message at InfoLevel.
	Info(msg string, v ...any)

	// Warn logs a message at WarnLevel.
	Warn(msg string, v ...any)

	// Error logs a message at ErrorLevel.
	Error(msg string, v ...any)
}

type output struct {
	mu sync.Mutex
	to io.Writer
}

// LogOption is a function that can be used to modify a Log.
type LogOption func(*Log)

// WithStdOut sets the writer for standard output.
func WithStdOut(w io.Writer) LogOption { return func(l *Log) { l.stdOut.to = w } }

// WithStdErr sets the writer for standard error.
func WithStdErr(w io.Writer) LogOption { return func(l *Log) { l.errOut.to = w } }

// Log is a logger that logs messages at specified level.
type Log struct {
	stdOut, errOut output
	lvl            Level
}

// New creates a new Logger with specified level.
func New(lvl Level, opts ...LogOption) *Log {
	var log = &Log{
		stdOut: output{to: os.Stdout},
		errOut: output{to: os.Stderr},
		lvl:    lvl,
	}

	for _, opt := range opts {
		opt(log)
	}

	return log
}

// NewNop creates a no-op Logger.
func NewNop() *Log {
	return &Log{
		stdOut: output{to: io.Discard},
		errOut: output{to: io.Discard},
		lvl:    noLevel,
	}
}

const (
	debugPrefix = " debug "
	infoPrefix  = "  info "
	warnPrefix  = "  warn "
	errorPrefix = " error "

	prefixGrowSize = 7 /* prefix */ + 8*4 /* colors */ + 12 /* timestamp */
)

var (
	debugColor       = color.New(color.FgMagenta)              //nolint:gochecknoglobals
	infoColor        = color.New(color.FgBlue)                 //nolint:gochecknoglobals
	warnColor        = color.New(color.FgHiYellow, color.Bold) //nolint:gochecknoglobals
	errorColor       = color.New(color.FgHiRed, color.Bold)    //nolint:gochecknoglobals
	underlineColor   = color.New(color.Underline)              //nolint:gochecknoglobals
	runtimeInfoColor = color.New(color.FgWhite)                //nolint:gochecknoglobals

	debugMarker = color.New(color.BgMagenta, color.FgHiMagenta) //nolint:gochecknoglobals
	infoMarker  = color.New(color.BgBlue, color.FgHiBlue)       //nolint:gochecknoglobals
	warnMarker  = color.New(color.BgHiYellow, color.FgBlack)    //nolint:gochecknoglobals
	errorMarker = color.New(color.BgHiRed, color.FgHiWhite)     //nolint:gochecknoglobals
)

func (*Log) write(out *output, prefix []byte, msg string, extra ...any) {
	var buf, extraBuf bytes.Buffer

	if len(extra) > 0 {
		extraBuf.Grow(len(extra) * 32)
		extraBuf.WriteRune('(')

		for i, v := range extra {
			extraBuf.WriteString(fmt.Sprint(v))

			if i < len(extra)-1 {
				extraBuf.WriteRune(' ')
			}
		}

		extraBuf.WriteRune(')')
	}

	buf.Grow(len(prefix) + len(msg) + extraBuf.Len() + 12)

	if len(prefix) > 0 {
		buf.Write(prefix)
		buf.WriteRune(' ')
	}

	buf.WriteString(msg)

	if extraBuf.Len() > 0 {
		buf.WriteRune(' ')
		_, _ = runtimeInfoColor.Fprint(&buf, extraBuf.String())
	}

	buf.WriteRune('\n')

	out.mu.Lock()
	_, _ = buf.WriteTo(out.to)
	out.mu.Unlock()
}

func (*Log) getTimestamp() string {
	const timeFormat = "15:04:05.000"

	return time.Now().Format(timeFormat)
}

// Debug logs a message at DebugLevel.
func (l *Log) Debug(msg string, v ...any) {
	if DebugLevel >= l.lvl {
		var prefix bytes.Buffer

		prefix.Grow(prefixGrowSize)
		_, _ = debugMarker.Fprint(&prefix, debugPrefix)
		prefix.WriteRune(' ')
		_, _ = debugColor.Fprint(&prefix, l.getTimestamp())

		if _, file, line, ok := runtime.Caller(1); ok {
			prefix.WriteRune(' ')
			_, _ = underlineColor.Fprintf(&prefix, "%s:%d", filepath.Base(file), line)
		}

		l.write(&l.stdOut, prefix.Bytes(), msg, v...)
	}
}

// Info logs a message at InfoLevel.
func (l *Log) Info(msg string, v ...any) {
	if InfoLevel >= l.lvl {
		var prefix bytes.Buffer

		prefix.Grow(prefixGrowSize)
		_, _ = infoMarker.Fprint(&prefix, infoPrefix)
		prefix.WriteRune(' ')
		_, _ = infoColor.Fprint(&prefix, l.getTimestamp())

		l.write(&l.stdOut, prefix.Bytes(), msg, v...)
	}
}

// Warn logs a message at WarnLevel.
func (l *Log) Warn(msg string, v ...any) {
	if WarnLevel >= l.lvl {
		var prefix bytes.Buffer

		prefix.Grow(prefixGrowSize)
		_, _ = warnMarker.Fprint(&prefix, warnPrefix)
		prefix.WriteRune(' ')
		_, _ = warnColor.Fprint(&prefix, l.getTimestamp())

		l.write(&l.stdOut, prefix.Bytes(), msg, v...)
	}
}

// Error logs a message at ErrorLevel.
func (l *Log) Error(msg string, v ...any) {
	if ErrorLevel >= l.lvl {
		var prefix bytes.Buffer

		prefix.Grow(prefixGrowSize)
		_, _ = errorMarker.Fprint(&prefix, errorPrefix)
		prefix.WriteRune(' ')
		_, _ = errorColor.Fprint(&prefix, l.getTimestamp())

		l.write(&l.errOut, prefix.Bytes(), msg, v...)
	}
}

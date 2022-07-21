// Package logger contains functions for a working with application logging.
package logger

import (
	"bytes"
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

var (
	debugColor     = color.New(color.FgMagenta)              //nolint:gochecknoglobals
	infoColor      = color.New(color.FgBlue)                 //nolint:gochecknoglobals
	warnColor      = color.New(color.FgHiYellow, color.Bold) //nolint:gochecknoglobals
	errorColor     = color.New(color.FgHiRed, color.Bold)    //nolint:gochecknoglobals
	underlineColor = color.New(color.Underline)              //nolint:gochecknoglobals
	extraDataColor = color.New(color.FgWhite)                //nolint:gochecknoglobals

	debugMarker = color.New(color.BgMagenta, color.FgHiMagenta).Sprint(" debug ") + " " //nolint:gochecknoglobals
	infoMarker  = color.New(color.BgBlue, color.FgHiBlue).Sprint("  info ") + " "       //nolint:gochecknoglobals
	warnMarker  = color.New(color.BgHiYellow, color.FgBlack).Sprint("  warn ") + " "    //nolint:gochecknoglobals
	errorMarker = color.New(color.BgHiRed, color.FgHiWhite).Sprint(" error ") + " "     //nolint:gochecknoglobals
)

const timeFormat = "15:04:05.000"

func (*Log) write(out *output, prefix, msg string, v ...any) {
	var buf bytes.Buffer

	buf.Grow(len(prefix) + len(msg) + len(v)*32)

	if prefix != "" {
		buf.WriteString(prefix)
		buf.WriteRune(' ')
	}

	buf.WriteString(msg)
	buf.WriteRune(' ')

	for i, extra := range v {
		buf.WriteString(extraDataColor.Sprint(extra))

		if i < len(v)-1 {
			buf.WriteRune(' ')
		}
	}

	buf.WriteRune('\n')

	out.mu.Lock()
	_, _ = out.to.Write(buf.Bytes())
	out.mu.Unlock()
}

// Debug logs a message at DebugLevel.
func (l *Log) Debug(msg string, v ...any) {
	if DebugLevel >= l.lvl {
		var prefix = debugMarker + debugColor.Sprint(time.Now().Format(timeFormat))

		if _, file, line, ok := runtime.Caller(1); ok {
			prefix += underlineColor.Sprintf(" %s:%d", filepath.Base(file), line)
		}

		l.write(&l.stdOut, prefix, msg, v...)
	}
}

// Info logs a message at InfoLevel.
func (l *Log) Info(msg string, v ...any) {
	if InfoLevel >= l.lvl {
		l.write(&l.stdOut, infoMarker+infoColor.Sprint(time.Now().Format(timeFormat)), msg, v...)
	}
}

// Warn logs a message at WarnLevel.
func (l *Log) Warn(msg string, v ...any) {
	if WarnLevel >= l.lvl {
		l.write(&l.stdOut, warnMarker+warnColor.Sprint(time.Now().Format(timeFormat)), msg, v...)
	}
}

// Error logs a message at ErrorLevel.
func (l *Log) Error(msg string, v ...any) {
	if ErrorLevel >= l.lvl {
		l.write(&l.errOut, errorMarker+errorColor.Sprint(time.Now().Format(timeFormat)), msg, v...)
	}
}

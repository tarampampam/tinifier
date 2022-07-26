// Package logger contains functions for a working with application logging.
package logger

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/pterm/pterm"
)

type (
	Logger interface {
		// Debug logs a message at DebugLevel.
		Debug(msg string, v ...Extra)

		// Info logs a message at InfoLevel.
		Info(msg string, v ...Extra)

		// Success logs a success message at InfoLevel.
		Success(msg string, v ...Extra)

		// Warn logs a message at WarnLevel.
		Warn(msg string, v ...Extra)

		// Error logs a message at ErrorLevel.
		Error(msg string, v ...Extra)
	}

	Extra interface {
		// Key returns the key of the extra field.
		Key() string

		// Value returns the value of the extra field.
		Value() any
	}
)

// extra is a helper struct for that implements Extra interface.
type extra struct {
	key   string
	value any
}

func (e *extra) Key() string { return e.key }
func (e *extra) Value() any  { return e.value }

// With returns an Extra logger field.
func With(key string, value any) Extra { return &extra{key: key, value: value} }

// LogOption is a function that can be used to modify a Log.
type LogOption func(*Log)

// WithStdOut sets the writer for standard output.
func WithStdOut(w io.Writer) LogOption {
	return func(l *Log) {
		l.debug.Writer = w
		l.info.Writer = w
		l.success.Writer = w
		l.warn.Writer = w
	}
}

// WithStdErr sets the writer for standard error.
func WithStdErr(w io.Writer) LogOption { return func(l *Log) { l.error.Writer = w } }

// Log is a logger that logs messages at specified level.
type Log struct {
	debug   *pterm.PrefixPrinter
	info    *pterm.PrefixPrinter
	success *pterm.PrefixPrinter
	warn    *pterm.PrefixPrinter
	error   *pterm.PrefixPrinter

	mu  sync.Mutex
	lvl Level
}

// New creates a new Logger with specified level.
func New(lvl Level, opts ...LogOption) *Log {
	var log = &Log{
		debug: &pterm.PrefixPrinter{
			MessageStyle: &pterm.Style{pterm.FgDefault, pterm.BgDefault},
			Prefix:       pterm.Prefix{Text: "debug", Style: &pterm.Style{pterm.BgGray, pterm.FgDefault}},
			Writer:       os.Stdout,
		},
		info: &pterm.PrefixPrinter{
			MessageStyle: &pterm.Style{pterm.FgDefault, pterm.BgDefault},
			Prefix:       pterm.Prefix{Text: " info", Style: &pterm.Style{pterm.BgBlue, pterm.FgLightWhite}},
			Writer:       os.Stdout,
		},
		success: &pterm.PrefixPrinter{
			MessageStyle: &pterm.Style{pterm.FgDefault, pterm.BgDefault},
			Prefix:       pterm.Prefix{Text: "   ok", Style: &pterm.Style{pterm.BgGreen, pterm.FgLightWhite}},
			Writer:       os.Stdout,
		},
		warn: &pterm.PrefixPrinter{
			MessageStyle: &pterm.Style{pterm.FgDefault, pterm.BgDefault},
			Prefix:       pterm.Prefix{Text: " warn", Style: &pterm.Style{pterm.BgYellow, pterm.FgLightWhite}},
			Writer:       os.Stdout,
		},
		error: &pterm.PrefixPrinter{
			MessageStyle: &pterm.Style{pterm.FgDefault, pterm.BgDefault},
			Prefix:       pterm.Prefix{Text: "error", Style: &pterm.Style{pterm.BgLightRed, pterm.FgLightWhite}},
			Writer:       os.Stderr,
		},

		lvl: lvl,
	}

	for _, opt := range opts {
		opt(log)
	}

	return log
}

// NewNop creates a no-op Logger.
func NewNop() *Log {
	var none = pterm.PrefixPrinter{Writer: io.Discard}

	return &Log{
		debug:   &none,
		info:    &none,
		success: &none,
		warn:    &none,
		error:   &none,
	}
}

func (l *Log) write(printer *pterm.PrefixPrinter, prefix string, msg string, extra ...Extra) {
	var buf bytes.Buffer

	buf.Grow(len(prefix) + len(msg) + len(extra)*64)

	if len(prefix) > 0 {
		buf.WriteString(prefix)
		buf.WriteRune(' ')
	}

	buf.WriteString(msg)

	if len(extra) > 0 {
		buf.WriteRune('\n')

		var extraBuf bytes.Buffer

		for i, v := range extra {
			extraBuf.Grow(len(v.Key()) + 32) //nolint:gomnd

			var isLast = i >= len(extra)-1

			extraBuf.WriteRune(' ')

			if !isLast {
				extraBuf.WriteRune('├')
			} else {
				extraBuf.WriteRune('└')
			}

			extraBuf.WriteRune('─')
			extraBuf.WriteRune(' ')
			extraBuf.WriteString(pterm.Bold.Sprint(v.Key()))
			extraBuf.WriteRune(':')
			extraBuf.WriteRune(' ')
			extraBuf.WriteString(fmt.Sprint(v.Value()))

			if !isLast {
				extraBuf.WriteRune('\n')
			}

			buf.WriteString(pterm.FgGray.Sprint(extraBuf.String()))

			extraBuf.Reset()
		}
	}

	l.mu.Lock()
	printer.Println(buf.String())
	l.mu.Unlock()
}

// ts returns the current timestamp.
func (l *Log) ts() string { return time.Now().Format("15:04:05.000") }

// Debug logs a message at DebugLevel.
func (l *Log) Debug(msg string, v ...Extra) {
	if DebugLevel >= l.lvl && l.debug.Writer != io.Discard {
		if _, file, line, ok := runtime.Caller(1); ok {
			v = append([]Extra{With("Caller", filepath.Base(file)+":"+strconv.Itoa(line))}, v...)
		}

		l.write(l.debug, pterm.FgGray.Sprint(l.ts()), msg, v...)
	}
}

// Info logs a message at InfoLevel.
func (l *Log) Info(msg string, v ...Extra) {
	if InfoLevel >= l.lvl && l.info.Writer != io.Discard {
		l.write(l.info, pterm.FgLightBlue.Sprint(l.ts()), msg, v...)
	}
}

// Success logs a success message at InfoLevel.
func (l *Log) Success(msg string, v ...Extra) {
	if InfoLevel >= l.lvl && l.success.Writer != io.Discard {
		l.write(l.success, pterm.FgLightGreen.Sprint(l.ts()), msg, v...)
	}
}

// Warn logs a message at WarnLevel.
func (l *Log) Warn(msg string, v ...Extra) {
	if WarnLevel >= l.lvl && l.warn.Writer != io.Discard {
		l.write(l.warn, pterm.FgLightYellow.Sprint(l.ts()), msg, v...)
	}
}

// Error logs a message at ErrorLevel.
func (l *Log) Error(msg string, v ...Extra) {
	if ErrorLevel >= l.lvl && l.error.Writer != io.Discard {
		l.write(l.error, pterm.FgRed.Sprint(l.ts()), msg, v...)
	}
}

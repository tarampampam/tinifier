package main

import (
	"github.com/logrusorgru/aurora"
	"log"
	"os"
)

type ILogger interface {
	SetVerbose(isEnabled bool)
	SetColors(isEnabled bool)
	Verbose(msg ...interface{})
	Info(msg ...interface{})
	Error(msg ...interface{})
	Panic(v interface{})
	Fatal(msg ...interface{})
	SetOnPanicFunc(f LoggerPanicFunc)
	SetOnExitFunc(f LoggerExitFunc)
}

type Logger struct {
	StdLogger *log.Logger
	ErrLogger *log.Logger
	colors    aurora.Aurora
	isVerbose bool
	useColors bool
	onPanic   LoggerPanicFunc
	onExit    LoggerExitFunc
}

type LoggerExitFunc func(code int)
type LoggerPanicFunc func(v interface{})

// Create new logger instance.
func NewLogger(std *log.Logger, err *log.Logger, isVerbose bool, useColors bool) *Logger {
	r := &Logger{
		StdLogger: std,
		ErrLogger: err,
	}

	r.SetVerbose(isVerbose)
	r.SetColors(useColors)
	r.SetOnExitFunc(func(code int) {
		os.Exit(code)
	})
	r.SetOnPanicFunc(func(v interface{}) {
		panic(v)
	})

	return r
}

// Enable or disable verbose mode.
func (l *Logger) SetVerbose(isEnabled bool) {
	l.isVerbose = isEnabled
}

// Enable or disable verbose color output.
func (l *Logger) SetColors(isEnabled bool) {
	l.useColors = isEnabled
	l.colors = aurora.NewAurora(isEnabled)
}

// Set "panic" function.
func (l *Logger) SetOnPanicFunc(f LoggerPanicFunc) {
	l.onPanic = f
}

// Set "exit" function.
func (l *Logger) SetOnExitFunc(f LoggerExitFunc) {
	l.onExit = f
}

// Output message only if verbose mode is enabled.
func (l *Logger) Verbose(msg ...interface{}) {
	if l.isVerbose {
		l.StdLogger.Println(msg...)
	}
}

// Output info message to the StdOut writer.
func (l *Logger) Info(msg ...interface{}) {
	l.StdLogger.Println(msg...)
}

// Output error message to the StdErr writer.
func (l *Logger) Error(msg ...interface{}) {
	if l.useColors {
		res := make([]interface{}, 0, len(msg))
		for _, v := range msg {
			res = append(res, l.colors.Colorize(v, aurora.BrightFg|aurora.RedFg|aurora.BoldFm))
		}
		l.ErrLogger.Print(res...)
	} else {
		l.ErrLogger.Print(msg...)
	}
}

// Panic is equivalent to l.Print() followed by a call to panic().
func (l *Logger) Panic(v interface{}) {
	l.onPanic(v)
}

// Fatal is equivalent to l.Error() followed by a call to os.Exit(1).
func (l *Logger) Fatal(msg ...interface{}) {
	const prefix string = "[Fatal Error]"

	if l.useColors {
		l.ErrLogger.SetPrefix(l.colors.Colorize(prefix, aurora.RedBg|aurora.WhiteFg|aurora.BoldFm).String() + " ")
	} else {
		l.ErrLogger.SetPrefix(prefix + " ")
	}

	l.Error(msg...)
	l.onExit(1)
}

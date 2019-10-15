package main

import (
	color "github.com/logrusorgru/aurora"
	"log"
	"os"
)

type Logger struct {
	std       *log.Logger
	err       *log.Logger
	isVerbose bool
}

var logger = Logger{}

// Init logger properties
func init() {
	logger.std = log.New(os.Stdout, "", 0)
	logger.err = log.New(os.Stderr, "", 0)
}

// Output message only if verbose mode is enabled.
func (l *Logger) Verbose(msg ...interface{}) {
	if l.isVerbose {
		l.std.Println(msg)
	}
}

// Output info message to the StdOut writer.
func (l *Logger) Info(msg ...interface{}) {
	l.std.Println(msg)
}

// Output error message to the StdErr writer.
func (l *Logger) Error(msg ...interface{}) {
	res := make([]interface{}, 0, len(msg))

	for _, v := range msg {
		res = append(res, color.Colorize(v, color.BrightFg|color.RedFg|color.BoldFm))
	}

	l.err.Print(res...)
	res = nil // free slice
}

// Fatal is equivalent to l.Error() followed by a call to os.Exit(1).
func (l *Logger) Fatal(msg ...interface{}) {
	l.err.SetPrefix(color.Colorize("[FATAL]", color.RedBg|color.WhiteFg|color.BoldFm).String() + " ")
	l.Error(msg...)
	os.Exit(1)
}

// Customize logger prefix(es).
func (l *Logger) SetPrefix(prefix string) {
	l.std.SetPrefix(prefix)
	l.err.SetPrefix(prefix)
}

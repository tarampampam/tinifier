// Package logger contains functions for a working with application logging.
package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/fatih/color"
)

type Logger struct {
	stdLog *log.Logger
	errLog *log.Logger
	lvl    Level
}

var (
	debugColor     = color.New(color.FgMagenta)
	infoColor      = color.New(color.FgBlue)
	warnColor      = color.New(color.FgHiYellow, color.Bold)
	errorColor     = color.New(color.FgHiRed, color.Bold)
	underlineColor = color.New(color.Underline)

	debugMarker = color.New(color.BgMagenta, color.FgHiMagenta).Sprint(" debug ") + " "
	infoMarker  = color.New(color.BgBlue, color.FgHiBlue).Sprint("  info ") + " "
	warnMarker  = color.New(color.BgHiYellow, color.FgBlack).Sprint("  warn ") + " "
	errorMarker = color.New(color.BgHiRed, color.FgHiWhite).Sprint(" error ") + " "
)

const timeFormat = "15:04:05.000"

func New(lvl Level) *Logger {
	const prefix, flag = "", 0

	return &Logger{
		stdLog: log.New(os.Stdout, prefix, flag),
		errLog: log.New(os.Stderr, prefix, flag),
		lvl:    lvl,
	}
}

func NewNop() *Logger {
	const prefix, flag = "", 0

	return &Logger{
		stdLog: log.New(io.Discard, prefix, flag),
		errLog: log.New(io.Discard, prefix, flag),
		lvl:    InfoLevel,
	}
}

func (Logger) write(log *log.Logger, prefix, msg string, v ...any) {
	var args []any

	if prefix != "" {
		args = make([]any, 2, len(v)+2)
		args[0], args[1] = prefix, msg
	} else {
		args = make([]any, 1, len(v)+1)
		args[0] = msg
	}

	args = append(args, v...)

	log.Println(args...)
}

func (l Logger) Debug(msg string, v ...any) {
	if DebugLevel >= l.lvl {
		var prefix = debugMarker + debugColor.Sprint(time.Now().Format(timeFormat))

		if _, file, line, ok := runtime.Caller(1); ok {
			prefix += " " + underlineColor.Sprint(filepath.Base(file)+":"+strconv.Itoa(line))
		}

		l.write(l.stdLog, prefix, msg, v...)
	}
}

func (l Logger) Info(msg string, v ...any) {
	if InfoLevel >= l.lvl {
		l.write(l.stdLog, infoMarker+infoColor.Sprint(time.Now().Format(timeFormat)), msg, v...)
	}
}

func (l Logger) Warn(msg string, v ...any) {
	if WarnLevel >= l.lvl {
		l.write(l.stdLog, warnMarker+warnColor.Sprint(time.Now().Format(timeFormat)), msg, v...)
	}
}

func (l Logger) Error(msg string, v ...any) {
	if ErrorLevel >= l.lvl {
		l.write(l.errLog, errorMarker+errorColor.Sprint(time.Now().Format(timeFormat)), msg, v...)
	}
}

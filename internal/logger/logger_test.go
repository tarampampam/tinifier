package logger_test

import (
	"bytes"
	"log"
	"regexp"
	"sync"
	"testing"

	"github.com/fatih/color"
	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"

	"github.com/tarampampam/tinifier/v4/internal/logger"
)

func TestNewNop(t *testing.T) {
	var l = logger.NewNop()

	out := capturer.CaptureOutput(func() {
		l.Debug("debug msg")
		l.Info("info msg")
		l.Warn("warn msg")
		l.Error("error msg")
	})

	assert.Empty(t, out)
}

// func TestUsage(t *testing.T) {
// 	var (
// 		extra = []any{"foo", 123, struct{}{}, []string{"bar"}}
// 		l     = logger.New(logger.DebugLevel)
// 	)
//
// 	color.NoColor = false
//
// 	l.Debug("debug msg", extra...)
// 	l.Info("info msg", extra...)
// 	l.Warn("warn msg", extra...)
// 	l.Error("error msg", extra...)
// }

func TestLog_Debug(t *testing.T) {
	var colorState = color.NoColor

	color.NoColor = true

	defer func() { color.NoColor = colorState }()

	var (
		stdOut, errOut bytes.Buffer
		extra          = []any{"foo", 123, struct{}{}, []string{"bar"}}
	)

	var l = logger.New(logger.DebugLevel, logger.WithStdOut(&stdOut), logger.WithStdErr(&errOut))

	l.Debug("debug msg", extra...)
	l.Info("info msg", extra...)
	l.Warn("warn msg", extra...)
	l.Error("error msg", extra...)

	var (
		stdOutStr = stdOut.String()
		stdErrStr = errOut.String()
	)

	assert.Contains(t, stdOutStr, "debug msg")
	assert.Contains(t, stdOutStr, "info msg")
	assert.Contains(t, stdOutStr, "warn msg")
	assert.NotContains(t, stdOutStr, "error msg")

	assert.NotContains(t, stdErrStr, "debug msg")
	assert.NotContains(t, stdErrStr, "info msg")
	assert.NotContains(t, stdErrStr, "warn msg")
	assert.Contains(t, stdErrStr, "error msg")

	assert.Regexp(t,
		regexp.MustCompile(`^\s+debug\s+\d{2}:\d{2}:\d{2}\.\d{3}\s\w+\.go:\d+ debug msg \(foo 123 {} \[bar]\)\n`),
		stdOutStr,
	)
}

func TestLog_Info(t *testing.T) {
	var colorState = color.NoColor

	color.NoColor = true

	defer func() { color.NoColor = colorState }()

	var (
		stdOut, errOut bytes.Buffer
		extra          = []any{"foo", 123, struct{}{}, []string{"bar"}}
	)

	var l = logger.New(logger.InfoLevel, logger.WithStdOut(&stdOut), logger.WithStdErr(&errOut))

	l.Debug("debug msg", extra...)
	l.Info("info msg", extra...)
	l.Warn("warn msg", extra...)
	l.Error("error msg", extra...)

	var (
		stdOutStr = stdOut.String()
		stdErrStr = errOut.String()
	)

	assert.NotContains(t, stdOutStr, "debug msg")
	assert.Contains(t, stdOutStr, "info msg")
	assert.Contains(t, stdOutStr, "warn msg")
	assert.NotContains(t, stdOutStr, "error msg")

	assert.NotContains(t, stdErrStr, "debug msg")
	assert.NotContains(t, stdErrStr, "info msg")
	assert.NotContains(t, stdErrStr, "warn msg")
	assert.Contains(t, stdErrStr, "error msg")

	assert.Regexp(t,
		regexp.MustCompile(`^\s+info\s+\d{2}:\d{2}:\d{2}\.\d{3} info msg \(foo 123 {} \[bar]\)\n`),
		stdOutStr,
	)
}

func TestLog_Warn(t *testing.T) {
	var colorState = color.NoColor

	color.NoColor = true

	defer func() { color.NoColor = colorState }()

	var (
		stdOut, errOut bytes.Buffer
		extra          = []any{"foo", 123, struct{}{}, []string{"bar"}}
	)

	var l = logger.New(logger.WarnLevel, logger.WithStdOut(&stdOut), logger.WithStdErr(&errOut))

	l.Debug("debug msg", extra...)
	l.Info("info msg", extra...)
	l.Warn("warn msg", extra...)
	l.Error("error msg", extra...)

	var (
		stdOutStr = stdOut.String()
		stdErrStr = errOut.String()
	)

	assert.NotContains(t, stdOutStr, "debug msg")
	assert.NotContains(t, stdOutStr, "info msg")
	assert.Contains(t, stdOutStr, "warn msg")
	assert.NotContains(t, stdOutStr, "error msg")

	assert.NotContains(t, stdErrStr, "debug msg")
	assert.NotContains(t, stdErrStr, "info msg")
	assert.NotContains(t, stdErrStr, "warn msg")
	assert.Contains(t, stdErrStr, "error msg")

	assert.Regexp(t,
		regexp.MustCompile(`^\s+warn\s+\d{2}:\d{2}:\d{2}\.\d{3} warn msg \(foo 123 {} \[bar]\)\n`),
		stdOutStr,
	)
}

func TestLog_Error(t *testing.T) {
	var colorState = color.NoColor

	color.NoColor = true

	defer func() { color.NoColor = colorState }()

	var (
		stdOut, errOut bytes.Buffer
		extra          = []any{"foo", 123, struct{}{}, []string{"bar"}}
	)

	var l = logger.New(logger.ErrorLevel, logger.WithStdOut(&stdOut), logger.WithStdErr(&errOut))

	l.Debug("debug msg", extra...)
	l.Info("info msg", extra...)
	l.Warn("warn msg", extra...)
	l.Error("error msg", extra...)

	var (
		stdOutStr = stdOut.String()
		stdErrStr = errOut.String()
	)

	assert.Empty(t, stdOutStr)

	assert.NotContains(t, stdErrStr, "debug msg")
	assert.NotContains(t, stdErrStr, "info msg")
	assert.NotContains(t, stdErrStr, "warn msg")
	assert.Contains(t, stdErrStr, "error msg")

	assert.Regexp(t,
		regexp.MustCompile(`^\s+error\s+\d{2}:\d{2}:\d{2}\.\d{3} error msg \(foo 123 {} \[bar]\)\n`),
		stdErrStr,
	)
}

func TestLog_Concurrent(t *testing.T) {
	var (
		stdOut, errOut bytes.Buffer

		l  = logger.New(logger.DebugLevel, logger.WithStdOut(&stdOut), logger.WithStdErr(&errOut))
		wg sync.WaitGroup
	)

	for i := 0; i < 100; i++ {
		wg.Add(4)

		go func() { defer wg.Done(); l.Debug("debug", struct{}{}) }()
		go func() { defer wg.Done(); l.Info("info", struct{}{}) }()
		go func() { defer wg.Done(); l.Warn("warn", struct{}{}) }()
		go func() { defer wg.Done(); l.Error("error", struct{}{}) }()
	}

	wg.Wait()

	assert.NotEmpty(t, stdOut.String())
	assert.NotEmpty(t, errOut.String())
}

// BenchmarkLog_Print-8         	  645908	      1592 ns/op	     764 B/op	      22 allocs/op
func BenchmarkLog_Print(b *testing.B) { // our logger is really slow
	b.ReportAllocs()

	var (
		buf bytes.Buffer
		l   = logger.New(logger.InfoLevel, logger.WithStdErr(&buf), logger.WithStdOut(&buf))
	)

	buf.Grow(1024)

	for i := 0; i < b.N; i++ {
		l.Info("message", struct{}{})
	}
}

// BenchmarkStdLibLog_Print-8   	 5297905	       339.6 ns/op	     120 B/op	       1 allocs/op
func BenchmarkStdLibLog_Print(b *testing.B) {
	b.ReportAllocs()

	var (
		buf bytes.Buffer
		l   = log.New(&buf, "log", log.LstdFlags)
	)

	buf.Grow(1024)

	for i := 0; i < b.N; i++ {
		l.Println("message", struct{}{})
	}
}

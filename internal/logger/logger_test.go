package logger_test

import (
	"bytes"
	"log"
	"regexp"
	"sync"
	"testing"

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

func TestLog_Debug(t *testing.T) {
	var (
		extra = []any{"foo", 123, struct{}{}, []string{"bar"}}
		do    = func() {
			var l = logger.New(logger.DebugLevel)

			l.Debug("debug msg", extra...)
			l.Info("info msg", extra...)
			l.Warn("warn msg", extra...)
			l.Error("error msg", extra...)
		}
	)

	var (
		stdout = capturer.CaptureStdout(do)
		stderr = capturer.CaptureStderr(do)
	)

	assert.Contains(t, stdout, "debug msg")
	assert.Contains(t, stdout, "info msg")
	assert.Contains(t, stdout, "warn msg")
	assert.NotContains(t, stdout, "error msg")

	assert.NotContains(t, stderr, "debug msg")
	assert.NotContains(t, stderr, "info msg")
	assert.NotContains(t, stderr, "warn msg")
	assert.Contains(t, stderr, "error msg")

	assert.Regexp(t,
		regexp.MustCompile(`^\s+debug\s+\d{2}:\d{2}:\d{2}\.\d{3}\s\w+\.go:\d+ debug msg \(foo 123 {} \[bar]\)\n`),
		stdout,
	)
}

func TestLog_Info(t *testing.T) {
	var (
		extra = []any{"foo", 123, struct{}{}, []string{"bar"}}
		do    = func() {
			var l = logger.New(logger.InfoLevel)

			l.Debug("debug msg", extra...)
			l.Info("info msg", extra...)
			l.Warn("warn msg", extra...)
			l.Error("error msg", extra...)
		}
	)

	var (
		stdout = capturer.CaptureStdout(do)
		stderr = capturer.CaptureStderr(do)
	)

	assert.NotContains(t, stdout, "debug msg")
	assert.Contains(t, stdout, "info msg")
	assert.Contains(t, stdout, "warn msg")
	assert.NotContains(t, stdout, "error msg")

	assert.NotContains(t, stderr, "debug msg")
	assert.NotContains(t, stderr, "info msg")
	assert.NotContains(t, stderr, "warn msg")
	assert.Contains(t, stderr, "error msg")

	assert.Regexp(t,
		regexp.MustCompile(`^\s+info\s+\d{2}:\d{2}:\d{2}\.\d{3} info msg \(foo 123 {} \[bar]\)\n`),
		stdout,
	)
}

func TestLog_Warn(t *testing.T) {
	var (
		extra = []any{"foo", 123, struct{}{}, []string{"bar"}}
		do    = func() {
			var l = logger.New(logger.WarnLevel)

			l.Debug("debug msg", extra...)
			l.Info("info msg", extra...)
			l.Warn("warn msg", extra...)
			l.Error("error msg", extra...)
		}
	)

	var (
		stdout = capturer.CaptureStdout(do)
		stderr = capturer.CaptureStderr(do)
	)

	assert.NotContains(t, stdout, "debug msg")
	assert.NotContains(t, stdout, "info msg")
	assert.Contains(t, stdout, "warn msg")
	assert.NotContains(t, stdout, "error msg")

	assert.NotContains(t, stderr, "debug msg")
	assert.NotContains(t, stderr, "info msg")
	assert.NotContains(t, stderr, "warn msg")
	assert.Contains(t, stderr, "error msg")

	assert.Regexp(t,
		regexp.MustCompile(`^\s+warn\s+\d{2}:\d{2}:\d{2}\.\d{3} warn msg \(foo 123 {} \[bar]\)\n`),
		stdout,
	)
}

func TestLog_Error(t *testing.T) {
	var (
		extra = []any{"foo", 123, struct{}{}, []string{"bar"}}
		do    = func() {
			var l = logger.New(logger.ErrorLevel)

			l.Debug("debug msg", extra...)
			l.Info("info msg", extra...)
			l.Warn("warn msg", extra...)
			l.Error("error msg", extra...)
		}
	)

	var (
		stdout = capturer.CaptureStdout(do)
		stderr = capturer.CaptureStderr(do)
	)

	assert.Empty(t, stdout)

	assert.NotContains(t, stderr, "debug msg")
	assert.NotContains(t, stderr, "info msg")
	assert.NotContains(t, stderr, "warn msg")
	assert.Contains(t, stderr, "error msg")

	assert.Regexp(t,
		regexp.MustCompile(`^\s+error\s+\d{2}:\d{2}:\d{2}\.\d{3} error msg \(foo 123 {} \[bar]\)\n`),
		stderr,
	)
}

func TestLog_Concurrent(t *testing.T) {
	var out = capturer.CaptureOutput(func() {
		var (
			l  = logger.New(logger.DebugLevel)
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
	})

	assert.NotEmpty(t, out)
}

// BenchmarkLog_Print-8         	 1119032	      1061 ns/op	     423 B/op	      16 allocs/op
func BenchmarkLog_Print(b *testing.B) { // our logger is slow
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

// BenchmarkStdLibLog_Print-8   	 5205457	       236.7 ns/op	     122 B/op	       1 allocs/op
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

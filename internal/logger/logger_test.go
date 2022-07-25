package logger_test

import (
	"bytes"
	"regexp"
	"sync"
	"testing"

	"github.com/kami-zh/go-capturer"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"

	"github.com/tarampampam/tinifier/v4/internal/logger"
)

func TestNewNop(t *testing.T) {
	var l = logger.NewNop()

	out := capturer.CaptureOutput(func() {
		l.Debug("debug msg", logger.With("foo", struct{}{}))
		l.Info("info msg", logger.With("foo", 123))
		l.Warn("warn msg", logger.With("foo", []string{"bar"}))
		l.Error("error msg", logger.With("foo", "bar"))
	})

	assert.Empty(t, out)
}

// func TestUsage(t *testing.T) {
// 	var (
// 		extra = []logger.Extra{
// 			logger.With("string", "foo"),
// 			logger.With("int", 123),
// 			logger.With("struct", struct{}{}),
// 			logger.With("slice", []string{"bar"}),
// 		}
// 		l = logger.New(logger.DebugLevel)
// 	)
//
// 	color.NoColor = false
//
// 	l.Debug("debug msg", extra...)
// 	l.Info("info msg", extra...)
// 	l.Success("success msg", extra...)
// 	l.Warn("warn msg", extra...)
// 	l.Error("error msg", extra...)
// }

func TestLog_Debug(t *testing.T) {
	pterm.DisableColor()
	defer pterm.EnableColor()

	var (
		stdOut, errOut bytes.Buffer
		extra          = []logger.Extra{
			logger.With("string", "foo"),
			logger.With("int", 123),
			logger.With("struct", struct{}{}),
			logger.With("slice", []string{"bar"}),
		}
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
		regexp.MustCompile(`^\s+debug\s+\d{2}:\d{2}:\d{2}\.\d{3} debug msg\n`),
		stdOutStr,
	)
	assert.Contains(t, stdOutStr, "string: foo")
	assert.Contains(t, stdOutStr, "int: 123")
	assert.Contains(t, stdOutStr, "struct: {}")
	assert.Contains(t, stdOutStr, "slice: [bar]")
}

func TestLog_Info(t *testing.T) {
	pterm.DisableColor()
	defer pterm.EnableColor()

	var (
		stdOut, errOut bytes.Buffer
		extra          = []logger.Extra{
			logger.With("string", "foo"),
			logger.With("int", 123),
			logger.With("struct", struct{}{}),
			logger.With("slice", []string{"bar"}),
		}
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
		regexp.MustCompile(`^\s+info\s+\d{2}:\d{2}:\d{2}\.\d{3} info msg\n`),
		stdOutStr,
	)
	assert.Contains(t, stdOutStr, "string: foo")
	assert.Contains(t, stdOutStr, "int: 123")
	assert.Contains(t, stdOutStr, "struct: {}")
	assert.Contains(t, stdOutStr, "slice: [bar]")
}

func TestLog_Warn(t *testing.T) {
	pterm.DisableColor()
	defer pterm.EnableColor()

	var (
		stdOut, errOut bytes.Buffer
		extra          = []logger.Extra{
			logger.With("string", "foo"),
			logger.With("int", 123),
			logger.With("struct", struct{}{}),
			logger.With("slice", []string{"bar"}),
		}
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
		regexp.MustCompile(`^\s+warn\s+\d{2}:\d{2}:\d{2}\.\d{3} warn msg\n`),
		stdOutStr,
	)
	assert.Contains(t, stdOutStr, "string: foo")
	assert.Contains(t, stdOutStr, "int: 123")
	assert.Contains(t, stdOutStr, "struct: {}")
	assert.Contains(t, stdOutStr, "slice: [bar]")
}

func TestLog_Error(t *testing.T) {
	pterm.DisableColor()
	defer pterm.EnableColor()

	var (
		stdOut, errOut bytes.Buffer
		extra          = []logger.Extra{
			logger.With("string", "foo"),
			logger.With("int", 123),
			logger.With("struct", struct{}{}),
			logger.With("slice", []string{"bar"}),
		}
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
		regexp.MustCompile(`^\s+error\s+\d{2}:\d{2}:\d{2}\.\d{3} error msg\n`),
		stdErrStr,
	)
	assert.Contains(t, stdErrStr, "string: foo")
	assert.Contains(t, stdErrStr, "int: 123")
	assert.Contains(t, stdErrStr, "struct: {}")
	assert.Contains(t, stdErrStr, "slice: [bar]")
}

func TestLog_Concurrent(t *testing.T) {
	var (
		stdOut, errOut bytes.Buffer

		l  = logger.New(logger.DebugLevel, logger.WithStdOut(&stdOut), logger.WithStdErr(&errOut))
		wg sync.WaitGroup
	)

	for i := 0; i < 100; i++ {
		wg.Add(4)

		go func() { defer wg.Done(); l.Debug("debug", logger.With("struct", struct{}{})) }()
		go func() { defer wg.Done(); l.Info("info", logger.With("struct", struct{}{})) }()
		go func() { defer wg.Done(); l.Warn("warn", logger.With("struct", struct{}{})) }()
		go func() { defer wg.Done(); l.Error("error", logger.With("struct", struct{}{})) }()
	}

	wg.Wait()

	assert.NotEmpty(t, stdOut.String())
	assert.NotEmpty(t, errOut.String())
}

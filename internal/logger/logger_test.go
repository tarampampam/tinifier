package logger_test

import (
	"testing"
	"time"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tarampampam/tinifier/v4/internal/logger"
)

func TestNewDebugLevel(t *testing.T) {
	output := capturer.CaptureStderr(func() {
		log, err := logger.New(logger.DebugLevel)
		require.NoError(t, err)

		log.Debug("dbg msg")
		log.Info("inf msg")
		log.Error("err msg")
	})

	assert.Contains(t, output, time.Now().Format("15:04:05"))
	assert.Regexp(t, `\t.+info.+\tinf msg`, output)
	assert.Regexp(t, `\t.+info.+\t.+logger_test\.go:\d+\tinf msg`, output)
	assert.Contains(t, output, "dbg msg")
	assert.Contains(t, output, "err msg")
}

func TestNewErrorLevel(t *testing.T) {
	output := capturer.CaptureStderr(func() {
		log, err := logger.New(logger.ErrorLevel)
		require.NoError(t, err)

		log.Debug("dbg msg")
		log.Info("inf msg")
		log.Error("err msg")
	})

	assert.NotContains(t, output, "inf msg")
	assert.NotContains(t, output, "dbg msg")
	assert.Contains(t, output, "err msg")
}

func TestNewErrors(t *testing.T) {
	_, err := logger.New(logger.Level(127))
	require.EqualError(t, err, "unsupported logging level")
}

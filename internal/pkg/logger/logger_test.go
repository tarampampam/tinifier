package logger_test

import (
	"testing"
	"time"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/tinifier/v3/internal/pkg/logger"
)

func TestNewNotVerboseDebug(t *testing.T) {
	output := capturer.CaptureStderr(func() {
		log, err := logger.New(false, false)
		assert.NoError(t, err)

		log.Info("inf msg")
		log.Debug("dbg msg")
		log.Error("err msg")
	})

	assert.Contains(t, output, time.Now().Format("15:04:05"))
	assert.Regexp(t, `\t.+info.+\tinf msg`, output)
	assert.NotContains(t, output, "dbg msg")
	assert.Contains(t, output, "err msg")
}

func TestNewVerboseDebug(t *testing.T) {
	output := capturer.CaptureStderr(func() {
		log, err := logger.New(true, true)
		assert.NoError(t, err)

		log.Info("inf msg")
		log.Debug("dbg msg")
		log.Error("err msg")
	})

	assert.Contains(t, output, time.Now().Format("15:04:05"))
	assert.Regexp(t, `\t.+info.+\t.+logger_test\.go:\d+\tinf msg`, output)
	assert.Contains(t, output, "dbg msg")
	assert.Contains(t, output, "err msg")
}

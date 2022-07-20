package logger_test

import (
	"testing"

	"github.com/fatih/color"

	"github.com/tarampampam/tinifier/v4/internal/logger"
)

func TestLogger_Debug(t *testing.T) {
	log := logger.New(logger.DebugLevel)
	color.NoColor = false

	log.Debug("debug level", "asd", 123, struct{}{})
	log.Info("info level", "asd", 123, struct{}{})
	log.Warn("warn level", "asd", 123, struct{}{})
	log.Error("error level", "asd", 123, struct{}{})
}

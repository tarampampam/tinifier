package main

import (
	"os"
	"path/filepath"

	"github.com/tarampampam/tinifier/internal/pkg/cli"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	var atomicLogLevel, logEncoderConfig = zap.NewAtomicLevel(), zap.NewDevelopmentEncoderConfig()

	logEncoderConfig.EncodeLevel = zapcore.LowercaseColorLevelEncoder
	logEncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")

	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(logEncoderConfig),
		zapcore.Lock(os.Stdout),
		atomicLogLevel,
	))

	defer func() {
		/*
			if err := logger.Sync(); err != nil {
				fmt.Println(err.Error()) // FIXME error is here, when `fmt.Println()` somewhere in code
			}
		*/
		_ = logger.Sync()
	}()

	var cmd = cli.NewCommand(logger, &atomicLogLevel, filepath.Base(os.Args[0]))

	if err := cmd.Execute(); err != nil {
		logger.Fatal(err.Error()) // `os.Exit(1)` here
	}
}

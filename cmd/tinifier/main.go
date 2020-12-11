package main

import (
	"os"
	"path/filepath"

	"github.com/tarampampam/tinifier/internal/pkg/cli"

	"github.com/sirupsen/logrus"
)

func main() {
	var (
		logger = logrus.New()
		cmd    = cli.NewCommand(logger, filepath.Base(os.Args[0]))
	)

	// configure logger (setup global properties)
	logger.SetOutput(os.Stdout) // not sure here
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:          true,
		TimestampFormat:        "15:04:05.000",
		DisableLevelTruncation: true,
		PadLevelText:           true,
	})

	if err := cmd.Execute(); err != nil {
		logger.Fatal(err) // `os.Exit(1)` here
	}
}

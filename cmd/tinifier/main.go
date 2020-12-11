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

	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05.000",
		DisableLevelTruncation: true,
	})

	if err := cmd.Execute(); err != nil {
		logger.Fatal(err) // `os.Exit(1)` here
	}
}

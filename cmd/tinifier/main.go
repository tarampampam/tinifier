package main

import (
	"os"
	"path/filepath"
	"tinifier/internal/pkg/cli"

	"github.com/sirupsen/logrus"
)

func main() {
	var (
		logger = logrus.New()
		cmd    = cli.NewCommand(logger, filepath.Base(os.Args[0]))
	)

	if err := cmd.Execute(); err != nil {
		logger.Fatal(err) // `os.Exit(1)` here
	}
}

// Package cli contains CLI command handlers.
package cli

import (
	"strings"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/tarampampam/tinifier/v4/internal/cli/compress"
	"github.com/tarampampam/tinifier/v4/internal/cli/quota"
	"github.com/tarampampam/tinifier/v4/internal/env"
	"github.com/tarampampam/tinifier/v4/internal/logger"
	"github.com/tarampampam/tinifier/v4/internal/version"
)

// NewApp creates new console application.
func NewApp() *cli.App {
	const (
		logLevelFlagName = "log-level"
		defaultLogLevel  = logger.InfoLevel
	)

	// create "default" logger (will be overwritten later with customized)
	var log = zap.NewNop()

	return &cli.App{
		Usage: "CLI client for images compressing using tinypng.com API",
		Before: func(c *cli.Context) error {
			var (
				logLevel = defaultLogLevel //nolint:ineffassign
				err      error
			)

			// parse logging level
			if logLevel, err = logger.ParseLevel([]byte(c.String(logLevelFlagName))); err != nil {
				return err
			}

			configured, err := logger.New(logLevel) // create new logger instance
			if err != nil {
				return err
			}

			*log = *configured // replace "default" logger with customized

			return nil
		},
		Commands: []*cli.Command{
			quota.NewCommand(log),
			compress.NewCommand(log),
		},
		Flags: []cli.Flag{ // global flags
			&cli.StringFlag{
				Name:    logLevelFlagName,
				Value:   defaultLogLevel.String(),
				Usage:   "logging level (" + strings.Join(logger.AllLevelStrings(), "|") + ")",
				EnvVars: []string{env.LogLevel.String()},
			},
		},
		Version: version.Version(),
	}
}

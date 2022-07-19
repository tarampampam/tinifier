// Package cli contains CLI command handlers.
package cli

import (
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/tarampampam/tinifier/v4/internal/cli/compress"
	"github.com/tarampampam/tinifier/v4/internal/cli/quota"
	"github.com/tarampampam/tinifier/v4/internal/env"
	"github.com/tarampampam/tinifier/v4/internal/logger"
	"github.com/tarampampam/tinifier/v4/internal/version"
)

// NewApp creates new console application.
func NewApp() *cli.App {
	const (
		logLevelFlagName  = "log-level"
		logFormatFlagName = "log-format"

		defaultLogLevel  = logger.InfoLevel
		defaultLogFormat = logger.ConsoleFormat
	)

	// create "default" logger (will be overwritten later with customized)
	log, _ := logger.New(defaultLogLevel, defaultLogFormat)

	return &cli.App{
		Usage: "CLI client for images compressing using tinypng.com API",
		Before: func(c *cli.Context) error {
			_ = log.Sync() // sync previous logger instance

			var (
				logLevel, logFormat = defaultLogLevel, defaultLogFormat //nolint:ineffassign
				err                 error
			)

			// parse logging level
			if logLevel, err = logger.ParseLevel([]byte(c.String(logLevelFlagName))); err != nil {
				return err
			}

			// parse logging format
			if logFormat, err = logger.ParseFormat([]byte(c.String(logFormatFlagName))); err != nil {
				return err
			}

			configured, err := logger.New(logLevel, logFormat) // create new logger instance
			if err != nil {
				return err
			}

			*log = *configured // replace "default" logger with customized

			return nil
		},
		After: func(c *cli.Context) error {
			// error ignoring reasons:
			// - <https://github.com/uber-go/zap/issues/772>
			// - <https://github.com/uber-go/zap/issues/328>
			_ = log.Sync()

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
			&cli.StringFlag{
				Name:    logFormatFlagName,
				Value:   defaultLogFormat.String(),
				Usage:   "logging format (" + strings.Join(logger.AllFormatStrings(), "|") + ")",
				EnvVars: []string{env.LogFormat.String()},
			},
		},
		Version: version.Version(),
	}
}

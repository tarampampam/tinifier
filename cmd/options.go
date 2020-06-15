package cmd

import (
	"tinifier/cmd/compress"
	"tinifier/cmd/version"
)

type (
	appOptions struct {
		IsVerbose bool `short:"v" long:"verbose" description:"Enable verbosity mode"`
		IsDebug   bool `short:"d" long:"debug" description:"Enable debug mode"`
	}

	subCommands struct {
		Version  version.Command  `command:"version" alias:"v" description:"Display application version"`
		Compress compress.Command `command:"compress" alias:"c" description:"Compress images"`
	}
)

type Options struct {
	appOptions
	subCommands
}

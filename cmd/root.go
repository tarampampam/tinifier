package cmd

import (
	"tinifier/cmd/compress"
	"tinifier/cmd/quota"
	"tinifier/cmd/version"
)

// Root is a basic commands struct.
type Root struct {
	Version  version.Command  `command:"version" alias:"v" description:"Display application version"`
	Compress compress.Command `command:"compress" alias:"c" description:"Compress images"`
	Quota    quota.Command    `command:"quota" alias:"q" description:"Get currently used quota"`
}

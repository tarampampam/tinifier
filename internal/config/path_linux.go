package config

import (
	"os"
	"path/filepath"
)

// osSpecificConfigDirPath determines the path to the directory where the configuration file is looked for by default
// on the Linux operating system.
func osSpecificConfigDirPath() string {
	if v, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		return v
	}

	if v, ok := os.LookupEnv("HOME"); ok {
		return filepath.Join(v, ".config")
	}

	return ""
}

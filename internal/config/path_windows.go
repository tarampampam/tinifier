package config

import (
	"os"
)

// osSpecificConfigDirPath determines the path to the directory where the configuration file is looked for by default
// on the Windows operating system.
func osSpecificConfigDirPath() string {
	if v, ok := os.LookupEnv("AppData"); ok {
		return v
	}

	return ""
}

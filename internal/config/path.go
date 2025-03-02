package config

import "os"

// FileName holds the name of the configuration file.
const FileName = "tinifier.yml"

// DefaultDirPathEnvName used to override the default directory path (useful for docs generation purposes).
const DefaultDirPathEnvName = "DEFAULT_CONFIG_FILE_DIR"

// DefaultDirPath returns the default directory path where the configuration file is looked for by default.
// Only in case of exception, this function returns an empty string.
func DefaultDirPath() string {
	if v, ok := os.LookupEnv(DefaultDirPathEnvName); ok {
		return v
	}

	if v := osSpecificConfigDirPath(); v != "" {
		return v
	}

	if v, err := os.Getwd(); err == nil {
		return v // fallback to the current working directory
	}

	return "" // no default path
}

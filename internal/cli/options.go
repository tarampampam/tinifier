package cli

import (
	"fmt"
	"os"

	"gh.tarampamp.am/tinifier/v5/internal/config"
)

type options struct {
	ApiKeys           []string
	FileExtensions    []string
	ThreadsCount      uint
	MaxErrorsToStop   uint
	Recursive         bool
	UpdateFileModDate bool
}

func newOptionsWithDefaults() options {
	return options{
		FileExtensions:    []string{"png", "PNG", "jpeg", "JPEG", "jpg", "JPG"},
		ThreadsCount:      16, //nolint:mnd
		MaxErrorsToStop:   10, //nolint:mnd
		Recursive:         false,
		UpdateFileModDate: false,
	}
}

// UpdateFromConfigFile loads the configuration from the file(s) and applies it to the options.
func (o *options) UpdateFromConfigFile(filePath string) error {
	if filePath == "" {
		return nil
	}

	if stat, err := os.Stat(filePath); err != nil || stat.IsDir() {
		return nil // skip missing files and directories
	}

	var cfg config.Config

	if err := cfg.FromFile(filePath); err != nil {
		return fmt.Errorf("failed to load the configuration file: %w", err)
	}

	setIfSourceNotNil(&o.ApiKeys, cfg.ApiKeys)

	return nil
}

// setIfSourceNotNil sets the target value to the source value if both are not nil.
func setIfSourceNotNil[T any](target, source *T) {
	if target == nil || source == nil {
		return
	}

	*target = *source
}

func (o *options) Validate() error {
	if len(o.ApiKeys) == 0 {
		return fmt.Errorf("API keys list cannot be empty")
	}

	if len(o.FileExtensions) == 0 {
		return fmt.Errorf("extensions list cannot be empty")
	}

	if o.ThreadsCount == 0 {
		return fmt.Errorf("threads count cannot be zero")
	}

	return nil
}

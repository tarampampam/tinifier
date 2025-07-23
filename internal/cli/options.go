package cli

import (
	"fmt"
	"os"
	"time"

	"gh.tarampamp.am/tinifier/v5/internal/config"
)

type options struct {
	ApiKeys             []string
	FileExtensions      []string
	ThreadsCount        uint
	MaxErrorsToStop     uint
	RetryAttempts       uint
	DelayBetweenRetries time.Duration
	Recursive           bool
	SkipIfDiffLessThan  float64 // in percents [0.00 - 100.00]
	PreserveTime        bool
	KeepOriginalFile    bool
}

func newOptionsWithDefaults() options {
	return options{
		FileExtensions:      []string{"png", "jpeg", "jpg", "webp", "avif"},
		ThreadsCount:        16, //nolint:mnd
		MaxErrorsToStop:     10, //nolint:mnd
		RetryAttempts:       3,  //nolint:mnd
		DelayBetweenRetries: time.Second,
		Recursive:           false,
		SkipIfDiffLessThan:  1, // 1.00% by default
		PreserveTime:        false,
		KeepOriginalFile:    false,
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
	// add other fields here

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

	if o.RetryAttempts == 0 {
		return fmt.Errorf("retry attempts cannot be zero")
	}

	if o.DelayBetweenRetries < 0 {
		return fmt.Errorf("delay between retries cannot be negative")
	}

	if o.ThreadsCount == 0 {
		return fmt.Errorf("threads count cannot be zero")
	}

	return nil
}

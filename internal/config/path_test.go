package config_test

import (
	"testing"

	"gh.tarampamp.am/tinifier/v5/internal/config"
)

func TestDefaultDirPath(t *testing.T) {
	t.Parallel()

	if config.DefaultDirPath() == "" {
		t.Error("DefaultDirPath is empty")
	}
}

package files

import (
	"os"
	"path/filepath"
)

// FilterMissed drops any strings, that not points to existing file or directory.
func FilterMissed(paths []string) []string {
	result := make([]string, 0)

	for _, path := range paths {
		// extract absolute path to the target
		if absPath, err := filepath.Abs(path); err == nil {
			// try to read file/dir info
			if _, err := os.Stat(absPath); err == nil {
				result = append(result, absPath)
			}
		}
	}

	return result
}

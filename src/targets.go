package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type ITargets interface {
	Load(targets []string, extensions *[]string) *[]string
}

type Targets struct {
	targets []string
	Files   []string
}

// Create new targets instance.
func NewTargets() *Targets {
	return &Targets{}
}

// Load and convert targets into files paths
func (t *Targets) Load(targets []string, extensions *[]string) *[]string {
	// Set raw targets list
	t.targets = targets
	// Read files in targets and assign into temporary variable
	var filesList, _ = t.targetsToFiles(&t.targets)
	// Filter files list using file extensions and assign result to the property
	t.Files = t.filterFilesUsingExtensions(filesList, extensions)

	return &t.Files
}

// Convert targets into file paths slice. If target points to the directory - directory files will be read and returned
// (with absolute path). If file - file absolute path will be returned. Any invalid value (path to the non-existing
// file, etc) - will be skipped.
func (t *Targets) targetsToFiles(targets *[]string) (result []string, lastError error) {
	// Iterate passed targets
	for _, path := range *targets {
		// Extract absolute path to the target
		if absPath, err := filepath.Abs(path); err == nil {
			// If file stats extracted successful
			if info, err := os.Stat(absPath); err == nil {
				switch mode := info.Mode(); {
				case mode.IsDir(): // If directory found - run files iterator inside it
					if files, err := ioutil.ReadDir(absPath); err == nil {
						for _, file := range files {
							if file.Mode().IsRegular() {
								if abs, err := filepath.Abs(absPath + "/" + file.Name()); err == nil {
									result = append(result, abs)
								}
							}
						}
					} else {
						lastError = err
					}

				case mode.IsRegular(): // If regular file found - append it into result
					result = append(result, absPath)
				}
			}
		} else {
			lastError = err
		}
	}

	return result, lastError
}

// Make files slice filtering using extensions slice. Extension can be combined (delimiter is ",").
func (t *Targets) filterFilesUsingExtensions(files []string, extensions *[]string) (result []string) {
	const delimiter = ","

	for _, path := range files {
		for _, extension := range *extensions {
			if strings.Contains(extension, delimiter) {
				for _, subExtension := range strings.Split(extension, delimiter) {
					if strings.HasSuffix(path, subExtension) {
						result = append(result, path)
					}
				}
			} else {
				if strings.HasSuffix(path, extension) {
					result = append(result, path)
				}
			}
		}
	}

	return result
}

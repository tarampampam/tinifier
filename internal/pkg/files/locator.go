package files

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Finder interface {
	// Find search for files somewhere.
	Find(recursive bool) ([]string, error)
}

type Locator struct {
	// locations list (files or directories)
	locations []string

	// file extensions map (for filtering, eg.: `zip`, `tar`)
	extMap map[string]struct{}
}

// NewLocator creates new files locator.
func NewLocator(locations []string, extensions []string) (*Locator, error) {
	if len(locations) < 1 {
		return nil, errors.New("empty locations list")
	}

	if len(extensions) < 1 {
		return nil, errors.New("empty extensions list")
	}

	l := &Locator{
		locations: locations,
		extMap:    make(map[string]struct{}),
	}

	// convert extensions into map (for faster searching)
	for i := range extensions {
		l.extMap[extensions[i]] = struct{}{}
	}

	return l, nil
}

func (l *Locator) hasCorrectExtension(info os.FileInfo) bool {
	if ext := filepath.Ext(info.Name()); ext != "" { // skip files without extension (eg.: `Dockerfile`)
		if _, ok := l.extMap[ext[1:]]; ok { // ext[1:] for dot (`.`) dropping
			return true
		}
	}

	return false
}

func (l *Locator) Find(recursive bool) ([]string, error) { //nolint:gocognit
	fileList := make(map[string]struct{}) // map is needed to prevent duplicates "out of the box"

	for _, location := range l.locations {
		info, err := os.Stat(location)
		if err != nil {
			return nil, err
		}

		switch mode := info.Mode(); {
		case mode.IsRegular() && l.hasCorrectExtension(info):
			fileList[location] = struct{}{}

		case mode.IsDir():
			if recursive {
				if err := filepath.Walk(location, func(path string, info os.FileInfo, err error) error {
					if info.Mode().IsRegular() && l.hasCorrectExtension(info) {
						fileList[path] = struct{}{}
					}

					return nil
				}); err != nil {
					return nil, err
				}
			} else {
				dirFiles, err := ioutil.ReadDir(location)
				if err != nil {
					return nil, err
				}

				for _, dirFile := range dirFiles {
					if dirFile.Mode().IsRegular() && l.hasCorrectExtension(dirFile) {
						fileList[filepath.Join(location, dirFile.Name())] = struct{}{}
					}
				}
			}
		}
	}

	// convert map into slice
	result := make([]string, 0, len(fileList))
	for k := range fileList {
		result = append(result, k)
	}

	return result, nil
}

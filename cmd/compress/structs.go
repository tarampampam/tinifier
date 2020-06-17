package compress

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type (
	fileExtensions []string
	targets        []string

	task struct {
		num      uint32
		filePath string
	}

	result struct {
		error               error
		fileType            string
		filePath            string
		originalSizeBytes   uint64
		compressedSizeBytes uint64
	}
)

func (e *fileExtensions) GetAll() (all []string) {
	const extensionsDelimiter = ","

	for _, ext := range *e {
		if strings.Contains(ext, extensionsDelimiter) {
			all = append(all, strings.Split(ext, extensionsDelimiter)...)
		} else {
			all = append(all, ext)
		}
	}

	return all
}

func (t *targets) Expand() (files []string) {
	for _, path := range *t {
		// Extract absolute path to the target
		if absPath, err := filepath.Abs(path); err == nil {
			// If file stats extracted successful
			if info, err := os.Stat(absPath); err == nil {
				switch mode := info.Mode(); {
				case mode.IsDir(): // If directory found - run files iterator inside it
					files = append(files, t.scanDir(absPath)...)

				case mode.IsRegular(): // If regular file found - append it into result
					files = append(files, absPath)
				}
			}
		}
	}

	return files
}

func (t *targets) scanDir(path string) (files []string) {
	if dirFiles, err := ioutil.ReadDir(path); err == nil {
		for _, file := range dirFiles {
			if file.Mode().IsRegular() {
				if abs, err := filepath.Abs(filepath.Join(path, file.Name())); err == nil {
					files = append(files, abs)
				}
			}
		}
	}

	return files
}

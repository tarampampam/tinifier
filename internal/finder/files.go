package finder

import (
	"context"
	"fmt"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"strings"
)

// FileFilterFn is a function that filters files.
type FileFilterFn = func(fs.FileInfo) bool

// FilterByExt returns a filter function that filters files by their extensions. If caseSensitive is false, the
// extension will be compared in a case-insensitive manner.
func FilterByExt(caseSensitive bool, exts ...string) FileFilterFn {
	var m = make(map[string]struct{}, len(exts))

	for _, ext := range exts {
		if !caseSensitive {
			ext = strings.ToLower(ext)
		}

		m[ext] = struct{}{}
	}

	return func(info fs.FileInfo) bool {
		if info.IsDir() {
			return false
		}

		if ext := filepath.Ext(info.Name()); ext != "" {
			if !caseSensitive {
				ext = strings.ToLower(ext)
			}

			if _, ok := m[ext[1:]]; ok {
				return true
			}
		}

		return false
	}
}

// Files returns a sequence of absolute paths to files in the specified directory. If recursive is true, it will
// walk the directory recursively. The filter functions are used to filter the files. If any of the filter functions
// return false, the file will be skipped.
func Files(
	ctx context.Context,
	where string,
	recursive bool,
	filter ...FileFilterFn,
) (iter.Seq[string], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if stat, err := os.Stat(where); err != nil {
		return nil, err
	} else if !stat.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", where)
	}

	if recursive {
		return iterateFilesRecursive(ctx, where, filter...), nil
	}

	return iterateFiles(ctx, where, filter...), nil
}

// iterateFiles returns a sequence of absolute paths to files in the specified directory.
func iterateFiles( //nolint:gocognit
	ctx context.Context,
	where string,
	filter ...FileFilterFn,
) iter.Seq[string] {
	return func(yield func(string) bool) {
		f, openErr := os.Open(where)
		if openErr != nil {
			return
		}

		names, readErr := f.Readdirnames(-1)
		if readErr != nil {
			return
		}

		_ = f.Close()

	loop:
		for _, path := range names {
			select {
			case <-ctx.Done():
				return
			default:
			}

			path = filepath.Join(where, path)

			if !filepath.IsAbs(path) { // make path absolute
				if abs, err := filepath.Abs(path); err != nil {
					return
				} else {
					path = abs
				}
			}

			stat, err := os.Stat(path)
			if err != nil || stat.IsDir() {
				continue // skip directories and stat errors
			}

			for _, fn := range filter {
				if !fn(stat) {
					continue loop
				}
			}

			if !yield(path) {
				return
			}
		}
	}
}

// iterateFilesRecursive returns a sequence of absolute paths to files in the specified directory recursively.
func iterateFilesRecursive(
	ctx context.Context,
	where string,
	filter ...FileFilterFn,
) iter.Seq[string] {
	return func(yield func(string) bool) {
		_ = filepath.WalkDir(where, func(path string, d fs.DirEntry, walkErr error) error {
			select {
			case <-ctx.Done():
				return filepath.SkipAll
			default:
			}

			if walkErr != nil || d.IsDir() {
				return nil // skip directories and walking errors
			}

			info, err := d.Info()
			if err != nil {
				return nil
			}

			for _, fn := range filter {
				if !fn(info) {
					return nil
				}
			}

			if !filepath.IsAbs(path) { // make path absolute
				if path, err = filepath.Abs(path); err != nil {
					return nil
				}
			}

			if !yield(path) {
				return filepath.SkipAll
			}

			return nil
		})
	}
}

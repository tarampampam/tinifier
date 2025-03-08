package finder

import (
	"context"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// FileFilterFn is a function type used to filter files.
// It takes a fs.FileInfo object and returns a boolean indicating
// whether the file should be included (true) or filtered out (false).
type FileFilterFn = func(fs.FileInfo) bool

// FilterByExt creates a file filter function that filters files based on their extensions.
// If caseSensitive is false, the extensions will be compared in a case-insensitive manner.
// The function returns a FileFilterFn that checks if a file's extension matches any of the provided exts.
func FilterByExt(caseSensitive bool, exts ...string) FileFilterFn {
	var m = make(map[string]struct{}, len(exts))

	// store the extensions in a map for quick lookup
	for _, ext := range exts {
		if !caseSensitive {
			ext = strings.ToLower(ext)
		}

		m[ext] = struct{}{}
	}

	// return the filter function
	return func(info fs.FileInfo) bool {
		// skip directories
		if info.IsDir() {
			return false
		}

		// extract and normalize the file extension
		if ext := filepath.Ext(info.Name()); ext != "" {
			if !caseSensitive {
				ext = strings.ToLower(ext)
			}

			// remove the leading dot (.) before checking the map
			if _, ok := m[ext[1:]]; ok {
				return true
			}
		}

		return false
	}
}

// Files returns a sequence of absolute paths to files found in the specified paths (`where`).
// If a path in `where` is a directory, it will be scanned for files.
// If `recursive` is true, directories will be searched recursively.
//
// The `filter` functions are applied only to files inside directories (not to the given file paths).
// If any filter function returns false, the file is skipped. If any filesystem error occurs,
// the file or directory is ignored.
//
// Example usage:
//
//	for path := range Files(ctx, []string{"/path/to/dir", "/path/to/file.txt"}, true) {
//	    fmt.Println(path)
//	}
func Files(
	ctx context.Context,
	where []string,
	recursive bool,
	filter ...FileFilterFn,
) iter.Seq[string] {
	var seq = make([]iter.Seq[string], 0, len(where))

	for _, path := range where {
		stat, err := os.Stat(path)
		if err != nil {
			continue // Ignore paths that cannot be accessed
		}

		if stat.IsDir() {
			if recursive {
				seq = append(seq, iterateFilesRecursive(ctx, path, filter...))
			} else {
				seq = append(seq, iterateFiles(ctx, path, filter...))
			}
		} else {
			seq = append(seq, singleFile(ctx, path))
		}
	}

	// Combine all sequences into a single sequence
	return func(yield func(string) bool) {
		for _, s := range seq {
			for path := range s {
				if err := ctx.Err(); err != nil {
					return // stop processing if the context is canceled
				}

				if !yield(path) {
					return // Stop yielding if the receiver stops accepting values
				}
			}
		}
	}
}

// singleFile returns a sequence containing a single file's absolute path.
// This is used when the input path is a single file instead of a directory.
func singleFile(
	ctx context.Context,
	where string,
) iter.Seq[string] {
	return func(yield func(string) bool) {
		// convert to absolute path if needed
		if !filepath.IsAbs(where) {
			var absErr error
			if where, absErr = filepath.Abs(where); absErr != nil {
				return // ignore files that can't be resolved to absolute paths
			}
		}

		if err := ctx.Err(); err != nil {
			return // stop processing if the context is canceled
		}

		// yield the absolute file path
		yield(where)
	}
}

// iterateFiles returns a sequence of absolute file paths inside the specified directory (non-recursively).
// The function applies the provided filter functions to determine which files should be included.
func iterateFiles( //nolint:gocognit
	ctx context.Context,
	where string,
	filter ...FileFilterFn,
) iter.Seq[string] {
	return func(yield func(string) bool) {
		if err := ctx.Err(); err != nil {
			return // stop processing if the context is canceled
		}

		f, openErr := os.Open(where)
		if openErr != nil {
			return // ignore directories that can't be opened
		}

		names, readErr := f.Readdirnames(-1) // read all file names in the directory
		if readErr != nil {
			return
		}

		_ = f.Close() // close the directory after reading

		slices.Sort(names)

	loop:
		for _, path := range names {
			// construct the full file path
			path = filepath.Join(where, path)

			// Convert to absolute path if needed
			if !filepath.IsAbs(path) {
				if abs, err := filepath.Abs(path); err != nil {
					return // ignore files that can't be resolved to absolute paths
				} else {
					path = abs
				}
			}

			stat, statErr := os.Stat(path)
			if statErr != nil || stat.IsDir() {
				continue // skip directories and files with stat errors
			}

			// apply all filter functions
			for _, fn := range filter {
				if !fn(stat) {
					continue loop // skip the file if it doesn't pass the filters
				}
			}

			if err := ctx.Err(); err != nil {
				return // stop processing if the context is canceled
			}

			// yield the file path
			if !yield(path) {
				return // stop yielding if the receiver stops accepting values
			}
		}
	}
}

// iterateFilesRecursive returns a sequence of absolute file paths inside the specified directory recursively.
// The function walks the directory tree and applies the provided filter functions.
func iterateFilesRecursive(
	ctx context.Context,
	where string,
	filter ...FileFilterFn,
) iter.Seq[string] {
	return func(yield func(string) bool) {
		_ = filepath.WalkDir(where, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return nil // skip directories and walking errors
			}

			info, err := d.Info()
			if err != nil {
				return nil // ignore files that can't be accessed
			}

			// apply all filter functions
			for _, fn := range filter {
				if !fn(info) {
					return nil // skip the file if it doesn't pass the filters
				}
			}

			// convert to absolute path if needed
			if !filepath.IsAbs(path) {
				if path, err = filepath.Abs(path); err != nil {
					return nil // ignore files that can't be resolved to absolute paths
				}
			}

			if ctxErr := ctx.Err(); ctxErr != nil {
				return filepath.SkipAll // stop processing if the context is canceled
			}

			// yield the file path
			if !yield(path) {
				return filepath.SkipAll // stop traversal if the receiver stops accepting values
			}

			return nil
		})
	}
}

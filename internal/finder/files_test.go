package finder_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gh.tarampamp.am/tinifier/v5/internal/finder"
)

func TestFiles(t *testing.T) {
	t.Parallel()

	var join = filepath.Join

	for name, tc := range map[string]struct {
		makeDirs      []string
		makeFiles     []string
		givePath      []string
		giveRecursive bool
		giveFilter    []finder.FileFilterFn
		want          []string
	}{
		"common, recursive": {
			makeDirs: []string{
				join("dir1", "dir2", "dir3"),
				join("dir1", "dir4"),
				"dir5",
			},
			makeFiles: []string{
				join("dir1", "file1"),
				join("dir1", "file2"),
				join("dir1", "dir2", "file3"),
				join("dir1", "dir2", "dir3", "file4"),
				join("dir1", "dir4", "file5"),
				join("dir5", "file6"), // <-- should be skipped
			},
			givePath:      []string{"dir1"},
			giveRecursive: true,
			want: []string{
				join("dir1", "dir2", "dir3", "file4"),
				join("dir1", "dir2", "file3"),
				join("dir1", "dir4", "file5"),
				join("dir1", "file1"),
				join("dir1", "file2"),
			},
		},
		"common, non-recursive": {
			makeDirs: []string{
				join("dir1", "dir2", "dir3"),
				join("dir1", "dir4"),
			},
			makeFiles: []string{
				join("dir1", "file1"),
				join("dir1", "file2"),
				join("dir1", "dir2", "file3"),         // <-- should be skipped
				join("dir1", "dir2", "dir3", "file4"), // <-- should be skipped
				join("dir1", "dir4", "file5"),         // <-- should be skipped
			},
			givePath:      []string{"dir1"},
			giveRecursive: false,
			want: []string{
				join("dir1", "file1"),
				join("dir1", "file2"),
			},
		},
		"filtering, non-recursive": {
			makeFiles: []string{
				"file.txt",
				"file.jpg",
				"file.png",
				"foobar",
			},
			giveFilter: []finder.FileFilterFn{
				finder.FilterByExt(false, "TXT", "jpg"),
			},
			givePath: []string{""},
			want: []string{
				"file.jpg",
				"file.txt",
			},
		},
		"filtering, recursive": {
			makeDirs: []string{
				"dir1",
			},
			makeFiles: []string{
				"file.txt",
				join("dir1", "file.jpg"),
				join("dir1", "file.png"),
				join("dir1", "foobar"),
			},
			giveFilter: []finder.FileFilterFn{func(info fs.FileInfo) bool {
				var (
					isTxtFile = strings.HasSuffix(info.Name(), ".txt")
					isJpgFile = strings.HasSuffix(info.Name(), ".jpg")
				)

				return isTxtFile || isJpgFile
			}},
			givePath:      []string{""},
			giveRecursive: true,
			want: []string{
				join("dir1", "file.jpg"),
				"file.txt",
			},
		},
		"single file": {
			makeFiles: []string{
				"file.txt",
			},
			givePath: []string{"file.txt"},
			want:     []string{"file.txt"},
		},
		"complex": {
			makeDirs: []string{
				"dir1",
				"dir2",
				join("dir2", "dir3"),
			},
			makeFiles: []string{
				"file1",
				join("dir1", "file2"),
				join("dir1", "file3.TXT"),
				join("dir2", "file4"),
				join("dir2", "dir3", "file5.txt"),
			},
			givePath: []string{
				"foobar",
				"dir1",
				"dir2",
				"file1",
				join("bar", "baz"),
			},
			giveRecursive: true,
			giveFilter: []finder.FileFilterFn{
				finder.FilterByExt(false, "txt"),
			},
			want: []string{
				join("dir1", "file3.TXT"),
				join("dir2", "dir3", "file5.txt"),
				"file1",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var tmpDir = t.TempDir()

			for _, dir := range tc.makeDirs {
				assertNoError(t, os.MkdirAll(join(tmpDir, dir), 0o755))
			}

			for _, file := range tc.makeFiles {
				assertNoError(t, os.WriteFile(join(tmpDir, file), nil, 0o600))
			}

			givePaths := make([]string, len(tc.givePath))
			for i, path := range tc.givePath {
				givePaths[i] = join(tmpDir, path)
			}

			res := finder.Files(t.Context(), givePaths, tc.giveRecursive, tc.giveFilter...)

			var slice = slices.Collect(res)

			// remove the temporary directory from the paths
			for i, path := range slice {
				slice[i] = strings.TrimPrefix(path, tmpDir+string(filepath.Separator))
			}

			assertSlicesEqual(t, tc.want, slice)
		})
	}
}

func assertSlicesEqual[T comparable](t *testing.T, expected, actual []T) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}

	for i := range expected {
		if expected[i] != actual[i] {
			t.Fatalf("expected %v, got %v", expected, actual)
		}
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

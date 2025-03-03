package finder_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
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
		givePath      string
		giveRecursive bool
		giveFilter    []finder.FileFilterFn
		want          []string
		wantErrSubstr string
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
			givePath:      "dir1",
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
			givePath:      "dir1",
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
				finder.FilterByExt(false, "txt", "jpg"),
			},
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
			giveRecursive: true,
			want: []string{
				join("dir1", "file.jpg"),
				"file.txt",
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

			res, err := finder.Files(t.Context(), join(tmpDir, tc.givePath), tc.giveRecursive, tc.giveFilter...)

			if tc.wantErrSubstr != "" {
				assertNil(t, res)
				assertError(t, err)
				assertErrorContains(t, err, tc.wantErrSubstr)

				return
			}

			var slice = slices.Collect(res)

			// remove the temporary directory from the paths
			for i, path := range slice {
				slice[i] = strings.TrimPrefix(path, tmpDir+string(filepath.Separator))
			}

			assertNoError(t, err)
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

func assertError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func assertErrorContains(t *testing.T, err error, substr ...string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	for _, s := range substr {
		var got = err.Error()

		if !strings.Contains(got, s) {
			t.Fatalf("expected error to contain %q, got %q", s, got)
		}
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertNil(t *testing.T, v any) {
	t.Helper()

	if ref := reflect.ValueOf(v); ref.Kind() == reflect.Ptr && !ref.IsNil() {
		t.Fatalf("expected nil, got %v", v)
	}
}

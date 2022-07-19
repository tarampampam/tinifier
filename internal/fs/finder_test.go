package fs_test

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tarampampam/tinifier/v4/internal/fs"
)

func TestFindFiles(t *testing.T) {
	var cases = map[string]struct {
		name        string
		makeOSDirs  []string
		makeOSFiles []string
		giveWhere   []string
		giveOptions []fs.FinderOption
		wantResult  []string
	}{
		"just files": {
			makeOSFiles: []string{
				"foo",
				"bar.txt",
				"baz.txt",
			},
			giveWhere:   []string{""},
			giveOptions: []fs.FinderOption{fs.WithFilesExt("txt")},
			wantResult: []string{
				"bar.txt",
				"baz.txt",
			},
		},
		"with directories (non-recursive)": {
			makeOSDirs: []string{
				"directory1",
			},
			makeOSFiles: []string{
				"foo.txt",
				"bar.txt",
				"baz",
				filepath.Join("directory1", "baz.txt"),
			},
			giveWhere: []string{
				".",
				"bar.txt", // duplicates must be skipped,
				"baz",
			},
			giveOptions: []fs.FinderOption{fs.WithFilesExt("txt")},
			wantResult: []string{
				"foo.txt",
				"bar.txt",
				"baz",
			},
		},
		"with directories (recursive)": {
			makeOSDirs: []string{
				"directory1",
				filepath.Join("directory1", "inside"),
			},
			makeOSFiles: []string{
				"foo.txt",
				"bar.txt",
				filepath.Join("directory1", "baz.txt"),
				filepath.Join("directory1", "inside", "blah.txt"),
			},
			giveWhere: []string{
				"",
			},
			giveOptions: []fs.FinderOption{fs.WithFilesExt("txt"), fs.WithRecursive(true)},
			wantResult: []string{
				"foo.txt",
				"bar.txt",
				filepath.Join("directory1", "baz.txt"),
				filepath.Join("directory1", "inside", "blah.txt"),
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			tmpDir, tmpDirErr := ioutil.TempDir("", "test-")
			require.NoError(t, tmpDirErr)

			defer func(d string) { require.NoError(t, os.RemoveAll(d)) }(tmpDir)

			for _, d := range tt.makeOSDirs {
				require.NoError(t, os.Mkdir(filepath.Join(tmpDir, d), 0777))
			}

			for _, f := range tt.makeOSFiles {
				file, createErr := os.Create(filepath.Join(tmpDir, f))
				require.NoError(t, createErr)

				_, fileWritingErr := file.Write([]byte{})
				require.NoError(t, fileWritingErr)
				require.NoError(t, file.Close())
			}

			// modify "give" paths and "want" results (append path to the tmp dir at the start)
			for i := 0; i < len(tt.giveWhere); i++ {
				tt.giveWhere[i] = filepath.Join(tmpDir, tt.giveWhere[i])
			}
			for i := 0; i < len(tt.wantResult); i++ {
				tt.wantResult[i] = filepath.Join(tmpDir, tt.wantResult[i])
			}

			var result []string

			assert.NoError(t, fs.FindFiles(context.Background(), tt.giveWhere, func(path string) {
				result = append(result, path)
			}, tt.giveOptions...))

			assert.ElementsMatch(t, tt.wantResult, result)
		})
	}
}

func TestFindFiles_CancelledContext(t *testing.T) {
	var ctx, cancel = context.WithCancel(context.Background())

	cancel() // <-- important

	assert.ErrorIs(t, fs.FindFiles(ctx, []string{"."}, func(path string) {}), context.Canceled)
}

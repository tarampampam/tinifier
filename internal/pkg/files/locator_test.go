package files

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLocator(t *testing.T) {
	_, err := NewLocator([]string{}, []string{"foo"})
	assert.Error(t, err)

	_, err = NewLocator([]string{"foo"}, []string{})
	assert.Error(t, err)
}

func TestLocator_Find(t *testing.T) {
	var cases = []struct {
		name           string
		makeOSDirs     []string
		makeOSFiles    []string
		giveLocations  []string
		giveExtensions []string
		giveRecursive  bool
		wantResult     []string
	}{
		{
			name: "just files",
			makeOSFiles: []string{
				"foo",
				"bar.txt",
				"baz.txt",
			},
			giveLocations:  []string{""},
			giveExtensions: []string{"txt"},
			giveRecursive:  false,
			wantResult: []string{
				"bar.txt",
				"baz.txt",
			},
		},
		{
			name: "with directories (non-recursive)",
			makeOSDirs: []string{
				"directory1",
			},
			makeOSFiles: []string{
				"foo.txt",
				"bar.txt",
				filepath.Join("directory1", "baz.txt"),
			},
			giveLocations: []string{
				"",
				"bar.txt", // duplicates must be skipped,
			},
			giveExtensions: []string{"txt"},
			giveRecursive:  false,
			wantResult: []string{
				"foo.txt",
				"bar.txt",
			},
		},
		{
			name: "with directories (recursive)",
			makeOSDirs: []string{
				"directory1",
			},
			makeOSFiles: []string{
				"foo.txt",
				"bar.txt",
				filepath.Join("directory1", "baz.txt"),
			},
			giveLocations: []string{
				"",
				filepath.Join("directory1", "baz.txt"), // duplicates must be skipped
			},
			giveExtensions: []string{"txt"},
			giveRecursive:  true, // important
			wantResult: []string{
				"foo.txt",
				"bar.txt",
				filepath.Join("directory1", "baz.txt"),
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, tmpDirErr := ioutil.TempDir("", "test-")
			assert.NoError(t, tmpDirErr)

			defer func(d string) { assert.NoError(t, os.RemoveAll(d)) }(tmpDir)

			for _, d := range tt.makeOSDirs {
				assert.NoError(t, os.Mkdir(filepath.Join(tmpDir, d), 0777))
			}

			for _, f := range tt.makeOSFiles {
				file, createErr := os.Create(filepath.Join(tmpDir, f))
				assert.NoError(t, createErr)
				_, fileWritingErr := file.Write([]byte{})
				assert.NoError(t, fileWritingErr)
				assert.NoError(t, file.Close())
			}

			// modify "give" paths and "want" results (append path to the tmp dir at the start)
			for i := 0; i < len(tt.giveLocations); i++ {
				tt.giveLocations[i] = filepath.Join(tmpDir, tt.giveLocations[i])
			}
			for i := 0; i < len(tt.wantResult); i++ {
				tt.wantResult[i] = filepath.Join(tmpDir, tt.wantResult[i])
			}

			locator, err := NewLocator(tt.giveLocations, tt.giveExtensions)
			assert.NoError(t, err)

			result, err := locator.Find(tt.giveRecursive)
			assert.NoError(t, err)

			assert.ElementsMatch(t, tt.wantResult, result)
		})
	}
}

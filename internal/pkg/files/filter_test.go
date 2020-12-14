package files

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterMissed(t *testing.T) {
	var cases = []struct {
		name        string
		makeOSDirs  []string
		makeOSFiles []string
		givePaths   []string
		wantResult  []string
	}{
		{
			name: "just files",

			makeOSFiles: []string{
				"foo",
				"bar.txt",
			},
			givePaths: []string{
				"1",
				"foo",
				"bar.txt",
				filepath.Join("sfsdfs", "sdfsdf"),
			},
			wantResult: []string{
				"foo",
				"bar.txt",
			},
		},
		{
			name: "with directories",

			makeOSDirs: []string{
				"foo",
				"bar",
				filepath.Join("bar", "one"),
			},
			makeOSFiles: []string{
				"root_file",
				filepath.Join("foo", "foo-file"),
				filepath.Join("bar", "one", "bar-one-file"),
			},
			givePaths: []string{
				"root_file",
				"fake-file",
				"foo",
				filepath.Join("foo", "foo-file"),
				filepath.Join("bar", "one", "bar-one-file"),
				filepath.Join("bar", "sdfsdfsdfsdf"),
			},
			wantResult: []string{
				"root_file",
				"foo",
				filepath.Join("foo", "foo-file"),
				filepath.Join("bar", "one", "bar-one-file"),
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
			for i := 0; i < len(tt.givePaths); i++ {
				tt.givePaths[i] = filepath.Join(tmpDir, tt.givePaths[i])
			}
			for i := 0; i < len(tt.wantResult); i++ {
				tt.wantResult[i] = filepath.Join(tmpDir, tt.wantResult[i])
			}

			assert.ElementsMatch(t, tt.wantResult, FilterMissed(tt.givePaths))
		})
	}
}

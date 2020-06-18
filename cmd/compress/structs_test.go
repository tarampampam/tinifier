package compress

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_fileExtensions_GetAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		extensions fileExtensions
		wantAll    []string
	}{
		{
			name:       "basic",
			extensions: fileExtensions{"foo,bar", "baz", "blah;boo"},
			wantAll:    []string{"foo", "bar", "baz", "blah;boo"},
		},
		{
			name:       "empty",
			extensions: fileExtensions{""},
			wantAll:    []string{""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantAll, tt.extensions.GetAll())
		})
	}
}

func Test_targets_Expand(t *testing.T) {
	t.Parallel()

	// Create directory in temporary
	tmpDir, err := ioutil.TempDir("", "test-")
	if err != nil {
		panic(err)
	}

	defer func(d string) { assert.NoError(t, os.RemoveAll(d)) }(tmpDir)

	// create dirs
	for _, d := range []string{"foo", "bar"} {
		assert.NoError(t, os.Mkdir(filepath.Join(tmpDir, d), 0777))
	}

	// create files
	for _, f := range []string{"1", "2", filepath.Join("foo", "3"), filepath.Join("bar", "4")} {
		if f, err := os.Create(filepath.Join(tmpDir, f)); err == nil {
			assert.Nil(t, f.Close())
		} else {
			panic(err)
		}
	}

	tests := []struct {
		name        string
		giveTargets *targets
		wantFiles   []string
	}{
		{
			name:        "root and one dir",
			giveTargets: &targets{tmpDir, filepath.Join(tmpDir, "bar")},
			wantFiles: []string{
				filepath.Join(tmpDir, "1"),
				filepath.Join(tmpDir, "2"),
				filepath.Join(tmpDir, "bar", "4"),
			},
		},
		{
			name:        "only root",
			giveTargets: &targets{tmpDir},
			wantFiles: []string{
				filepath.Join(tmpDir, "1"),
				filepath.Join(tmpDir, "2"),
			},
		},
		{
			name:        "only dir",
			giveTargets: &targets{filepath.Join(tmpDir, "foo")},
			wantFiles: []string{
				filepath.Join(tmpDir, "foo", "3"),
			},
		},
		{
			name: "only files",
			giveTargets: &targets{
				filepath.Join(tmpDir, "foo", "3"),
				filepath.Join(tmpDir, "bar", "4"),
			},
			wantFiles: []string{
				filepath.Join(tmpDir, "foo", "3"),
				filepath.Join(tmpDir, "bar", "4"),
			},
		},
		{
			name: "missing should be ignored",
			giveTargets: &targets{
				filepath.Join(tmpDir, "foo", "3"),
				filepath.Join(tmpDir, "blah"),
			},
			wantFiles: []string{
				filepath.Join(tmpDir, "foo", "3"),
			},
		},
		{
			name: "All is missing",
			giveTargets: &targets{
				filepath.Join(tmpDir, "blah"),
				"foo",
			},
			wantFiles: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantFiles, tt.giveTargets.Expand())
		})
	}
}

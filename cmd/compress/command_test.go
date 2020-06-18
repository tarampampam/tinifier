package compress

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"tinifier/tinypng"

	"bou.ke/monkey"
	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
)

func Test_Execute(t *testing.T) {
	// Create directory in temporary
	tmpDir, err := ioutil.TempDir("", "test-")
	if err != nil {
		panic(err)
	}

	defer func(d string) { assert.Nil(t, os.RemoveAll(d)) }(tmpDir)

	// create dirs
	for _, d := range []string{"foo", "bar"} {
		assert.NoError(t, os.Mkdir(filepath.Join(tmpDir, d), 0777))
	}

	// create files
	for _, f := range []string{"1.a", "2.txt", filepath.Join("foo", "3.A"), filepath.Join("bar", "4.txt")} {
		if f, err := os.Create(filepath.Join(tmpDir, f)); err == nil {
			assert.Nil(t, f.Close())
		} else {
			panic(err)
		}
	}

	// patch client method
	var where *tinypng.Client
	guard := monkey.PatchInstanceMethod(reflect.TypeOf(where), "Compress",
		func(_ *tinypng.Client, ctx context.Context, body io.Reader) (*tinypng.Result, error) {
			return &tinypng.Result{
				Input: tinypng.Input{
					Size: 100,
					Type: "image/png",
				},
				Output: tinypng.Output{
					Size:   90,
					Type:   "image/png",
					Width:  50,
					Height: 50,
					Ratio:  0.123,
					URL:    "https://foo.com/bar",
				},
				Error:            nil,
				Message:          nil,
				CompressionCount: 123,
				Compressed:       ioutil.NopCloser(bytes.NewBufferString("compressed ok")),
			}, nil
		},
	)

	defer guard.Restore()

	// create and configure command
	cmd := &Command{}
	cmd.FileExtensions = []string{"a", "A"}
	cmd.Threads = 2
	cmd.Targets.Path = []string{tmpDir, filepath.Join(tmpDir, "foo"), filepath.Join(tmpDir, "bar")}

	capturer.CaptureOutput(func() {
		assert.Nil(t, cmd.Execute([]string{}))
	})

	for _, filePath := range []string{filepath.Join(tmpDir, "1.a"), filepath.Join(tmpDir, "foo", "3.A")} {
		content, _ := ioutil.ReadFile(filePath)
		assert.Equal(t, "compressed ok", string(content))
	}
}

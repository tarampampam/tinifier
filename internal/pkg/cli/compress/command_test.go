package compress

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tarampampam/tinifier/pkg/tinypng"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func copyTestdata(t *testing.T, destination string) error {
	t.Helper()

	source, err := filepath.Abs("./testdata")
	if err != nil {
		return err
	}

	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		var relPath = strings.Replace(path, source, "", 1)

		if relPath == "" {
			return nil
		}

		if info.IsDir() {
			return os.Mkdir(filepath.Join(destination, relPath), 0755)
		}

		data, readErr := ioutil.ReadFile(filepath.Join(source, relPath))
		if readErr != nil {
			return readErr
		}

		return ioutil.WriteFile(filepath.Join(destination, relPath), data, info.Mode())
	})
}

func Test_CommandSuccessfulRunning(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "test-") // create temporary directory
	assert.NoError(t, err)

	assert.NoError(t, copyTestdata(t, tmpDir)) // copy testdata into temporary directory

	defer func(d string) { assert.NoError(t, os.RemoveAll(d)) }(tmpDir) // remove temp dir after all

	logger := zap.NewNop()

	var (
		where *tinypng.Client
		what  = "Compress"
		guard *monkey.PatchGuard
	)

	guard = monkey.PatchInstanceMethod(
		reflect.TypeOf(where), what, func(_ *tinypng.Client, ctx context.Context, body io.Reader) (*tinypng.Result, error) {
			defer guard.Restore()

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

	cmd := NewCommand(logger)
	cmd.SetArgs([]string{
		"--api-key", strings.Repeat("x", 32),
		tmpDir,
	})

	assert.NoError(t, cmd.Execute()) // TODO write asserts
}

func TestCommand(t *testing.T) {
	t.Skip("Write better tests")
}

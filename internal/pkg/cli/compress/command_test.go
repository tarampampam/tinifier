package compress

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tarampampam/tinifier/pkg/tinypng"

	"bou.ke/monkey"
	"github.com/kami-zh/go-capturer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
		} else {
			data, err := ioutil.ReadFile(filepath.Join(source, relPath))
			if err != nil {
				return err
			}

			return ioutil.WriteFile(filepath.Join(destination, relPath), data, 0777)
		}
	})
}

func Test_CommandSuccessfulRunning(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "test-") // create temporary directory
	assert.NoError(t, err)

	assert.NoError(t, copyTestdata(t, tmpDir)) // copy testdata into temporary directory

	defer func(d string) { assert.Nil(t, os.RemoveAll(d)) }(tmpDir) // remove temp dir after all

	logger := logrus.New()
	//logger.SetOutput(ioutil.Discard)

	var (
		where *tinypng.Client
		what  = "Compress"
		guard *monkey.PatchGuard
	)

	guard = monkey.PatchInstanceMethod(
		reflect.TypeOf(where), what, func(_ *tinypng.Client, ctx context.Context, body io.Reader) (*tinypng.Result, error) {
			defer guard.Restore()

			return nil, errors.New("foo error")
		},
	)

	cmd := NewCommand(logger)
	cmd.SetArgs([]string{
		"--api-key", strings.Repeat("x", 32),
		tmpDir,
	})

	output := capturer.CaptureStdout(func() {
		assert.NoError(t, cmd.Execute())
	})

	t.Log(output)
}

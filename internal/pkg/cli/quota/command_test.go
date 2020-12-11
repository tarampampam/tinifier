package quota

import (
	"context"
	"errors"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tarampampam/tinifier/pkg/tinypng"

	"bou.ke/monkey"
	"github.com/kami-zh/go-capturer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func Test_CommandSuccessfulRunning(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	var (
		where *tinypng.Client
		what  = "GetCompressionCount"
		guard *monkey.PatchGuard
	)

	guard = monkey.PatchInstanceMethod(
		reflect.TypeOf(where), what, func(_ *tinypng.Client, ctx context.Context) (uint64, error) {
			defer guard.Restore()

			return 1234321, nil
		},
	)

	cmd := NewCommand(logger)
	cmd.SetArgs([]string{"--api-key", strings.Repeat("x", 32)})

	output := capturer.CaptureStdout(func() {
		assert.NoError(t, cmd.Execute())
	})

	assert.Contains(t, output, "quota is:")
	assert.Contains(t, output, strconv.FormatUint(1234321, 10))
}

func Test_CommandErroredRunning(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	var (
		where *tinypng.Client
		what  = "GetCompressionCount"
		guard *monkey.PatchGuard
	)

	guard = monkey.PatchInstanceMethod(
		reflect.TypeOf(where), what, func(_ *tinypng.Client, ctx context.Context) (uint64, error) {
			defer guard.Restore()

			time.Sleep(time.Microsecond * 100)

			return 0, errors.New("fake error")
		},
	)

	cmd := NewCommand(logger)
	cmd.SetArgs([]string{"--api-key", strings.Repeat("x", 32)})

	output := capturer.CaptureStderr(func() {
		assert.Error(t, cmd.Execute())
	})

	assert.Contains(t, output, "fake error")
}

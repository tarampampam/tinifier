package quota

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/tarampampam/tinifier/v4/pkg/tinypng"

	"bou.ke/monkey"
	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
)

func Test_CommandSuccessfulRunning(t *testing.T) {
	var (
		where *tinypng.Client
		what  = "CompressionCount"
		guard *monkey.PatchGuard
	)

	guard = monkey.PatchInstanceMethod(
		reflect.TypeOf(where), what, func(_ *tinypng.Client, timeout ...time.Duration) (uint64, error) {
			defer guard.Restore()

			return 1234321, nil
		},
	)

	cmd := NewCommand(zap.NewNop())
	cmd.SetArgs([]string{"--api-key", strings.Repeat("x", 32)})

	output := capturer.CaptureStdout(func() {
		assert.NoError(t, cmd.Execute())
	})

	assert.Contains(t, output, "quota is:")
	assert.Contains(t, output, "1234321")
}

func Test_CommandRunningWithoutApiKey(t *testing.T) {
	cmd := NewCommand(zap.NewNop())
	cmd.SetArgs([]string{"--api-key", ""})

	output := capturer.CaptureStderr(func() {
		assert.Error(t, cmd.Execute())
	})

	assert.Contains(t, output, "API key was not provided")
}

func Test_CommandRunningWithWrongApiKey(t *testing.T) {
	cmd := NewCommand(zap.NewNop())
	cmd.SetArgs([]string{"--api-key", "xxx"})

	output := capturer.CaptureStderr(func() {
		assert.Error(t, cmd.Execute())
	})

	assert.Contains(t, output, "API key")
	assert.Contains(t, output, "is too short")
}

func Test_CommandRunningWithAPIKeyInEnvironment(t *testing.T) {
	var (
		where *tinypng.Client
		what  = "CompressionCount"
		guard *monkey.PatchGuard
	)

	guard = monkey.PatchInstanceMethod(
		reflect.TypeOf(where), what, func(_ *tinypng.Client, timeout ...time.Duration) (uint64, error) {
			defer guard.Restore()

			return 32123, nil
		},
	)

	assert.NoError(t, os.Setenv("TINYPNG_API_KEY", strings.Repeat("x", 32)))
	defer os.Unsetenv("TINYPNG_API_KEY")

	cmd := NewCommand(zap.NewNop())
	cmd.SetArgs([]string{})

	output := capturer.CaptureStdout(func() {
		assert.NoError(t, cmd.Execute())
	})

	assert.Contains(t, output, "quota is:")
	assert.Contains(t, output, "32123")
}

func Test_CommandErroredRunning(t *testing.T) {
	var (
		where *tinypng.Client
		what  = "CompressionCount"
		guard *monkey.PatchGuard
	)

	guard = monkey.PatchInstanceMethod(
		reflect.TypeOf(where), what, func(_ *tinypng.Client, timeout ...time.Duration) (uint64, error) {
			defer guard.Restore()

			time.Sleep(time.Microsecond * 100)

			return 0, errors.New("fake error")
		},
	)

	cmd := NewCommand(zap.NewNop())
	cmd.SetArgs([]string{"--api-key", strings.Repeat("x", 32)})

	output := capturer.CaptureStderr(func() {
		assert.Error(t, cmd.Execute())
	})

	assert.Contains(t, output, "fake error")
}

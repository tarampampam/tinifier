package quota

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/tarampampam/tinifier/pkg/tinypng"

	"bou.ke/monkey"
	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
)

func Test_CommandSuccessfulRunning(t *testing.T) {
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

	cmd := NewCommand(zap.NewNop())
	cmd.SetArgs([]string{"--api-key", strings.Repeat("x", 32)})

	output := capturer.CaptureStdout(func() {
		assert.NoError(t, cmd.Execute())
	})

	assert.Contains(t, output, "quota is:")
	assert.Contains(t, output, "1234321")
}

func Test_CommandErroredRunning(t *testing.T) {
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

	cmd := NewCommand(zap.NewNop())
	cmd.SetArgs([]string{"--api-key", strings.Repeat("x", 32)})

	output := capturer.CaptureStderr(func() {
		assert.Error(t, cmd.Execute())
	})

	assert.Contains(t, output, "fake error")
}

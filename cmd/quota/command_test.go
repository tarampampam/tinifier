package quota

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"tinifier/tinypng"

	"bou.ke/monkey"
	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
)

func Test_Execute(t *testing.T) {
	var (
		where *tinypng.Client
		what  = "GetCompressionCount"
	)

	guard := monkey.PatchInstanceMethod(
		reflect.TypeOf(where), what, func(_ *tinypng.Client, ctx context.Context) (uint64, error) {
			return 1234321, nil
		},
	)

	defer guard.Restore()

	cmd := &Command{}
	cmd.APIKey = "aaaa000bbbb"

	stdout := capturer.CaptureStdout(func() {
		assert.Nil(t, cmd.Execute([]string{}))
	})

	assert.Contains(t, stdout, "aaaa***bbbb")
	assert.Contains(t, stdout, "1234321")
}

func Test_ExecuteWithError(t *testing.T) {
	var (
		where *tinypng.Client
		what  = "GetCompressionCount"
	)

	guard := monkey.PatchInstanceMethod(
		reflect.TypeOf(where), what, func(_ *tinypng.Client, ctx context.Context) (uint64, error) {
			return 0, errors.New("fake error")
		},
	)

	defer guard.Restore()

	cmd := &Command{}
	cmd.APIKey = "aaaa000bbbb"

	output := capturer.CaptureOutput(func() {
		assert.Error(t, cmd.Execute([]string{}), "fake error")
	})

	assert.Empty(t, output)
}

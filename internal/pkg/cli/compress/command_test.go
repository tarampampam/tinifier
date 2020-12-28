package compress

import (
	"testing"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func Test_CommandRunningWithoutApiKey(t *testing.T) {
	cmd := NewCommand(zap.NewNop())
	cmd.SetArgs([]string{"--api-key", "", "/tmp"})

	output := capturer.CaptureStderr(func() {
		assert.Error(t, cmd.Execute())
	})

	assert.Contains(t, output, "API key was not provided")
}

func Test_CommandRunningWithWrongApiKey(t *testing.T) {
	cmd := NewCommand(zap.NewNop())
	cmd.SetArgs([]string{"--api-key", "xxx", "/tmp"})

	output := capturer.CaptureStderr(func() {
		assert.Error(t, cmd.Execute())
	})

	assert.Contains(t, output, "API key")
	assert.Contains(t, output, "is too short")
}

func TestCommandRunning(t *testing.T) {
	t.Skip("implement me")
}

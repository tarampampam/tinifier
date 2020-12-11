package main

import (
	"os"
	"testing"

	"bou.ke/monkey"
	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
)

func Test_Main(t *testing.T) {
	origFlags := make([]string, 0)
	origFlags = append(origFlags, os.Args...)

	defer func() { os.Args = origFlags }()

	os.Args = []string{"", "--help"}

	output := capturer.CaptureStdout(func() {
		main()
	})

	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "Available Commands:")
	assert.Contains(t, output, "Flags:")
}

func Test_MainWrongCommand(t *testing.T) {
	origFlags := make([]string, 0)
	origFlags = append(origFlags, os.Args...)

	defer func() { os.Args = origFlags }()

	var (
		osExitGuard *monkey.PatchGuard
		exitCode    int = 666
	)

	osExitGuard = monkey.Patch(os.Exit, func(code int) {
		osExitGuard.Unpatch()
		defer osExitGuard.Restore()

		exitCode = code
	})

	os.Args = []string{"", "foo bar"}

	output := capturer.CaptureStderr(func() {
		main()
	})

	assert.Contains(t, output, "unknown command")
	assert.Contains(t, output, "foo bar")
	assert.Equal(t, 1, exitCode)
}

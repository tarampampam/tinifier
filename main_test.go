package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Main(t *testing.T) {
	captureOutput := func(f func()) string {
		t.Helper()

		r, w, err := os.Pipe()
		if err != nil {
			panic(err)
		}

		stdout := os.Stdout
		os.Stdout = w

		defer func() { os.Stdout = stdout }()

		f()

		_ = w.Close()

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)

		return buf.String()
	}

	origFlags := make([]string, 0)
	origFlags = append(origFlags, os.Args...)

	defer func() { os.Args = origFlags }()

	os.Args = []string{"", "-h"}

	output := captureOutput(func() {
		main()
	})

	assert.Contains(t, output, "Help Options")
	assert.Contains(t, output, "Application Options")
	assert.Contains(t, output, "Available commands")
}

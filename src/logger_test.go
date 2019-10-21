package main

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	"log"
	"testing"
)

type FakeWriter struct {
	buf []byte
}

func (w *FakeWriter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)
	return n, err
}

func (w *FakeWriter) CleanBuf() {
	w.buf = w.buf[:0]
}

func (w *FakeWriter) ToStringAndClean() string {
	s := string(w.buf)
	w.CleanBuf()

	return s
}

func TestLoggerVerbose(t *testing.T) {
	t.Parallel()

	var (
		std = log.New(&FakeWriter{}, "", 0)
		err = log.New(&FakeWriter{}, "", 0)
	)

	var cases = []struct {
		logger      Logger
		value       interface{}
		expectedStd interface{}
		expectedErr interface{}
	}{
		{
			logger:      NewLogger(std, err, true, false),
			value:       "foo",
			expectedStd: "foo\n",
			expectedErr: "",
		},
		{
			logger:      NewLogger(std, err, true, true),
			value:       "foo bar Baz\n 123",
			expectedStd: "foo bar Baz\n 123\n",
			expectedErr: "",
		},
		{
			logger:      NewLogger(std, err, true, false),
			value:       []string{"1", "foo", "2"},
			expectedStd: "[1 foo 2]\n",
			expectedErr: "",
		},
		{
			logger:      NewLogger(std, err, false, true),
			value:       "foo bar Baz\n 123",
			expectedStd: "",
			expectedErr: "",
		},
	}

	for _, testCase := range cases {
		testCase.logger.Verbose(testCase.value)

		stdWriter := testCase.logger.StdLogger.Writer().(*FakeWriter)
		errWriter := testCase.logger.ErrLogger.Writer().(*FakeWriter)

		if stdOut := stdWriter.ToStringAndClean(); stdOut != testCase.expectedStd {
			t.Errorf("Exprected value %s for stdOut, got %s", testCase.expectedStd, stdOut)
		}

		if errOut := errWriter.ToStringAndClean(); errOut != testCase.expectedErr {
			t.Errorf("Exprected value %s for stdErr, got %s", testCase.expectedErr, errOut)
		}
	}
}

func TestLoggerInfo(t *testing.T) {
	t.Parallel()

	var (
		std = log.New(&FakeWriter{}, "", 0)
		err = log.New(&FakeWriter{}, "", 0)
	)

	var cases = []struct {
		logger      Logger
		value       interface{}
		expectedStd interface{}
		expectedErr interface{}
	}{
		{
			logger:      NewLogger(std, err, true, false),
			value:       "foo",
			expectedStd: "foo\n",
			expectedErr: "",
		},
		{
			logger:      NewLogger(std, err, true, true),
			value:       "foo bar Baz\n 123",
			expectedStd: "foo bar Baz\n 123\n",
			expectedErr: "",
		},
		{
			logger:      NewLogger(std, err, true, false),
			value:       []string{"1", "foo", "2"},
			expectedStd: "[1 foo 2]\n",
			expectedErr: "",
		},
		{
			logger:      NewLogger(std, err, false, true),
			value:       "foo bar Baz\n 123",
			expectedStd: "foo bar Baz\n 123\n",
			expectedErr: "",
		},
	}

	for _, testCase := range cases {
		testCase.logger.Info(testCase.value)

		stdWriter := testCase.logger.StdLogger.Writer().(*FakeWriter)
		errWriter := testCase.logger.ErrLogger.Writer().(*FakeWriter)

		if stdOut := stdWriter.ToStringAndClean(); stdOut != testCase.expectedStd {
			t.Errorf("Exprected value %s for stdOut, got %s", testCase.expectedStd, stdOut)
		}

		if errOut := errWriter.ToStringAndClean(); errOut != testCase.expectedErr {
			t.Errorf("Exprected value %s for stdErr, got %s", testCase.expectedErr, errOut)
		}
	}
}

func TestLoggerError(t *testing.T) {
	t.Parallel()

	var (
		std   = log.New(&FakeWriter{}, "", 0)
		err   = log.New(&FakeWriter{}, "", 0)
		flags = aurora.BrightFg | aurora.RedFg | aurora.BoldFm
	)

	var cases = []struct {
		logger      Logger
		value       interface{}
		expectedStd interface{}
		expectedErr interface{}
	}{
		{
			logger:      NewLogger(std, err, true, false),
			value:       "foo",
			expectedStd: "",
			expectedErr: "foo\n",
		},
		{
			logger:      NewLogger(std, err, true, true),
			value:       "foo bar Baz\n 123",
			expectedStd: "",
			expectedErr: aurora.Colorize("foo bar Baz\n 123", flags).String() + "\n",
		},
		{
			logger:      NewLogger(std, err, true, false),
			value:       []string{"1", "foo", "2"},
			expectedStd: "",
			expectedErr: "[1 foo 2]\n",
		},
		{
			logger:      NewLogger(std, err, false, true),
			value:       []string{"foo bar Baz\n 123", "blah"},
			expectedStd: "",
			expectedErr: aurora.Colorize("[foo bar Baz\n 123 blah]", flags).String() + "\n",
		},
	}

	for _, testCase := range cases {
		testCase.logger.Error(testCase.value)

		stdWriter := testCase.logger.StdLogger.Writer().(*FakeWriter)
		errWriter := testCase.logger.ErrLogger.Writer().(*FakeWriter)

		if stdOut := stdWriter.ToStringAndClean(); stdOut != testCase.expectedStd {
			t.Errorf("Exprected value %+v for stdOut, got %+v", testCase.expectedStd, stdOut)
		}

		if errOut := errWriter.ToStringAndClean(); errOut != testCase.expectedErr {
			fmt.Printf("+++ %v", errOut)
			t.Errorf("Exprected value %+v for stdErr, got %+v", testCase.expectedErr, errOut)
		}
	}
}

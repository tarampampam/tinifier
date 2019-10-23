package main

import (
	"fmt"
	"log"
	"strings"
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
		c   = NewAnsiColors()
	)

	var cases = []struct {
		logger      Logger
		value       interface{}
		expectedStd interface{}
		expectedErr interface{}
	}{
		{
			logger:      *NewLogger(c, std, err, true, false),
			value:       "foo",
			expectedStd: "foo\n",
			expectedErr: "",
		},
		{
			logger:      *NewLogger(c, std, err, true, true),
			value:       "foo bar Baz\n 123",
			expectedStd: "foo bar Baz\n 123\n",
			expectedErr: "",
		},
		{
			logger:      *NewLogger(c, std, err, true, false),
			value:       []string{"1", "foo", "2"},
			expectedStd: "[1 foo 2]\n",
			expectedErr: "",
		},
		{
			logger:      *NewLogger(c, std, err, false, true),
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
		c   = NewAnsiColors()
	)

	var cases = []struct {
		logger      Logger
		value       interface{}
		expectedStd interface{}
		expectedErr interface{}
	}{
		{
			logger:      *NewLogger(c, std, err, true, false),
			value:       "foo",
			expectedStd: "foo\n",
			expectedErr: "",
		},
		{
			logger:      *NewLogger(c, std, err, true, true),
			value:       "foo bar Baz\n 123",
			expectedStd: "foo bar Baz\n 123\n",
			expectedErr: "",
		},
		{
			logger:      *NewLogger(c, std, err, true, false),
			value:       []string{"1", "foo", "2"},
			expectedStd: "[1 foo 2]\n",
			expectedErr: "",
		},
		{
			logger:      *NewLogger(c, std, err, false, true),
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
		flags = []AnsiColor{AnsiBrightFg, AnsiRedFg, AnsiBoldFm}
		c     = NewAnsiColors()
	)

	var cases = []struct {
		logger      Logger
		value       interface{}
		expectedStd interface{}
		expectedErr interface{}
	}{
		{
			logger:      *NewLogger(c, std, err, true, false),
			value:       "foo",
			expectedStd: "",
			expectedErr: "foo\n",
		},
		{
			logger:      *NewLogger(c, std, err, true, true),
			value:       "foo bar Baz\n 123",
			expectedStd: "",
			expectedErr: NewAnsiColors().Colorize("foo bar Baz\n 123", flags...).(fmt.Stringer).String() + "\n",
		},
		{
			logger:      *NewLogger(c, std, err, true, false),
			value:       []string{"1", "foo", "2"},
			expectedStd: "",
			expectedErr: "[1 foo 2]\n",
		},
		{
			logger:      *NewLogger(c, std, err, false, true),
			value:       []string{"foo bar Baz\n 123", "blah"},
			expectedStd: "",
			expectedErr: NewAnsiColors().Colorize("[foo bar Baz\n 123 blah]", flags...).(fmt.Stringer).String() + "\n",
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

func TestPanic(t *testing.T) {
	t.Parallel()

	var exitCalled, panicCalled bool
	var panicValue interface{}

	logger := *NewLogger(
		NewAnsiColors(),
		log.New(&FakeWriter{}, "", 0),
		log.New(&FakeWriter{}, "", 0),
		true,
		false,
	)

	logger.SetOnExitFunc(func(code int) {
		exitCalled = true
	})

	logger.SetOnPanicFunc(func(v interface{}) {
		panicCalled = true
		panicValue = v
	})

	logger.Panic("foo")

	if exitCalled {
		t.Error("Exit callable called, but shouldn't")
	}

	if !panicCalled {
		t.Error("Panic callable NOT called, but should")
	}

	if fmt.Sprintf("%v", panicValue) != "foo" {
		t.Error("Wrong value passed to the onPanic function")
	}
}

func TestFatal(t *testing.T) {
	t.Parallel()

	var (
		std = log.New(&FakeWriter{}, "", 0)
		err = log.New(&FakeWriter{}, "", 0)
		c   = NewAnsiColors()
	)

	var cases = []struct {
		logger      Logger
		message     interface{}
		exitCalled  bool
		panicCalled bool
		exitCode    int
	}{
		{
			logger:  *NewLogger(c, std, err, true, true),
			message: "foo",
		},
		{
			logger:  *NewLogger(c, std, err, true, false),
			message: "foo bar Baz\n 123",
		},
		{
			logger:  *NewLogger(c, std, err, false, true),
			message: []string{"1", "foo", "2"},
		},
		{
			logger:  *NewLogger(c, std, err, false, false),
			message: []string{"foo bar Baz\n 123", "blah"},
		},
	}

	for _, testCase := range cases {
		testCase.logger.SetOnExitFunc(func(code int) {
			testCase.exitCalled = true
			testCase.exitCode = code
		})

		testCase.logger.SetOnPanicFunc(func(v interface{}) {
			testCase.panicCalled = true
		})

		testCase.logger.Fatal(testCase.message)

		if !testCase.exitCalled {
			t.Error("Exit callable NOT called, but should")
		}

		if testCase.panicCalled {
			t.Error("Panic callable called, but shouldn't")
		}

		if testCase.exitCode != 1 {
			t.Errorf("Wrong exit code. Expected 1, got %d", testCase.exitCode)
		}

		if testCase.logger.StdLogger.Writer().(*FakeWriter).ToStringAndClean() != "" {
			t.Error("stdOut should be empty")
		}

		content := testCase.logger.ErrLogger.Writer().(*FakeWriter).ToStringAndClean()

		if !strings.Contains(content, fmt.Sprintf("%v", testCase.message)) {
			t.Errorf("stdErr [%s] should contains passed message [%s]", content, "foo")
		}

		if !strings.Contains(content, "[Fatal Error]") {
			t.Errorf("stdErr [%s] should contains [%s]", content, "Fatal Error")
		}
	}
}

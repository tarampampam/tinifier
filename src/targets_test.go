package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestFilterFilesUsingExtensions(t *testing.T) {
	type testCase struct {
		values     []string
		extensions []string
		expected   []string
	}

	var asserts = []testCase{
		{[]string{"foo.bar"}, []string{"bar"}, []string{"foo.bar"}},
		{[]string{"foo.bar", "foo.baz", "baz.bar", "foo.foo"}, []string{"baz", "foo"}, []string{"foo.baz", "foo.foo"}},
		{[]string{"foo.bar", "foo.baz", "baz.bar", "foo.foo"}, []string{"baz,foo"}, []string{"foo.baz", "foo.foo"}},
		{[]string{"aa", "ab", "ac", "d"}, []string{"a,b", "d"}, []string{"aa", "ab", "d"}},
		{[]string{"aa", "ab", "ac"}, []string{"d"}, []string{}},
	}

	for _, testCase := range asserts {
		var result = filterFilesUsingExtensions(testCase.values, &testCase.extensions)

		// Test size
		if len(result) != len(testCase.expected) {
			t.Error(
				"For", testCase.values,
				"expected result count", len(testCase.expected),
				"but got", len(result),
			)
		}

		// Test expected entries
		for _, expectedEntry := range testCase.expected {
			AssertStringSliceContainsString(result, expectedEntry, t)
		}
	}
}

func TestTargetsToFiles(t *testing.T) {
	var mkFile = func(pathElem ...string) *os.File {
		if f, err := os.Create(filepath.Join(pathElem...)); err != nil {
			panic(err)
		} else {
			return f
		}
	}
	var mkDir = func(pathElem ...string) string {
		path := filepath.Join(pathElem...)
		if err := os.Mkdir(path, 0777); err != nil {
			panic(err)
		}
		return path
	}

	// Create directory in temporary
	dir, err := ioutil.TempDir("", "test-")
	if err != nil {
		t.Error(err)
	}

	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	}()

	// Create files structure
	mkFile(dir, "bar.a")
	mkFile(dir, "bar.b")
	mkDir(dir, "baz")
	mkFile(dir, "baz", "foo.a")
	mkFile(dir, "baz", "foo.b")
	mkDir(dir, "zzz")

	type testCase struct {
		targets   []string
		expected  []string
		withError bool
	}

	var asserts = []testCase{
		{
			targets:  []string{filepath.Join(dir)},
			expected: []string{filepath.Join(dir, "bar.a"), filepath.Join(dir, "bar.b")},
		},
		{
			targets:  []string{filepath.Join(dir, "baz")},
			expected: []string{filepath.Join(dir, "baz", "foo.a"), filepath.Join(dir, "baz", "foo.b")},
		},
		{
			targets: []string{filepath.Join(dir), filepath.Join(dir, "baz")},
			expected: []string{
				filepath.Join(dir, "bar.a"), filepath.Join(dir, "bar.b"),
				filepath.Join(dir, "baz", "foo.a"), filepath.Join(dir, "baz", "foo.b"),
			},
		},
		{
			targets:  []string{filepath.Join(dir, "zzz")},
			expected: []string{},
		},
		{
			targets:   []string{filepath.Join(dir, strconv.Itoa(int(time.Now().UnixNano())))},
			expected:  []string{},
		},
	}

	for _, testCase := range asserts {
		var result, err = targetsToFiles(&testCase.targets)

		if testCase.withError && err == nil {
			t.Error("For", testCase.targets, "expects error")
		}

		// Test size
		if len(result) != len(testCase.expected) {
			t.Error(
				"For", testCase.targets,
				"expected result count", len(testCase.expected),
				"but got", len(result),
			)
		}

		// Test expected entries
		for _, expectedEntry := range testCase.expected {
			AssertStringSliceContainsString(result, expectedEntry, t)
		}
	}
}

// @todo: write more tests

// Assert that strings slice contains expected string
func AssertStringSliceContainsString(slice []string, expected string, t *testing.T) {
	for _, item := range slice {
		if item == expected {
			return
		}
	}

	t.Error("In", slice, "value", expected, "was not found")
}

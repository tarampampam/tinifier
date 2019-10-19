package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestTargetsLoad(t *testing.T) {
	dir := createTempDir()

	createFilesAndDirs(
		[][]string{{dir, "baz"}, {dir, "zzz"}},
		[][]string{{dir, "bar.a"}, {dir, "bar.b"}, {dir, "baz", "foo.a"}, {dir, "baz", "foo.b"}},
	)

	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			panic(err)
		}
	}()

	targets = Targets{}

	type testCase struct {
		targets    []string
		extensions []string
		expected   []string
	}

	var asserts = []testCase{
		{
			targets:    []string{dir},
			extensions: []string{".a"},
			expected:   []string{filepath.Join(dir, "bar.a")},
		},
		{
			targets:    []string{dir},
			extensions: []string{".a", ".b"},
			expected:   []string{filepath.Join(dir, "bar.a"), filepath.Join(dir, "bar.b")},
		},
		{
			targets:    []string{dir},
			extensions: []string{".a,.b"},
			expected:   []string{filepath.Join(dir, "bar.a"), filepath.Join(dir, "bar.b")},
		},
		{
			targets:    []string{dir, filepath.Join(dir, "baz"), filepath.Join(dir, "zzz")},
			extensions: []string{".a"},
			expected:   []string{filepath.Join(dir, "bar.a"), filepath.Join(dir, "baz", "foo.a")},
		},
	}

	for _, testCase := range asserts {
		targets = Targets{}
		targets.Load(testCase.targets, &testCase.extensions)

		// Test size
		if len(targets.Files) != len(testCase.expected) {
			t.Error(
				"For", testCase.targets,
				"expected result count", len(testCase.expected),
				"but got", len(targets.Files),
			)
		}

		// Test expected entries
		for _, expectedEntry := range testCase.expected {
			AssertStringSliceContainsString(targets.Files, expectedEntry, t)
		}
	}
}

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
	dir := createTempDir()

	createFilesAndDirs(
		[][]string{{dir, "baz"}, {dir, "zzz"}},
		[][]string{{dir, "bar.a"}, {dir, "bar.b"}, {dir, "baz", "foo.a"}, {dir, "baz", "foo.b"}},
	)

	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			panic(err)
		}
	}()

	type testCase struct {
		targets   []string
		expected  []string
		withError bool
	}

	var asserts = []testCase{
		{
			targets:  []string{dir},
			expected: []string{filepath.Join(dir, "bar.a"), filepath.Join(dir, "bar.b")},
		},
		{
			targets:  []string{filepath.Join(dir, "baz")},
			expected: []string{filepath.Join(dir, "baz", "foo.a"), filepath.Join(dir, "baz", "foo.b")},
		},
		{
			targets: []string{dir, filepath.Join(dir, "baz")},
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
			targets:  []string{filepath.Join(dir, strconv.Itoa(int(time.Now().UnixNano())))},
			expected: []string{},
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

// Assert that strings slice contains expected string
func AssertStringSliceContainsString(slice []string, expected string, t *testing.T) {
	for _, item := range slice {
		if item == expected {
			return
		}
	}

	t.Error("In", slice, "value", expected, "was not found")
}

// Prepare files structure (create files and directories).
func createFilesAndDirs(dirs [][]string, files [][]string) {
	for _, d := range dirs {
		if err := os.Mkdir(filepath.Join(d...), 0777); err != nil {
			panic(err)
		}
	}

	for _, f := range files {
		if _, err := os.Create(filepath.Join(f...)); err != nil {
			panic(err)
		}
	}
}

// Create directory in temporary
func createTempDir() string {
	if dir, err := ioutil.TempDir("", "test-"); err != nil {
		panic(err)
	} else {
		return dir
	}
}

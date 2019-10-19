package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestTargetsLoad(t *testing.T) {
	var (
		dir     = createTempDir()
		targets = Targets{}
	)

	createFilesAndDirs(
		[][]string{{dir, "baz"}, {dir, "zzz"}},
		[][]string{{dir, "bar.a"}, {dir, "bar.b"}, {dir, "baz", "foo.a"}, {dir, "baz", "foo.b"}},
	)

	defer func(d string) {
		if err := os.RemoveAll(d); err != nil {
			panic(err)
		}
	}(dir)

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

		if !reflect.DeepEqual(targets.Files, testCase.expected) {
			t.Errorf("Fir %+x expected %+v, got %+v", testCase.targets, testCase.expected, targets.Files)
		}
	}
}

func TestFilterFilesUsingExtensions(t *testing.T) {
	var cases = []struct {
		values     []string
		extensions []string
		expected   []string
	}{
		{[]string{"foo.bar"}, []string{"bar"}, []string{"foo.bar"}},
		{[]string{"foo.bar", "foo.baz", "baz.bar", "foo.foo"}, []string{"baz", "foo"}, []string{"foo.baz", "foo.foo"}},
		{[]string{"foo.bar", "foo.baz", "baz.bar", "foo.foo"}, []string{"baz,foo"}, []string{"foo.baz", "foo.foo"}},
		{[]string{"aa", "ab", "ac", "d"}, []string{"a,b", "d"}, []string{"aa", "ab", "d"}},
		{[]string{"aa", "ab", "ac"}, []string{"d"}, []string{}},
	}

	for _, testCase := range cases {
		var result = filterFilesUsingExtensions(testCase.values, &testCase.extensions)

		if len(testCase.expected) == 0 {
			if len(result) != 0 {
				t.Errorf("For %+v expected empty result", testCase.values)
			}
		} else {
			if !reflect.DeepEqual(result, testCase.expected) {
				t.Errorf("For %+v expected %+v, got %+v", testCase.values, testCase.expected, result)
			}
		}
	}
}

func TestTargetsToFiles(t *testing.T) {
	dir := createTempDir()

	createFilesAndDirs(
		[][]string{{dir, "baz"}, {dir, "zzz"}},
		[][]string{{dir, "bar.a"}, {dir, "bar.b"}, {dir, "baz", "foo.a"}, {dir, "baz", "foo.b"}},
	)

	defer func(d string) {
		if err := os.RemoveAll(d); err != nil {
			panic(err)
		}
	}(dir)

	var cases = []struct {
		targets   []string
		expected  []string
		withError bool
	}{
		{
			targets:  []string{dir},
			expected: []string{filepath.Join(dir, "bar.a"), filepath.Join(dir, "bar.b")},
		},
		{
			targets:  []string{dir, filepath.Join(dir, "baz", "foo.a")},
			expected: []string{filepath.Join(dir, "bar.a"), filepath.Join(dir, "bar.b"), filepath.Join(dir, "baz", "foo.a")},
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

	for _, testCase := range cases {
		var result, err = targetsToFiles(&testCase.targets)

		if testCase.withError && err == nil {
			t.Error("For", testCase.targets, "expects error")
		}

		if len(testCase.expected) == 0 {
			if len(result) != 0 {
				t.Errorf("For %+v expected empty result", testCase.targets)
			}
		} else {
			if !reflect.DeepEqual(result, testCase.expected) {
				t.Errorf("For %+v expected %+v, got %+v", testCase.targets, testCase.expected, result)
			}
		}
	}
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

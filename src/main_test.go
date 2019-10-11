package main

import (
	"testing"
)

func TestFilterFilesUsingExtensions(t *testing.T) {
	type slicesTestPair struct {
		values     []string
		extensions []string
		expected   []string
	}

	var asserts = []slicesTestPair{
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

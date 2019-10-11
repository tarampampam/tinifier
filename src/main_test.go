package main

import (
	"fmt"
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

		fmt.Println(result)
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

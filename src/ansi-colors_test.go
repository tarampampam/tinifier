package main

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	"strings"
	"testing"
)

func TestColorize(t *testing.T) {
	t.Parallel()

	ansiColors := NewAnsiColors()

	for _, testCase := range []struct {
		value          interface{}
		flags          []AnsiColor
		shouldContains string
	}{
		{
			value:          "foo",
			flags:          []AnsiColor{AnsiBrightFg},
			shouldContains: "foo",
		},
		{
			value:          "foo",
			flags:          []AnsiColor{},
			shouldContains: "foo",
		},
		{
			value:          []string{"1", "2"},
			flags:          []AnsiColor{},
			shouldContains: "[1 2]",
		},
		{
			value:          123,
			flags:          []AnsiColor{AnsiBoldFm},
			shouldContains: "123",
		},
	} {
		res := ansiColors.Colorize(testCase.value, testCase.flags...)

		if _, ok := res.(aurora.Value); !ok {
			t.Errorf("For value %+v returns wrong type", testCase.value)
		}

		if _, ok := res.(fmt.Stringer); !ok {
			t.Errorf("For value %+v result should be 'stringable'", testCase.value)
		}

		if !strings.Contains(res.(fmt.Stringer).String(), testCase.shouldContains) {
			t.Errorf("Value '%+v' should contains '%+v'", res, testCase.shouldContains)
		}

		if len(testCase.flags) > 0 && len(res.(fmt.Stringer).String()) <= len(fmt.Sprint(testCase.value)) {
			t.Errorf("Result length should be grater then original length %d", len(res.(fmt.Stringer).String()))
		}
	}
}

func TestColorizeMany(t *testing.T) {
	t.Parallel()

	createSlice := func(args ...interface{}) []interface{} {
		t.Helper()
		res := make([]interface{}, 0, len(args))
		for _, v := range args {
			res = append(res, v)
		}
		return res
	}

	ansiColors := NewAnsiColors()

	for _, testCase := range []struct {
		value          []interface{}
		flags          []AnsiColor
		shouldContains []interface{}
	}{
		{
			value:          createSlice("foo", []string{"bar"}),
			flags:          []AnsiColor{AnsiBrightFg},
			shouldContains: createSlice("foo", "[bar]"),
		},
		{
			value:          createSlice(123, true),
			flags:          []AnsiColor{AnsiBrightFg},
			shouldContains: createSlice("123", "true"),
		},
		{
			value:          createSlice("baz", [...]int{1, 2}),
			flags:          []AnsiColor{},
			shouldContains: createSlice("baz", "[1 2]"),
		},
	} {
		result := ansiColors.ColorizeMany(testCase.value, testCase.flags...)

		for i := range result {
			if _, ok := result[i].(aurora.Value); !ok {
				t.Errorf("For value %+v returns wrong type", testCase.value[i])
			}

			if _, ok := result[i].(fmt.Stringer); !ok {
				t.Errorf("For value1 %+v result should be 'stringable'", testCase.value[i])
			}

			if !strings.Contains(result[i].(fmt.Stringer).String(), testCase.shouldContains[i].(string)) {
				t.Errorf("Value '%+v' should contains '%+v'", result[i], testCase.shouldContains[i])
			}

			if len(testCase.flags) > 0 && len(result[i].(fmt.Stringer).String()) <= len(fmt.Sprint(testCase.value[i])) {
				t.Errorf("Result length should be grater then original length %d", len(result[i].(fmt.Stringer).String()))
			}
		}
	}
}

package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/logrusorgru/aurora"
)

func TestNewAnsiColors(t *testing.T) {
	t.Parallel()

	if _, ok := NewAnsiColors().(*AnsiColors); !ok {
		t.Error("Constructor returns wrong instance")
	}
}

func TestAnsiToAuroraFlags(t *testing.T) {
	t.Parallel()

	for ansiColor, auroraColor := range map[AnsiColor]aurora.Color{
		AnsiBrightFg: aurora.BrightFg,
		AnsiRedFg:    aurora.RedFg,
		AnsiBoldFm:   aurora.BoldFm,
		AnsiRedBg:    aurora.RedBg,
		AnsiWhiteFg:  aurora.WhiteFg,
		AnsiYellowFg: aurora.YellowFg,
	} {
		if ansiToAuroraFlagsMap[ansiColor] != auroraColor {
			t.Errorf("For ansi color %+v mapped wrong color %+v", ansiColor, auroraColor)
		}
	}
}

func TestColorizeSingleAndMany(t *testing.T) {
	t.Parallel()

	ansiColors := NewAnsiColors()

	for _, c := range []struct {
		value         interface{}
		flags         []AnsiColor
		expectedColor aurora.Color
	}{
		{
			value:         "foo",
			flags:         []AnsiColor{AnsiBrightFg},
			expectedColor: aurora.BrightFg,
		},
		{
			value:         "foo",
			flags:         []AnsiColor{},
			expectedColor: 0,
		},
		{
			value:         [...]string{"1", "2"},
			flags:         []AnsiColor{},
			expectedColor: 0,
		},
		{
			value:         123,
			flags:         []AnsiColor{AnsiBoldFm, AnsiBrightFg},
			expectedColor: aurora.BoldFm | aurora.BrightFg,
		},
	} {
		res := ansiColors.Colorize(c.value, c.flags...)

		if _, ok := res.(aurora.Value); !ok {
			t.Errorf("For value %+v returns wrong type", c.value)
		}

		if fmt.Sprint(res.(aurora.Value).Value()) != fmt.Sprint(c.value) {
			t.Errorf("For value %+v returns modificated value", c.value)
		}

		if res.(aurora.Value).Color() != c.expectedColor {
			t.Errorf("For value %+v returns wrong color", c.value)
		}

		if !reflect.DeepEqual(ansiColors.ColorizeMany(createMixedSlice(c.value), c.flags...), createMixedSlice(res)) {
			t.Errorf("For %+v ColorizeMany works not correctly", c.value)
		}
	}
}

func TestUncolorizeSingleAndMany(t *testing.T) {
	t.Parallel()

	ansiColors := NewAnsiColors()

	for _, c := range []struct {
		value interface{}
	}{
		{
			value: aurora.Colorize("foo", aurora.BrightFg),
		},
		{
			value: aurora.Colorize([...]string{"1", "2"}, aurora.BrightFg),
		},
		{
			value: aurora.Colorize(123, aurora.BrightFg|aurora.RedBg),
		},
		{
			value: 123,
		},
	} {
		var expected = c.value

		// Unwrap value, if needed
		if v, ok := c.value.(aurora.Value); ok {
			expected = v.Value()
		}

		res := ansiColors.Uncolorize(c.value)

		if expected != res {
			t.Errorf("For value %+v returns non-original value", c.value)
		}

		if !reflect.DeepEqual(ansiColors.UncolorizeMany(createMixedSlice(c.value)), createMixedSlice(expected)) {
			t.Errorf("For %+v UncolorizeMany works not correctly", c.value)
		}
	}
}

// Create `[]interface{}`
func createMixedSlice(args ...interface{}) []interface{} {
	res := make([]interface{}, 0, len(args))
	for _, v := range args {
		res = append(res, v)
	}
	return res
}

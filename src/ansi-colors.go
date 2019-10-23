package main

import (
	"github.com/logrusorgru/aurora"
)

type AnsiColor uint

const (
	AnsiBrightFg AnsiColor = iota
	AnsiRedFg
	AnsiBoldFm
	AnsiRedBg
	AnsiWhiteFg
	AnsiYellowFg
)

var ansiToAuroraFlagsMap = map[AnsiColor]aurora.Color{
	AnsiBrightFg: aurora.BrightFg,
	AnsiRedFg:    aurora.RedFg,
	AnsiBoldFm:   aurora.BoldFm,
	AnsiRedBg:    aurora.RedBg,
	AnsiWhiteFg:  aurora.WhiteFg,
	AnsiYellowFg: aurora.YellowFg,
}

type IAnsiColors interface {
	Colorize(v interface{}, colorFlags ...AnsiColor) interface{}
	ColorizeMany(v []interface{}, colorFlags ...AnsiColor) []interface{}
	Uncolorize(v interface{}) interface{}
	UncolorizeMany(v []interface{}) []interface{}
}

type AnsiColors struct {
	aurora aurora.Aurora
}

// Ansi colors constructor.
func NewAnsiColors() IAnsiColors {
	return &AnsiColors{
		aurora: aurora.NewAurora(true),
	}
}

// Colorize input.
func (c *AnsiColors) Colorize(v interface{}, colorFlags ...AnsiColor) interface{} {
	return c.aurora.Colorize(v, c.colorFlagsToAuroraColor(colorFlags...))
}

// Colorize many input values.
func (c *AnsiColors) ColorizeMany(v []interface{}, colorFlags ...AnsiColor) (res []interface{}) {
	for _, v := range v {
		res = append(res, c.Colorize(v, colorFlags...))
	}
	return res
}

// Uncolorize (discolor) input.
func (c *AnsiColors) Uncolorize(v interface{}) interface{} {
	if value, ok := v.(aurora.Value); ok {
		return value.Value()
	}
	return v
}

// Uncolorize (discolor) many input values.
func (c *AnsiColors) UncolorizeMany(v []interface{}) (res []interface{}) {
	for _, v := range v {
		res = append(res, c.Uncolorize(v))
	}
	return res
}

// Convert ansi color flags to aurora color bit-mask.
func (c *AnsiColors) colorFlagsToAuroraColor(colorFlags ...AnsiColor) (color aurora.Color) {
	for _, flag := range colorFlags {
		color = color | ansiToAuroraFlagsMap[flag]
	}
	return color
}

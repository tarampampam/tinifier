package ui_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tarampampam/tinifier/v4/internal/ui"
)

func ExampleTextStyle_Wrap() {
	ui.ColorsEnabled(false) // change to true to see colors

	fmt.Println((ui.FgRed | ui.Bold).Wrap("Foo Bar"))

	// output:
	// Foo Bar
}

func ExampleTextStyle_Start() {
	ui.ColorsEnabled(false) // change to true to see colors

	var style = ui.FgRed | ui.Bold

	fmt.Println(style.Start(), "Foo Bar", style.Reset())

	// output:
	// Foo Bar
}

func TestColorsEnabled(t *testing.T) {
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() { defer wg.Done(); ui.ColorsEnabled(false); ui.ColorsEnabled(true); ui.ColorsEnabled() }()
	}

	wg.Wait()

	ui.ColorsEnabled(false)

	assert.False(t, ui.ColorsEnabled())
	ui.ColorsEnabled(false)
	assert.False(t, ui.ColorsEnabled())

	ui.ColorsEnabled(true)
	assert.True(t, ui.ColorsEnabled())
}

func TestTextStyle_Has(t *testing.T) {
	var s = ui.BgBlack | ui.FgWhite | ui.Bold

	assert.False(t, s.Has(ui.BgWhite))
	assert.False(t, s.Has(ui.FgBlack))
	assert.False(t, s.Has(ui.Italic))
	assert.False(t, s.Has(ui.Reset))
	assert.False(t, s.Has(ui.BgDefault))

	assert.True(t, s.Has(ui.BgBlack))
	assert.True(t, s.Has(ui.FgWhite))
	assert.True(t, s.Has(ui.Bold))
}

func TestTextStyle_Add(t *testing.T) {
	var s = ui.BgBlack

	assert.False(t, s.Has(ui.BgWhite))
	assert.False(t, s.Has(ui.Bold))
	assert.True(t, s.Has(ui.BgBlack))
	assert.False(t, s.Has(ui.Underline))

	s.Add(ui.BgWhite, ui.Bold)

	assert.True(t, s.Has(ui.BgWhite))
	assert.True(t, s.Has(ui.Bold))
	assert.False(t, s.Has(ui.Underline))
}

func TestTextStyle_Remove(t *testing.T) {
	var s = ui.BgBlack | ui.BgWhite | ui.Bold

	assert.True(t, s.Has(ui.BgWhite))
	assert.True(t, s.Has(ui.Bold))
	assert.True(t, s.Has(ui.BgBlack))
	assert.False(t, s.Has(ui.Underline))

	s.Remove(ui.BgWhite, ui.Bold, ui.Underline)

	assert.False(t, s.Has(ui.BgWhite))
	assert.False(t, s.Has(ui.Bold))
	assert.False(t, s.Has(ui.Underline))
}

func TestTextStyle_ColorCodes(t *testing.T) {
	var colorsState = ui.ColorsEnabled()

	defer ui.ColorsEnabled(colorsState)

	for name, tt := range map[string]struct {
		giveTextStyle        ui.TextStyle
		wantStart, wantReset string
	}{
		"Reset":                      {ui.Reset, "\x1b[0m", ""},
		"Reset | FgBlack | BgYellow": {ui.Reset | ui.FgBlack | ui.BgYellow, "\x1b[0m", ""},

		"FgBlack":   {ui.FgBlack, "\x1b[30m", "\x1b[39m"},
		"FgRed":     {ui.FgRed, "\x1b[31m", "\x1b[39m"},
		"FgGreen":   {ui.FgGreen, "\x1b[32m", "\x1b[39m"},
		"FgYellow":  {ui.FgYellow, "\x1b[33m", "\x1b[39m"},
		"FgBlue":    {ui.FgBlue, "\x1b[34m", "\x1b[39m"},
		"FgMagenta": {ui.FgMagenta, "\x1b[35m", "\x1b[39m"},
		"FgCyan":    {ui.FgCyan, "\x1b[36m", "\x1b[39m"},
		"FgWhite":   {ui.FgWhite, "\x1b[37m", "\x1b[39m"},
		"FgDefault": {ui.FgDefault, "\x1b[39m", ""},

		"FgBlack | FgBright":   {ui.FgBlack | ui.FgBright, "\x1b[90m", "\x1b[39m"},
		"FgRed | FgBright":     {ui.FgRed | ui.FgBright, "\x1b[91m", "\x1b[39m"},
		"FgGreen | FgBright":   {ui.FgGreen | ui.FgBright, "\x1b[92m", "\x1b[39m"},
		"FgYellow | FgBright":  {ui.FgYellow | ui.FgBright, "\x1b[93m", "\x1b[39m"},
		"FgBlue | FgBright":    {ui.FgBlue | ui.FgBright, "\x1b[94m", "\x1b[39m"},
		"FgMagenta | FgBright": {ui.FgMagenta | ui.FgBright, "\x1b[95m", "\x1b[39m"},
		"FgCyan | FgBright":    {ui.FgCyan | ui.FgBright, "\x1b[96m", "\x1b[39m"},
		"FgWhite | FgBright":   {ui.FgWhite | ui.FgBright, "\x1b[97m", "\x1b[39m"},
		"FgDefault | FgBright": {ui.FgDefault | ui.FgBright, "\x1b[39m", ""},

		"BgBlack":   {ui.BgBlack, "\x1b[40m", "\x1b[49m"},
		"BgRed":     {ui.BgRed, "\x1b[41m", "\x1b[49m"},
		"BgGreen":   {ui.BgGreen, "\x1b[42m", "\x1b[49m"},
		"BgYellow":  {ui.BgYellow, "\x1b[43m", "\x1b[49m"},
		"BgBlue":    {ui.BgBlue, "\x1b[44m", "\x1b[49m"},
		"BgMagenta": {ui.BgMagenta, "\x1b[45m", "\x1b[49m"},
		"BgCyan":    {ui.BgCyan, "\x1b[46m", "\x1b[49m"},
		"BgWhite":   {ui.BgWhite, "\x1b[47m", "\x1b[49m"},
		"BgDefault": {ui.BgDefault, "\x1b[49m", ""},

		"BgBlack | BgBright":   {ui.BgBlack | ui.BgBright, "\x1b[100m", "\x1b[49m"},
		"BgRed | BgBright":     {ui.BgRed | ui.BgBright, "\x1b[101m", "\x1b[49m"},
		"BgGreen | BgBright":   {ui.BgGreen | ui.BgBright, "\x1b[102m", "\x1b[49m"},
		"BgYellow | BgBright":  {ui.BgYellow | ui.BgBright, "\x1b[103m", "\x1b[49m"},
		"BgBlue | BgBright":    {ui.BgBlue | ui.BgBright, "\x1b[104m", "\x1b[49m"},
		"BgMagenta | BgBright": {ui.BgMagenta | ui.BgBright, "\x1b[105m", "\x1b[49m"},
		"BgCyan | BgBright":    {ui.BgCyan | ui.BgBright, "\x1b[106m", "\x1b[49m"},
		"BgWhite | BgBright":   {ui.BgWhite | ui.BgBright, "\x1b[107m", "\x1b[49m"},
		"BgDefault | BgBright": {ui.BgDefault | ui.BgBright, "\x1b[49m", ""},

		"Bold":      {ui.Bold, "\x1b[1m", "\x1b[22m"},
		"Faint":     {ui.Faint, "\x1b[2m", "\x1b[22m"},
		"Italic":    {ui.Italic, "\x1b[3m", "\x1b[23m"},
		"Underline": {ui.Underline, "\x1b[4m", "\x1b[24m"},
		"Blinking":  {ui.Blinking, "\x1b[5m", "\x1b[25m"},
		"Reverse":   {ui.Reverse, "\x1b[7m", "\x1b[27m"},
		"Invisible": {ui.Invisible, "\x1b[8m", "\x1b[28m"},
		"Strike":    {ui.Strike, "\x1b[9m", "\x1b[29m"},

		"FgBlack(2) | FgBright | Bold | Underline": {
			ui.FgBlack | ui.FgBlack | ui.FgBright | ui.Bold | ui.Underline, //nolint:gocritic
			"\x1b[1;4;90m",
			"\x1b[39;24;22m",
		},

		"<zero>": {0, "", ""},
	} {
		t.Run(name, func(t *testing.T) {
			ui.ColorsEnabled(true) // enable colors

			var start, reset = tt.giveTextStyle.ColorCodes()

			assert.EqualValues(t, tt.wantStart, start)
			assert.EqualValues(t, tt.wantReset, reset)

			assert.EqualValues(t, tt.wantStart, tt.giveTextStyle.Start())
			assert.EqualValues(t, tt.wantStart, tt.giveTextStyle.String())
			assert.EqualValues(t, tt.wantReset, tt.giveTextStyle.Reset())

			ui.ColorsEnabled(false) // disable colors

			start, reset = tt.giveTextStyle.ColorCodes()

			assert.EqualValues(t, tt.wantStart, start) // not changed
			assert.EqualValues(t, tt.wantReset, reset) // not changed

			assert.EqualValues(t, "", tt.giveTextStyle.Start())  // empty
			assert.EqualValues(t, "", tt.giveTextStyle.String()) // empty
			assert.EqualValues(t, "", tt.giveTextStyle.Reset())  // empty
		})
	}
}

func TestTextStyle_Wrap(t *testing.T) {
	var (
		colorsState = ui.ColorsEnabled()
		testStyle   = ui.FgBlack | ui.FgBright | ui.Bold | ui.Underline
	)

	defer ui.ColorsEnabled(colorsState)

	ui.ColorsEnabled(true) // enable colors

	assert.EqualValues(t, "\x1b[1;4;90mFOOBAR\x1b[39;24;22m", testStyle.Wrap("FOOBAR"))

	ui.ColorsEnabled(false) // disable colors

	assert.EqualValues(t, "FOOBAR", testStyle.Wrap("FOOBAR"))
}

var bmWrapRes string

//	BenchmarkColorCodes-8   	25061175	        46.69 ns/op	      32 B/op	       1 allocs/op
func BenchmarkColorCodes(b *testing.B) {
	var colorsState = ui.ColorsEnabled()

	defer ui.ColorsEnabled(colorsState)

	ui.ColorsEnabled(true)
	b.ReportAllocs()
	_ = bmWrapRes //nolint:wsl

	for i := 0; i < b.N; i++ {
		bmWrapRes = (ui.FgGreen | ui.BgRed | ui.Bold).Wrap("FOOBAR")
	}
}

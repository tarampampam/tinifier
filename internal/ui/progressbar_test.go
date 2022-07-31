package ui_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/tarampampam/tinifier/v4/internal/ui"
)

func TestProgressBar_Render_Common(t *testing.T) {
	const (
		width   = 40
		noColor = ui.TextStyle(0)
	)

	var p = ui.NewProgressBar(10, ui.WithWidth(width), ui.WithTheme(ui.ProgressBarTheme{
		PrefixColor:    noColor,
		CounterColor:   noColor,
		Start:          '[',
		StartColor:     noColor,
		Fill:           '#',
		FillColor:      noColor,
		Cursor:         '>',
		CursorColor:    noColor,
		Spacer:         '_',
		SpacerColor:    noColor,
		End:            ']',
		EndColor:       noColor,
		PercentColor:   noColor,
		SeparatorColor: noColor,
		TimeColor:      noColor,
	}))

	p.SetPrefix("foo")
	p.Start(ui.NoOut())
	p.Set(5)

	rendered := p.Render()

	assert.Len(t, rendered, width-1)
	assert.EqualValues(t, "foo [5/10] [########>_______]  50% | 0s", rendered)

	p.SetPrefix("barbaz")
	p.Set(9)

	rendered = p.Render()

	assert.Len(t, rendered, width-1)
	assert.EqualValues(t, "barbaz [9/10] [############>]  90% | 0s", rendered)

	p.SetPrefix("")
	p.Set(1)

	rendered = p.Render()

	assert.Len(t, rendered, width-1)
	assert.EqualValues(t, "[1/10] [##>_________________]  10% | 0s", rendered)
}

func TestProgressBar_Render_Clear(t *testing.T) {
	const (
		width   = 40
		noColor = ui.TextStyle(0)
		noRune  = rune(0)
	)

	var p = ui.NewProgressBar(10, ui.WithWidth(width), ui.WithTheme(ui.ProgressBarTheme{
		PrefixColor:    noColor,
		CounterColor:   noColor,
		Start:          noRune,
		StartColor:     noColor,
		Fill:           noRune,
		FillColor:      noColor,
		Cursor:         noRune,
		CursorColor:    noColor,
		Spacer:         noRune,
		SpacerColor:    noColor,
		End:            noRune,
		EndColor:       noColor,
		PercentColor:   noColor,
		SeparatorColor: noColor,
		TimeColor:      noColor,
	}), ui.WithTimeRounding(time.Hour))

	p.SetPrefix("foo")
	p.Start(ui.NoOut())
	p.Set(5)

	rendered := p.Render()

	assert.Len(t, rendered, width-1)
	assert.EqualValues(t, "foo [5/10]                     50% | 0s", rendered)
}

//	BenchmarkProgressBar_Render-8   	    2466	    478197 ns/op	  698825 B/op	      17 allocs/op // non-optimized
//	BenchmarkProgressBar_Render-8   	   18397	     60097 ns/op	  637584 B/op	       9 allocs/op // optimized
func BenchmarkProgressBar_Render(b *testing.B) {
	ui.ColorsEnabled(true)

	b.ReportAllocs()

	p := ui.NewProgressBar(uint32(b.N))
	p.Start(ui.NoOut())

	for i := 0; i < b.N; i++ {
		p.Add(1)
		p.Render()
	}
}

package ui_test

import (
	"testing"

	"github.com/tarampampam/tinifier/v4/internal/ui"
)

var progressBarRendered string

//	BenchmarkProgressBar_Render-8   	    2466	    478197 ns/op	  698825 B/op	      17 allocs/op
func BenchmarkProgressBar_Render(b *testing.B) {
	ui.ColorsEnabled(true)

	b.ReportAllocs()

	p := ui.NewProgressBar(uint32(b.N))
	p.Start()
	_ = progressBarRendered

	for i := 0; i < b.N; i++ {
		p.Add(1)
		progressBarRendered = p.Render()
	}
}

package ui

import (
	"math"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

type (
	ProgressBar struct {
		max     uint32
		maxText string

		prefix atomic.Value // string
		theme  ProgressBarTheme
		width  uint16 // default (0) = full width

		current uint32 // atomic usage only

		timeRounding time.Duration
		startedAt    time.Time
	}

	ProgressBarTheme struct {
		PrefixColor  TextStyle
		CounterColor TextStyle

		Start      rune
		StartColor TextStyle

		Fill      rune
		FillColor TextStyle

		Cursor      rune
		CursorColor TextStyle

		Spacer      rune
		SpacerColor TextStyle

		End      rune
		EndColor TextStyle

		PercentColor   TextStyle
		SeparatorColor TextStyle
		TimeColor      TextStyle
	}
)

type ProgressBarOption func(*ProgressBar)

func WithTheme(theme ProgressBarTheme) ProgressBarOption {
	return func(p *ProgressBar) { p.theme = theme }
}

func WithTimeRounding(d time.Duration) ProgressBarOption {
	return func(p *ProgressBar) { p.timeRounding = d }
}

func NewProgressBar(max uint32, opts ...ProgressBarOption) *ProgressBar {
	var p = &ProgressBar{
		max:     max,
		maxText: strconv.Itoa(int(max)),
		theme: ProgressBarTheme{
			PrefixColor:  FgBlue | FgBright | Bold,
			CounterColor: FgDefault,

			Start:      '▐',
			StartColor: FgWhite | FgBright,

			Fill:      '█',
			FillColor: FgWhite | FgBright,

			Cursor:      '▒',
			CursorColor: FgWhite | FgBright,

			Spacer:      '░',
			SpacerColor: FgWhite,

			End:      '▌',
			EndColor: FgWhite | FgBright,

			PercentColor:   FgGreen | Bold,
			SeparatorColor: FgWhite,
			TimeColor:      FgWhite | FgBright,
		},
	}

	p.prefix.Store("")

	for _, opt := range opts {
		opt(p)
	}

	if p.timeRounding == time.Duration(0) {
		p.timeRounding = time.Second
	}

	return p
}

// getTerminalSize returns the visible dimensions of the given terminal.
func (p *ProgressBar) getTerminalSize() (width, height uint16) {
	w, h, _ := term.GetSize(int(os.Stdin.Fd()))

	return uint16(w), uint16(h)
}

func (p *ProgressBar) Add(delta uint32) {
	if current := atomic.LoadUint32(&p.current); current+delta > p.max {
		atomic.StoreUint32(&p.current, p.max)
	} else {
		atomic.CompareAndSwapUint32(&p.current, current, current+delta)
	}
}

func (p *ProgressBar) SetPrefix(prefix string) {
	p.prefix.Store(prefix)
}

func (p *ProgressBar) Set(val uint32) { atomic.StoreUint32(&p.current, val) }

func (p *ProgressBar) Start() {
	p.startedAt = time.Now().Add(-time.Hour + time.Second*6)
}

func (p *ProgressBar) Stop() {}

func (p *ProgressBar) digitsCount(n uint32) (count uint32) {
	if n == 0 {
		return 1
	}

	for n > 0 {
		n, count = n/10, count+1
	}

	return
}

func (p *ProgressBar) Render() string {
	var (
		buf strings.Builder

		current   = atomic.LoadUint32(&p.current)
		prefix    = p.prefix.Load().(string)
		elapsed   = time.Since(p.startedAt).Round(p.timeRounding).String()
		leftSize  = uint16(utf8.RuneCountInString(prefix)) + uint16(p.digitsCount(p.max)+p.digitsCount(current)) + 6
		rightSize = uint16(utf8.RuneCountInString(elapsed)) + 10

		width = p.width
	)

	if width == 0 {
		width, _ = p.getTerminalSize()
	}

	var (
		percent  = float64((float32(current) / float32(p.max)) * 100)
		barWidth = width - (leftSize + rightSize)
	)

	if p.theme.Start == 0 {
		barWidth++
	}

	if p.theme.End == 0 {
		barWidth++
	}

	if prefix == "" {
		barWidth++
	}

	const colorsExtraSize = 8 /* colors count */ * 5 /* color size */ * 2 /* color reset */

	buf.Grow(int(width) + colorsExtraSize)

	// prefix
	if prefix != "" {
		buf.WriteString(p.theme.PrefixColor.Start())
		buf.WriteString(prefix)
		buf.WriteString(p.theme.PrefixColor.Reset())
		buf.WriteRune(' ')
	}

	// counter
	buf.WriteString(p.theme.CounterColor.Start())
	buf.WriteRune('[')
	buf.WriteString(strconv.Itoa(int(current)))
	buf.WriteRune('/')
	buf.WriteString(p.maxText)
	buf.WriteRune(']')
	buf.WriteString(p.theme.CounterColor.Reset())
	buf.WriteRune(' ')

	// start
	if p.theme.Start != rune(0) {
		buf.WriteString(p.theme.StartColor.Start())
		buf.WriteRune(p.theme.Start)
		buf.WriteString(p.theme.StartColor.Reset())
	}

	var cursorPos = uint16(math.Round(float64(barWidth) / 100 * percent))

	// fill
	if r := ' '; cursorPos > 0 {
		if p.theme.Fill != rune(0) {
			r = p.theme.Fill
		}

		buf.WriteString(p.theme.FillColor.Reset())
		buf.WriteString(strings.Repeat(string(r), int(cursorPos)))
		buf.WriteString(p.theme.FillColor.Reset())
	}

	// cursor
	if cursorPos < barWidth {
		buf.WriteString(p.theme.CursorColor.Start())
		buf.WriteRune(p.theme.Cursor)
		buf.WriteString(p.theme.CursorColor.Reset())
	}

	// spacer
	if r, spacesCount := ' ', int(barWidth-cursorPos)-1; spacesCount > 0 {
		if p.theme.Spacer != rune(0) {
			r = p.theme.Spacer
		}

		buf.WriteString(p.theme.SpacerColor.Start())
		buf.WriteString(strings.Repeat(string(r), spacesCount))
		buf.WriteString(p.theme.SpacerColor.Reset())
	}

	// end
	if p.theme.End != 0 {
		buf.WriteString(p.theme.EndColor.Start())
		buf.WriteRune(p.theme.End)
		buf.WriteString(p.theme.EndColor.Reset())
	}

	buf.WriteRune(' ')

	// percent
	for i := p.digitsCount(uint32(percent)); i < 3; i++ {
		buf.WriteRune(' ')
	}

	buf.WriteString(p.theme.PercentColor.Start())
	buf.WriteString(strconv.FormatFloat(percent, 'f', 0, 64))
	buf.WriteRune('%')
	buf.WriteString(p.theme.PercentColor.Reset())
	buf.WriteRune(' ')

	// time
	buf.WriteString(p.theme.SeparatorColor.Start())
	buf.WriteRune('|')
	buf.WriteString(p.theme.SeparatorColor.Reset())
	buf.WriteRune(' ')
	buf.WriteString(p.theme.TimeColor.Start())
	buf.WriteString(elapsed)
	buf.WriteString(p.theme.TimeColor.Reset())

	return buf.String()
}

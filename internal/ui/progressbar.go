package ui

import (
	"bytes"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

type (
	// ProgressBar is a progress bar (wow :D).
	ProgressBar struct {
		max          uint32 // maximal progress bar value (X of ???)
		maxText      string // string representation of max (needed for optimization reasons)
		width        uint16 // default (0) = full width
		timeRounding time.Duration
		theme        ProgressBarTheme

		mu        sync.RWMutex // protects all fields above
		prefix    string       // user-defined progress bar prefix string
		current   uint32       // current progress value (??? of X)
		isStarted bool         // is the progress bar drawing started?
		startedAt time.Time    // when the progress bar was started
		onStop    func()
	}

	// ProgressBarTheme defines the theme of the progress bar.
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

// ProgressBarOption is a function that can be used to configure a progress bar.
type ProgressBarOption func(*ProgressBar)

// WithTheme sets the theme of the progress bar.
func WithTheme(theme ProgressBarTheme) ProgressBarOption {
	return func(p *ProgressBar) { p.theme = theme }
}

// WithTimeRounding sets the rounding of the time.
func WithTimeRounding(d time.Duration) ProgressBarOption {
	return func(p *ProgressBar) { p.timeRounding = d }
}

// WithWidth sets the width of the progress bar.
func WithWidth(width uint16) ProgressBarOption {
	return func(p *ProgressBar) { p.width = width }
}

// NewProgressBar creates a new progress bar.
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
		timeRounding: time.Second,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// SetPrefix sets the prefix of the progress bar.
func (p *ProgressBar) SetPrefix(prefix string) { p.mu.Lock(); p.prefix = prefix; p.mu.Unlock() }

// Add adds the given value to the current progress.
func (p *ProgressBar) Add(delta uint32) {
	p.mu.RLock()
	c := p.current
	p.mu.RUnlock()

	p.Set(c + delta)
}

// Set sets the current progress to the given value. If the value is greater than the maximal progress value, it is set
// to the maximal value.
func (p *ProgressBar) Set(val uint32) {
	var n uint32

	if val > p.max {
		n = p.max
	} else {
		n = val
	}

	p.mu.Lock()
	p.current = n
	p.mu.Unlock()
}

func (p *ProgressBar) Start(out interface {
	io.Writer
	WritingMutator
}) {
	p.mu.RLock()
	var isStarted = p.isStarted
	p.mu.RUnlock()

	if isStarted {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.isStarted = true
	p.startedAt = time.Now()
	p.onStop = out.Mutate(func(data *[]byte) {
		var buf bytes.Buffer

		// buf.WriteRune('\n')
		buf.WriteRune('\r')
		buf.WriteString(p.Render())

		*data = append(*data, buf.Bytes()...)
	})
}

func (p *ProgressBar) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.isStarted = false

	if p.onStop != nil {
		p.onStop()
	}
}

// digitsCount returns the number of digits in the given number.
func (p *ProgressBar) digitsCount(n uint32) (count uint32) {
	if n == 0 {
		return 1
	}

	for n > 0 {
		n, count = n/10, count+1 //nolint:gomnd
	}

	return
}

// getTerminalSize returns the visible dimensions of the given terminal.
func (p *ProgressBar) getTerminalSize() (width, height uint16) {
	w, h, _ := term.GetSize(int(os.Stdin.Fd()))

	return uint16(w), uint16(h)
}

func (p *ProgressBar) Render() string { //nolint:funlen,gocyclo
	p.mu.RLock()
	var current, prefix = p.current, p.prefix
	p.mu.RUnlock()

	var (
		elapsed   = time.Since(p.startedAt).Round(p.timeRounding).String()
		leftSize  = uint16(utf8.RuneCountInString(prefix)) + uint16(p.digitsCount(p.max)+p.digitsCount(current)) + 6
		rightSize = uint16(utf8.RuneCountInString(elapsed)) + 10

		width = p.width
	)

	if width == 0 {
		width, _ = p.getTerminalSize()
	}

	var (
		percent  = float64((float32(current) / float32(p.max)) * 100) //nolint:gomnd
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

	const colorsExtraSize = 8 /* colors count */ * 6 /* color size */ * 2 /* color reset */

	var buf strings.Builder

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

		switch {
		case p.theme.Cursor != rune(0):
			buf.WriteRune(p.theme.Cursor)
		case p.theme.Fill != rune(0):
			buf.WriteRune(p.theme.Fill)
		default:
			buf.WriteRune(' ')
		}

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
	var percentText = strconv.FormatFloat(percent, 'f', 0, 64) //nolint:gomnd
	if l := 3 - utf8.RuneCountInString(percentText); l > 0 {   //nolint:gomnd
		buf.WriteString(strings.Repeat(" ", l))
	}

	buf.WriteString(p.theme.PercentColor.Start())
	buf.WriteString(percentText)
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

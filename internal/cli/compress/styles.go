package compress

import (
	"strconv"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/text"
)

var unitsAsIs = progress.Units{ //nolint:gochecknoglobals
	Notation:         "",
	NotationPosition: progress.UnitsNotationPositionBefore,
	Formatter:        func(value int64) string { return strconv.Itoa(int(value)) },
}

func newProgressBar(expectedTrackersNum int, withOverall bool) progress.Writer {
	var (
		progressStyleDefault = progress.Style{
			Name: "StyleCustomized",
			Chars: progress.StyleChars{
				BoxLeft:       "▐",
				BoxRight:      "▌",
				Finished:      "█",
				Finished25:    "░",
				Finished50:    "▒",
				Finished75:    "▓",
				Indeterminate: progress.StyleCharsBlocks.Indeterminate,
				Unfinished:    "░",
			},
			Colors: progress.StyleColors{
				Message: text.Colors{text.FgWhite},
				Error:   text.Colors{text.FgRed, text.Bold},
				Percent: text.Colors{text.FgHiBlue},
				Pinned:  text.Colors{text.BgHiBlack, text.FgWhite, text.Bold},
				Stats:   text.Colors{text.FgHiBlack},
				Time:    text.Colors{text.FgGreen},
				Tracker: text.Colors{text.FgYellow},
				Value:   text.Colors{text.FgCyan},
				Speed:   text.Colors{text.FgMagenta},
			},
			Options:    progress.StyleOptionsDefault,
			Visibility: progress.StyleVisibilityDefault,
		}
	)

	var pw = progress.NewWriter()

	pw.SetNumTrackersExpected(expectedTrackersNum)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetStyle(progressStyleDefault)
	pw.SetUpdateFrequency(time.Millisecond * 100) //nolint:gomnd

	pw.Style().Visibility.Value = false
	pw.Style().Visibility.Percentage = false
	pw.Style().Visibility.Pinned = true
	pw.Style().Visibility.ETA = true
	pw.Style().Visibility.TrackerOverall = withOverall
	pw.Style().Visibility.Tracker = !withOverall
	pw.Style().Options.TimeInProgressPrecision = time.Millisecond
	pw.Style().Options.TimeDonePrecision = time.Millisecond

	return pw
}

package compress

import (
	"strconv"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/text"
)

var (
	progressStyleDefault = progress.Style{ //nolint:gochecknoglobals
		Name: "StyleCustomized",
		Chars: progress.StyleChars{
			BoxLeft:    "▐",
			BoxRight:   "▌",
			Finished:   "█",
			Finished25: "░",
			Finished50: "▒",
			Finished75: "▓",
			Indeterminate: progress.IndeterminateIndicatorMovingLeftToRight(
				"▒█▒", progress.DefaultUpdateFrequency/2, //nolint:gomnd
			),
			Unfinished: "░",
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

	unitsAsIs = progress.Units{ //nolint:gochecknoglobals
		Notation:         "",
		NotationPosition: progress.UnitsNotationPositionBefore,
		Formatter:        func(value int64) string { return strconv.Itoa(int(value)) },
	}
)

package compress

import (
	"fmt"
	"io"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/schollz/progressbar/v3"
)

func newProgress(desc string, max int, writer io.Writer) *progressbar.ProgressBar {
	const throttle time.Duration = 333 * time.Millisecond

	bar := progressbar.NewOptions(max,
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSetWriter(writer),
		progressbar.OptionFullWidth(),
		progressbar.OptionThrottle(throttle),
		progressbar.OptionShowCount(),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionOnCompletion(func() {
			_, _ = fmt.Fprintln(writer, "")
		}),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerHead:    "",
			SaucerPadding: " ",
			BarStart:      "[green]",
			BarEnd:        "[reset]",
		}))

	return bar
}

func newTable(writer io.Writer) *tablewriter.Table {
	table := tablewriter.NewWriter(writer)

	table.SetAutoFormatHeaders(false)
	table.SetAlignment(tablewriter.ALIGN_CENTER)

	return table
}

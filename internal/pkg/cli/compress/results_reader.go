package compress

import (
	"fmt"
	"io"
	"math"
	"path/filepath"

	"github.com/tarampampam/tinifier/internal/pkg/pipeline"

	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
)

type ResultsReader struct {
	table *tablewriter.Table
	stats struct {
		originalSize   uint64
		compressedSize uint64
		savedBytes     int64
		totalFiles     uint32
	}
}

// NewResultsReader creates results reader instance.
func NewResultsReader(writer io.Writer) ResultsReader {
	// create and configure table
	table := tablewriter.NewWriter(writer)
	table.SetAutoFormatHeaders(false)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetHeader([]string{"File Name", "Type", "Size Difference", "Saved"})

	return ResultsReader{
		table: table,
	}
}

func (rr *ResultsReader) Append(result pipeline.TaskResult) {
	rr.stats.originalSize += result.OriginalSize
	rr.stats.compressedSize += result.CompressedSize
	rr.stats.savedBytes += int64(result.OriginalSize) - int64(result.CompressedSize)
	rr.stats.totalFiles++

	// append a row in a table
	rr.table.Append([]string{
		filepath.Base(result.FilePath), // File Name
		result.FileType,                // Type
		fmt.Sprintf( // Size Difference
			"%s  →  %s",
			humanize.IBytes(result.OriginalSize),
			humanize.IBytes(result.CompressedSize),
		),
		fmt.Sprintf( // Saved
			"%s (%s)",
			rr.humanBytesDiff(float64(result.OriginalSize), float64(result.CompressedSize)),
			rr.percentageDiff(float64(result.CompressedSize), float64(result.OriginalSize)),
		),
	})
}

// Draw the table with results.
func (rr *ResultsReader) Draw() {
	rr.table.SetFooter([]string{
		"", // File Name
		"", // Type
		fmt.Sprintf("Total saved (%d files)", rr.stats.totalFiles), // Size Difference
		fmt.Sprintf("%s (%s)", // Saved
			rr.humanBytesDiff(float64(rr.stats.savedBytes)),
			rr.percentageDiff(float64(rr.stats.compressedSize), float64(rr.stats.originalSize)),
		),
	})

	if rr.table.NumLines() > 0 {
		rr.table.Render()
	}
}

// humanBytesDiff formats difference between two values (byte sizes) in human-readable string representation.
func (rr *ResultsReader) humanBytesDiff(first float64, second ...float64) string {
	var diff = first

	if len(second) > 0 {
		diff = first - second[0]
	}

	var sign string

	if diff < 0 {
		sign = "-"
	}

	return fmt.Sprintf("%s%s", sign, humanize.IBytes(uint64(math.Abs(diff))))
}

// percentageDiff calculates difference between passed values in percentage representation.
func (rr *ResultsReader) percentageDiff(from, to float64) string {
	return fmt.Sprintf("%0.2f%%", ((from-to)/to)*100) //nolint:gomnd
}

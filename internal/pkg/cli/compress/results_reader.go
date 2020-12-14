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

type resultsReader struct {
	table *tablewriter.Table
	stats struct {
		originalSize   uint64
		compressedSize uint64
		savedBytes     int64
		totalFiles     uint32
	}
}

// newResultsReader creates results reader instance.
func newResultsReader(writer io.Writer) resultsReader {
	// create and configure table
	table := tablewriter.NewWriter(writer)
	table.SetAutoFormatHeaders(false)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetHeader([]string{"File Name", "Type", "Size Difference", "Saved"})

	return resultsReader{
		table: table,
	}
}

func (rr *resultsReader) Append(result pipeline.TaskResult) {
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
			"%s (-%0.2f%%)",
			rr.humanBytesDiff(float64(result.OriginalSize), float64(result.CompressedSize)),
			rr.percentageDiff(float64(result.CompressedSize), float64(result.OriginalSize)),
		),
	})
}

// Draw the table with results.
func (rr *resultsReader) Draw() {
	rr.table.SetFooter([]string{
		"", // File Name
		"", // Type
		fmt.Sprintf("Total saved (%d files)", rr.stats.totalFiles), // Size Difference
		fmt.Sprintf("%s (-%0.2f%%)", // Saved
			rr.humanBytesDiff(float64(rr.stats.savedBytes)),
			rr.percentageDiff(float64(rr.stats.compressedSize), float64(rr.stats.originalSize)),
		),
	})

	if rr.table.NumLines() > 0 {
		rr.table.Render()
	}
}

// humanBytesDiff formats difference between two values (byte sizes) in human-readable string representation.
func (rr *resultsReader) humanBytesDiff(first float64, second ...float64) string {
	var (
		diff = first
		sign rune
	)

	if len(second) > 0 {
		diff = first - second[0]
	}

	if diff < 0 {
		sign = '-'
	}

	return string(sign) + humanize.IBytes(uint64(math.Abs(diff)))
}

// percentageDiff calculates difference between passed values in percentage representation.
func (rr *resultsReader) percentageDiff(from, to float64) float64 {
	if to <= 0 {
		return 0
	}

	return math.Abs(((from - to) / to) * 100) //nolint:gomnd
}

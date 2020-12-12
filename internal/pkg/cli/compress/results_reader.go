package compress

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
)

type resultsReader struct {
	table *tablewriter.Table
	stats struct {
		originalSizeBytes   uint64
		compressedSizeBytes uint64
		savedBytes          int64
		totalFiles          uint32
	}
}

// newResultsReader creates results reader instance.
func newResultsReader() resultsReader {
	// create and configure table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoFormatHeaders(false)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetHeader([]string{"File Name", "Type", "Size Difference", "Saved"})

	return resultsReader{
		table: table,
	}
}

func (rr *resultsReader) Append(result taskResult) {
	rr.stats.originalSizeBytes += result.originalSizeBytes
	rr.stats.compressedSizeBytes += result.compressedSizeBytes
	rr.stats.savedBytes += int64(result.originalSizeBytes) - int64(result.compressedSizeBytes)
	rr.stats.totalFiles++

	// append a row in a table
	rr.table.Append([]string{
		filepath.Base(result.filePath), // File Name
		result.fileType,                // Type
		fmt.Sprintf( // Size Difference
			"%s  →  %s",
			humanize.IBytes(result.originalSizeBytes),
			humanize.IBytes(result.compressedSizeBytes),
		),
		fmt.Sprintf( // Saved
			"%s (-%0.2f%%)",
			rr.humanBytesDiff(float64(result.originalSizeBytes), float64(result.compressedSizeBytes)),
			rr.percentageDiff(float64(result.compressedSizeBytes), float64(result.originalSizeBytes)),
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
			rr.percentageDiff(float64(rr.stats.compressedSizeBytes), float64(rr.stats.originalSizeBytes)),
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

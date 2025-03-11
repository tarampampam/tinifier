package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"gh.tarampamp.am/tinifier/v5/internal/humanize"
)

type fileStat struct {
	Path, Type         string
	OrigSize, CompSize uint64
	Skipped            bool
}

type fileStats struct {
	Items []fileStat
	mu    sync.Mutex
}

func (fs *fileStats) Add(v fileStat) {
	fs.mu.Lock()
	fs.Items = append(fs.Items, v)
	fs.mu.Unlock()
}

func (fs *fileStats) Table() string { //nolint:funlen
	if len(fs.Items) == 0 {
		return ""
	}

	var (
		columns = make(map[string][4]string, len(fs.Items))

		longestFileName int
		longestType     int
		longestDiffSize int

		totalOrig    int
		totalComp    int
		totalSkipped int
	)

	for _, item := range fs.Items {
		var (
			fileName = filepath.Base(item.Path)
			typeName = item.Type
			diffSize = fmt.Sprintf("%s → %s",
				humanize.Bytes(item.OrigSize),
				humanize.Bytes(item.CompSize),
			)
			deltaSize = fmt.Sprintf("%s, %s",
				humanize.BytesDiff(item.CompSize, item.OrigSize),
				humanize.PercentageDiff(item.CompSize, item.OrigSize),
			)
		)

		if v := utf8.RuneCountInString(fileName); v > longestFileName {
			longestFileName = v
		}

		if v := utf8.RuneCountInString(typeName); v > longestType {
			longestType = v
		}

		if v := utf8.RuneCountInString(diffSize); v > longestDiffSize {
			longestDiffSize = v
		}

		totalOrig += int(item.OrigSize) //nolint:gosec
		totalComp += int(item.CompSize) //nolint:gosec

		if item.Skipped {
			totalSkipped++
		}

		columns[item.Path] = [4]string{fileName, typeName, diffSize, deltaSize}
	}

	var b strings.Builder

	b.Grow(len(fs.Items) * (longestFileName + longestType + longestDiffSize + 32)) //nolint:mnd // preallocate buffer

	const pad = "  "

	for i, item := range fs.Items {
		b.WriteRune(' ')

		if item.Skipped {
			b.WriteRune('✘')
		} else {
			b.WriteRune('✔')
		}

		b.WriteRune(' ')

		row := columns[item.Path]
		fileName, typeName, diffSize, deltaSize := row[0], row[1], row[2], row[3]

		b.WriteString(fileName)
		b.WriteString(strings.Repeat(" ", max(0, longestFileName-utf8.RuneCountInString(fileName))))
		b.WriteString(pad)

		b.WriteString(typeName)
		b.WriteString(strings.Repeat(" ", max(0, longestType-utf8.RuneCountInString(typeName))))
		b.WriteString(pad)

		if !item.Skipped {
			b.WriteString(diffSize)
			b.WriteString(strings.Repeat(" ", max(0, longestDiffSize-utf8.RuneCountInString(diffSize))))
			b.WriteString(pad)

			b.WriteRune('(')
			b.WriteString(deltaSize)
			b.WriteRune(')')
		} else {
			b.WriteString("(skipped)")
		}

		if i < len(fs.Items)-1 {
			b.WriteRune('\n')
		}
	}

	if l := len(fs.Items); l > 1 && totalSkipped < l {
		b.WriteRune('\n')
		b.WriteString("   ") // [space][emoji][space]
		b.WriteString(strings.Repeat(" ", max(0, longestFileName)))
		b.WriteString(pad)

		const total = "Total:"

		b.WriteString(total)
		b.WriteString(strings.Repeat(" ", max(0, longestType-utf8.RuneCountInString(total))))
		b.WriteString(pad)

		var (
			diffSize = fmt.Sprintf("%s → %s",
				humanize.Bytes(totalOrig),
				humanize.Bytes(totalComp),
			)
			deltaSize = fmt.Sprintf("%s, %s",
				humanize.BytesDiff(totalOrig, totalComp),
				humanize.PercentageDiff(totalOrig, totalComp),
			)
		)

		b.WriteString(diffSize)
		b.WriteString(strings.Repeat(" ", max(0, longestDiffSize-utf8.RuneCountInString(diffSize))))
		b.WriteString(pad)

		b.WriteRune('(')
		b.WriteString(deltaSize)
		b.WriteRune(')')
	}

	return b.String()
}

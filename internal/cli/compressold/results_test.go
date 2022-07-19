package compressold

// import (
// 	"bytes"
// 	"testing"
//
// 	"github.com/tarampampam/tinifier/v4/internal/pkg/pool"
//
// 	"github.com/stretchr/testify/assert"
// )
//
// func TestResultsReader_Draw(t *testing.T) {
// 	cases := []struct {
// 		name           string
// 		giveResults    []pool.Result
// 		wantSubstrings []string
// 	}{
// 		{
// 			name: "basic",
// 			giveResults: []pool.Result{
// 				{FileType: "foo/foo", FilePath: "/tmp/foo.png", OriginalSize: 10000, CompressedSize: 600},
// 			},
// 			wantSubstrings: []string{
// 				"File Name", "Type", "Size Difference", "Saved", // header
// 				"-------------------", // line
// 				" foo.png ", "foo/foo", "9.8 KiB  →  600 B", "9.2 KiB (-94.00%)",
// 				"Total saved (1 files)", "9.2 KiB (-94.00%)", // footer
// 			},
// 		},
// 		{
// 			name: "two results",
// 			giveResults: []pool.Result{
// 				{FileType: "foo/foo", FilePath: "/tmp/foo.png", OriginalSize: 10000, CompressedSize: 600},
// 				{FileType: "bar/bar", FilePath: "/tmp/bar.jpg", OriginalSize: 600, CompressedSize: 10000},
// 			},
// 			wantSubstrings: []string{
// 				"File Name", "Type", "Size Difference", "Saved", // header
// 				"-------------------", // line
// 				" foo.png ", "foo/foo", "9.8 KiB  →  600 B", "9.2 KiB (-94.00%)",
// 				" bar.jpg ", "bar/bar", "600 B  →  9.8 KiB", "-9.2 KiB (1566.67%)",
// 				"Total saved (2 files)", "0 B (0.00%)", // footer
// 			},
// 		},
// 	}
//
// 	for _, tt := range cases {
// 		t.Run(tt.name, func(t *testing.T) {
// 			buf := bytes.NewBuffer([]byte{})
//
// 			rr := NewResultsReader(buf)
//
// 			for _, res := range tt.giveResults {
// 				rr.Append(res)
// 			}
//
// 			rr.Draw()
//
// 			// t.Log(buf.String())
//
// 			for _, res := range tt.wantSubstrings {
// 				assert.Contains(t, buf.String(), res)
// 			}
// 		})
// 	}
// }
//
// func TestResultsReader_DrawSkipOnEmptyRows(t *testing.T) {
// 	buf := bytes.NewBuffer([]byte{})
// 	rr := NewResultsReader(buf)
//
// 	rr.Draw()
//
// 	assert.Empty(t, buf.String())
// }

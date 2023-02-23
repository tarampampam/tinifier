package compress_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/tinifier/v4/internal/cli/compress"
)

func TestStatsStorage(t *testing.T) {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		stats       = compress.NewStatsStorage(0)
	)

	defer cancel()

	go stats.Watch(ctx)

	stats.Push(ctx, compress.CompressionStat{
		FilePath:       "foo",
		FileType:       "foo/type",
		CompressedSize: 100,
		OriginalSize:   200,
	})

	stats.Push(ctx, compress.CompressionStat{
		FilePath:       "bar",
		FileType:       "bar/type",
		CompressedSize: 200,
		OriginalSize:   300,
	})

	runtime.Gosched()

	cancel() // optional
	stats.Close()

	require.Len(t, stats.History(), 2)
	require.EqualValues(t, 500, stats.TotalOriginalSize())
	require.EqualValues(t, 300, stats.TotalCompressedSize())
	require.EqualValues(t, 200, stats.TotalSavedBytes())
	require.EqualValues(t, 2, stats.TotalFiles())

	stats.Close()
	stats.Close()
	stats.Close()

	require.EqualValues(t, 500, stats.TotalOriginalSize())
}

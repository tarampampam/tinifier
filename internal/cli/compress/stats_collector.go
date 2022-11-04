package compress

import (
	"context"
	"sync/atomic"
)

type (
	CompressionStat struct {
		filePath, fileType           string
		compressedSize, originalSize uint64
	}

	StatsCollector struct {
		ch      chan CompressionStat
		history []CompressionStat

		totalOriginalSize   uint64
		totalCompressedSize uint64

		closed uint32
		close  chan struct{}
	}
)

func NewStatsCollector(expectedHistoryLen int) *StatsCollector {
	return &StatsCollector{
		ch:      make(chan CompressionStat, 1),
		history: make([]CompressionStat, 0, expectedHistoryLen),
		close:   make(chan struct{}),
	}
}

func (s StatsCollector) Watch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case <-s.close:
			return

		case stat, isOpened := <-s.ch:
			if !isOpened {
				return
			}

			s.history = append(s.history, stat)
			s.totalOriginalSize += stat.originalSize
			s.totalCompressedSize += stat.compressedSize
		}
	}
}

func (s StatsCollector) Push(ctx context.Context, stat CompressionStat) {
	select {
	case <-ctx.Done():
	case <-s.close:
	case s.ch <- stat:
	}
}

func (s StatsCollector) Close() {
	if atomic.CompareAndSwapUint32(&s.closed, 0, 1) {
		close(s.close)
		close(s.ch)
	}
}

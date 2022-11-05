package compress

import (
	"context"
	"sync/atomic"
)

type (
	StatsCollector interface {
		Watch(context.Context)
		Push(context.Context, CompressionStat)
		Close()

		History() []CompressionStat
		TotalOriginalSize() uint64
		TotalCompressedSize() uint64
		TotalSavedBytes() int64
		TotalFiles() uint32
	}

	CompressionStat struct {
		FilePath, FileType           string
		CompressedSize, OriginalSize uint64
	}

	StatsStorage struct {
		ch      chan CompressionStat
		history []CompressionStat

		totalOriginalSize   uint64
		totalCompressedSize uint64
		totalSavedBytes     int64

		closed uint32
		close  chan struct{}
	}
)

var _ StatsCollector = (*StatsStorage)(nil) // ensure that struct implements the StatsCollector interface

func NewStatsStorage(expectedHistoryLen int) *StatsStorage {
	return &StatsStorage{
		ch:      make(chan CompressionStat, 1),
		history: make([]CompressionStat, 0, expectedHistoryLen),
		close:   make(chan struct{}),
	}
}

func (s *StatsStorage) History() []CompressionStat  { return s.history }
func (s *StatsStorage) TotalOriginalSize() uint64   { return s.totalOriginalSize }
func (s *StatsStorage) TotalCompressedSize() uint64 { return s.totalCompressedSize }
func (s *StatsStorage) TotalSavedBytes() int64      { return s.totalSavedBytes }
func (s *StatsStorage) TotalFiles() uint32          { return uint32(len(s.history)) }

func (s *StatsStorage) Watch(ctx context.Context) {
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
			s.totalOriginalSize += stat.OriginalSize
			s.totalCompressedSize += stat.CompressedSize
			s.totalSavedBytes += int64(stat.OriginalSize) - int64(stat.CompressedSize)
		}
	}
}

func (s *StatsStorage) Push(ctx context.Context, stat CompressionStat) {
	select {
	case <-ctx.Done():
	case <-s.close:
	case s.ch <- stat:
	}
}

func (s *StatsStorage) Close() {
	if atomic.CompareAndSwapUint32(&s.closed, 0, 1) {
		close(s.close)
		close(s.ch)
	}
}

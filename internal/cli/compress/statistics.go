package compress

import (
	"context"
	"sync"
	"sync/atomic"
)

type (
	StatsCollector interface {
		Watch(context.Context)                 // starts collecting stats
		Push(context.Context, CompressionStat) // pushes a new stat
		Close()                                // stops collecting stats

		History() []CompressionStat  // returns all collected stats
		TotalOriginalSize() uint64   // returns total original size of all files
		TotalCompressedSize() uint64 // returns total compressed size of all files
		TotalSavedBytes() int64      // returns total saved bytes of all files
		TotalFiles() uint32          // returns total number of files
	}

	// CompressionStat represents a single compression stat.
	CompressionStat struct {
		FilePath, FileType           string
		CompressedSize, OriginalSize uint64
	}

	// StatsStorage is a storage for compression stats.
	StatsStorage struct {
		mu sync.Mutex

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

// NewStatsStorage creates a new StatsStorage.
func NewStatsStorage(expectedHistoryLen int) *StatsStorage {
	return &StatsStorage{
		ch:      make(chan CompressionStat, 1),
		history: make([]CompressionStat, 0, expectedHistoryLen),
		close:   make(chan struct{}),
	}
}

func (s *StatsStorage) History() []CompressionStat {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.history
}

func (s *StatsStorage) TotalOriginalSize() (total uint64) {
	s.mu.Lock()
	total = s.totalOriginalSize
	s.mu.Unlock()

	return
}

func (s *StatsStorage) TotalCompressedSize() (total uint64) {
	s.mu.Lock()
	total = s.totalCompressedSize
	s.mu.Unlock()

	return
}

func (s *StatsStorage) TotalSavedBytes() (total int64) {
	s.mu.Lock()
	total = s.totalSavedBytes
	s.mu.Unlock()

	return
}

func (s *StatsStorage) TotalFiles() (total uint32) {
	s.mu.Lock()
	total = uint32(len(s.history))
	s.mu.Unlock()

	return
}

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

			s.mu.Lock()
			s.history = append(s.history, stat)
			s.totalOriginalSize += stat.OriginalSize
			s.totalCompressedSize += stat.CompressedSize
			s.totalSavedBytes += int64(stat.OriginalSize) - int64(stat.CompressedSize)
			s.mu.Unlock()
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

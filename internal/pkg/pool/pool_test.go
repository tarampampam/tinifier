package pool

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeFileInfo struct {
	s uint64
	t string
}

func (fi *fakeFileInfo) Size() uint64 { return fi.s }
func (fi *fakeFileInfo) Type() string { return fi.t }

type fakeWorker struct {
	upload      func(ctx context.Context, filePath string) (string, FileInfo, error)
	download    func(ctx context.Context, url, toFilePath string) (FileInfo, error)
	copyContent func(fromFilePath, toFilePath string) error
	removeFile  func(filePath string) error
	preRun      func(task Task)

	uploadExecCounter      uint32
	downloadExecCounter    uint32
	copyContentExecCounter uint32
	removeFileExecCounter  uint32
	preRunExecCounter      uint32
}

func (w *fakeWorker) Upload(ctx context.Context, fp string) (string, FileInfo, error) {
	atomic.AddUint32(&w.uploadExecCounter, 1)

	return w.upload(ctx, fp)
}

func (w *fakeWorker) Download(ctx context.Context, url, fp string) (FileInfo, error) {
	atomic.AddUint32(&w.downloadExecCounter, 1)

	return w.download(ctx, url, fp)
}

func (w *fakeWorker) CopyContent(from, to string) error {
	atomic.AddUint32(&w.copyContentExecCounter, 1)

	return w.copyContent(from, to)
}

func (w *fakeWorker) RemoveFile(fp string) error {
	atomic.AddUint32(&w.removeFileExecCounter, 1)

	return w.removeFile(fp)
}

func (w *fakeWorker) PreTaskRun(task Task) {
	atomic.AddUint32(&w.preRunExecCounter, 1)
	w.preRun(task)
}

const fakeURL = "https://example.com/foo"

func newFakeWorker(
	upload func(ctx context.Context, filePath string) (string, FileInfo, error),
	download func(ctx context.Context, url, toFilePath string) (FileInfo, error),
	copyContent func(fromFilePath, toFilePath string) error,
	removeFile func(filePath string) error,
	preRun func(task Task),
) *fakeWorker {
	return &fakeWorker{
		upload:      upload,
		download:    download,
		copyContent: copyContent,
		removeFile:  removeFile,
		preRun:      preRun,
	}
}

func TestPool_Run(t *testing.T) {
	targets := []string{"a", "b", "c", "d", "e"}

	worker := newFakeWorker(
		func(ctx context.Context, filePath string) (string, FileInfo, error) { // upload
			assert.NotNil(t, ctx)
			assert.False(t, strings.HasSuffix(filePath, ".tiny"))

			return fakeURL, &fakeFileInfo{s: 22, t: "type1"}, nil
		},
		func(ctx context.Context, url, toFilePath string) (FileInfo, error) { // download
			assert.NotNil(t, ctx)
			assert.Equal(t, fakeURL, url)
			assert.True(t, strings.HasSuffix(toFilePath, ".tiny"))

			return &fakeFileInfo{s: 11, t: "type2"}, nil
		},
		func(fromFilePath, toFilePath string) error { // copyContent
			assert.False(t, strings.HasSuffix(toFilePath, ".tiny"))
			assert.True(t, strings.HasSuffix(fromFilePath, ".tiny"))

			return nil
		},
		func(filePath string) error { // removeFile
			assert.True(t, strings.HasSuffix(filePath, ".tiny"))

			return nil
		},
		func(task Task) { // preRun
			assert.Equal(t, uint(len(targets)), task.TasksCount)
			assert.Contains(t, []uint{1, 2, 3, 4, 5}, task.TaskNumber)
			assert.Contains(t, targets, task.FilePath)
		},
	)

	p := NewPool(context.Background(), worker)

	results, resCounter := p.Run(targets, 2), 0

	for {
		res, isOpened := <-results
		if !isOpened {
			break
		}

		resCounter++

		assert.Contains(t, targets, res.Task.FilePath)
		assert.Equal(t, uint(len(targets)), res.Task.TasksCount)

		assert.Equal(t, uint64(22), res.OriginalSize)
		assert.Equal(t, uint64(11), res.CompressedSize)
		assert.Equal(t, "type2", res.FileType)
		assert.Contains(t, targets, res.FilePath)

		assert.NoError(t, res.Err)
	}

	assert.Equal(t, 5, resCounter)
	assert.Equal(t, uint32(5), worker.uploadExecCounter)
	assert.Equal(t, uint32(5), worker.downloadExecCounter)
	assert.Equal(t, uint32(5), worker.copyContentExecCounter)
	assert.Equal(t, uint32(5), worker.removeFileExecCounter)
	assert.Equal(t, uint32(5), worker.preRunExecCounter)
}

func TestPool_RunWithCancelledContext(t *testing.T) {
	targets := []string{"a", "b"}

	worker := newFakeWorker(
		func(ctx context.Context, filePath string) (string, FileInfo, error) { // upload
			t.Error("should not be executed")

			return fakeURL, &fakeFileInfo{s: 22, t: "type1"}, nil
		},
		func(ctx context.Context, url, toFilePath string) (FileInfo, error) { // download
			t.Error("should not be executed")

			return &fakeFileInfo{s: 11, t: "type2"}, nil
		},
		func(fromFilePath, toFilePath string) error { // copyContent
			t.Error("should not be executed")

			return nil
		},
		func(filePath string) error { // removeFile
			t.Error("should not be executed")

			return nil
		},
		func(task Task) { // preRun
			t.Error("should not be executed")
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	p := NewPool(ctx, worker)

	cancel() // <-- important

	results, resCounter := p.Run(targets, 2), 0

	for {
		_, isOpened := <-results
		if !isOpened {
			break
		}

		t.Error("should not be executed")
	}

	assert.Equal(t, 0, resCounter)
}

func TestPool_RunWithCancelledContextWhileResultsReading(t *testing.T) {
	targets := []string{"a", "b", "c", "d", "e", "f"}

	worker := newFakeWorker(
		func(ctx context.Context, filePath string) (string, FileInfo, error) { // upload
			return fakeURL, &fakeFileInfo{s: 22, t: "type1"}, nil
		},
		func(ctx context.Context, url, toFilePath string) (FileInfo, error) { // download
			return &fakeFileInfo{s: 11, t: "type2"}, nil
		},
		func(fromFilePath, toFilePath string) error { // copyContent
			return nil
		},
		func(filePath string) error { // removeFile
			return nil
		},
		func(task Task) { // preRun
			// do nothing
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := NewPool(ctx, worker)

	results, resCounter := p.Run(targets, 2), 0

	for {
		_, isOpened := <-results
		if !isOpened {
			break
		}

		resCounter++

		cancel() // <-- important
	}

	assert.InDelta(t, 2, resCounter, 1)
}

func TestPool_RunWithUploadingError(t *testing.T) {
	targets := []string{"a", "b", "c", "d", "e"}

	worker := newFakeWorker(
		func(ctx context.Context, filePath string) (string, FileInfo, error) { // upload
			return "", nil, errors.New("fake error") // <-- error is important
		},
		func(ctx context.Context, url, toFilePath string) (FileInfo, error) { // download
			t.Error("should not be executed")

			return &fakeFileInfo{s: 11, t: "type2"}, nil
		},
		func(fromFilePath, toFilePath string) error { // copyContent
			t.Error("should not be executed")

			return nil
		},
		func(filePath string) error { // removeFile
			t.Error("should not be executed")

			return nil
		},
		func(task Task) { // preRun
			// no nothing
		},
	)

	p := NewPool(context.Background(), worker)

	results, resCounter := p.Run(targets, 2), 0

	for {
		res, isOpened := <-results
		if !isOpened {
			break
		}

		resCounter++

		assert.EqualError(t, res.Err, "fake error")
	}

	assert.Equal(t, 5, resCounter)
	assert.Equal(t, uint32(5), worker.uploadExecCounter)
	assert.Equal(t, uint32(0), worker.downloadExecCounter)
	assert.Equal(t, uint32(0), worker.copyContentExecCounter)
	assert.Equal(t, uint32(0), worker.removeFileExecCounter)
	assert.Equal(t, uint32(5), worker.preRunExecCounter)
}

func TestPool_RunWithDownloadingError(t *testing.T) {
	targets := []string{"a", "b", "c", "d", "e"}

	worker := newFakeWorker(
		func(ctx context.Context, filePath string) (string, FileInfo, error) { // upload
			return fakeURL, &fakeFileInfo{s: 22, t: "type1"}, nil
		},
		func(ctx context.Context, url, toFilePath string) (FileInfo, error) { // download
			return nil, errors.New("fake error") // <-- error is important
		},
		func(fromFilePath, toFilePath string) error { // copyContent
			t.Error("should not be executed")

			return nil
		},
		func(filePath string) error { // removeFile
			return nil
		},
		func(task Task) { // preRun
			// no nothing
		},
	)

	p := NewPool(context.Background(), worker)

	results, resCounter := p.Run(targets, 2), 0

	for {
		res, isOpened := <-results
		if !isOpened {
			break
		}

		resCounter++

		assert.EqualError(t, res.Err, "fake error")
	}

	assert.Equal(t, 5, resCounter)
	assert.Equal(t, uint32(5), worker.uploadExecCounter)
	assert.Equal(t, uint32(5), worker.downloadExecCounter)
	assert.Equal(t, uint32(0), worker.copyContentExecCounter)
	assert.Equal(t, uint32(5), worker.removeFileExecCounter)
	assert.Equal(t, uint32(5), worker.preRunExecCounter)
}

func TestPool_RunWithCopyContentError(t *testing.T) {
	targets := []string{"a", "b", "c", "d", "e"}

	worker := newFakeWorker(
		func(ctx context.Context, filePath string) (string, FileInfo, error) { // upload
			return fakeURL, &fakeFileInfo{s: 22, t: "type1"}, nil
		},
		func(ctx context.Context, url, toFilePath string) (FileInfo, error) { // download
			return &fakeFileInfo{s: 11, t: "type2"}, nil
		},
		func(fromFilePath, toFilePath string) error { // copyContent
			return errors.New("fake error") // <-- error is important
		},
		func(filePath string) error { // removeFile
			return nil
		},
		func(task Task) { // preRun
			// no nothing
		},
	)

	p := NewPool(context.Background(), worker)

	results, resCounter := p.Run(targets, 2), 0

	for {
		res, isOpened := <-results
		if !isOpened {
			break
		}

		resCounter++

		assert.EqualError(t, res.Err, "fake error")
	}

	assert.Equal(t, 5, resCounter)
	assert.Equal(t, uint32(5), worker.uploadExecCounter)
	assert.Equal(t, uint32(5), worker.downloadExecCounter)
	assert.Equal(t, uint32(5), worker.copyContentExecCounter)
	assert.Equal(t, uint32(5), worker.removeFileExecCounter)
	assert.Equal(t, uint32(5), worker.preRunExecCounter)
}

func TestPool_RunWithRemoveFileError(t *testing.T) {
	targets := []string{"a", "b", "c", "d", "e"}

	worker := newFakeWorker(
		func(ctx context.Context, filePath string) (string, FileInfo, error) { // upload
			return fakeURL, &fakeFileInfo{s: 22, t: "type1"}, nil
		},
		func(ctx context.Context, url, toFilePath string) (FileInfo, error) { // download
			return &fakeFileInfo{s: 11, t: "type2"}, nil
		},
		func(fromFilePath, toFilePath string) error { // copyContent
			return nil
		},
		func(filePath string) error { // removeFile
			return errors.New("fake error") // <-- error is important
		},
		func(task Task) { // preRun
			// no nothing
		},
	)

	p := NewPool(context.Background(), worker)

	results, resCounter := p.Run(targets, 2), 0

	for {
		res, isOpened := <-results
		if !isOpened {
			break
		}

		resCounter++

		assert.EqualError(t, res.Err, "fake error")
	}

	assert.Equal(t, 5, resCounter)
	assert.Equal(t, uint32(5), worker.uploadExecCounter)
	assert.Equal(t, uint32(5), worker.downloadExecCounter)
	assert.Equal(t, uint32(5), worker.copyContentExecCounter)
	assert.Equal(t, uint32(5), worker.removeFileExecCounter)
	assert.Equal(t, uint32(5), worker.preRunExecCounter)
}

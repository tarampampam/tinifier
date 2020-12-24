package compress

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/tarampampam/tinifier/internal/pkg/pool"
	"github.com/tarampampam/tinifier/internal/pkg/retry"
	"github.com/tarampampam/tinifier/internal/pkg/validator"
	"github.com/tarampampam/tinifier/pkg/tinypng"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type keysKeeper interface {
	Get() (string, error)
	Remove(keys ...string)
}

type fInfo struct {
	size uint64
	t    string
}

func (i *fInfo) Size() uint64 { return i.size }
func (i *fInfo) Type() string { return i.t }

var errNoAvailableAPIKey = errors.New("no one API key can be used")

type Worker struct {
	log    *zap.Logger
	keeper keysKeeper
	tiny   *tinypng.Client

	retryAttempts uint
	retryInterval time.Duration
}

func (w *Worker) PreTaskRun(task pool.Task) {
	w.log.Info(fmt.Sprintf("[%d of %d] Compressing file \"%s\"", task.TaskNumber, task.TasksCount, task.FilePath))
}

func (w *Worker) Upload(ctx context.Context, filePath string) (string, pool.FileInfo, error) {
	const uploadTimeout = time.Minute * 3 // TODO do not hardcode timeout, calculate it

	var (
		stat os.FileInfo
		info *tinypng.CompressionResult
	)

	if err := retry.Do(
		func(attemptNum uint) error {
			key, keyErr := w.refreshTinyKey()
			if keyErr != nil {
				return errNoAvailableAPIKey
			}

			file, fileInfo, openErr := w.openSourceFile(filePath)
			if openErr != nil {
				return openErr
			}
			defer func() { _ = file.Close() }()

			compResponse, uplErr := w.tiny.CompressImage(file, uploadTimeout)
			if uplErr != nil {
				w.log.Warn("Image uploading failed",
					zap.Error(uplErr),
					zap.String("file", filePath),
					zap.Uint("attempt", attemptNum),
					zap.String("key", key),
				)

				if errors.Is(uplErr, tinypng.ErrTooManyRequests) || errors.Is(uplErr, tinypng.ErrUnauthorized) {
					w.keeper.Remove(key)
				}

				return uplErr
			}

			stat = fileInfo
			info = compResponse

			return nil
		},
		retry.WithContext(ctx),
		retry.WithAttempts(w.retryAttempts),
		retry.WithDelay(w.retryInterval),
		retry.WithLastErrorReturning(),
		retry.WithRetryStoppingErrors(errNoAvailableAPIKey, tinypng.ErrTooManyRequests, tinypng.ErrUnauthorized),
	); err != nil {
		return "", nil, err
	}

	return info.Output.URL, &fInfo{
		size: uint64(stat.Size()),
		t:    "", // TODO detect source file type
	}, nil
}

func (w *Worker) openSourceFile(path string) (*os.File, os.FileInfo, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, nil, err
	}

	var keepFileOpened bool

	defer func() {
		if !keepFileOpened {
			_ = file.Close()
		}
	}()

	stat, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}

	if !stat.Mode().IsRegular() {
		return nil, nil, errors.New("is not regular file")
	}

	if ok, valErr := validator.IsImage(file); !ok || valErr != nil {
		return nil, nil, errors.New("wrong image file")
	}

	if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
		return nil, nil, seekErr
	}

	keepFileOpened = true

	return file, stat, nil
}

func (w *Worker) Download(ctx context.Context, url string, toFilePath string) (pool.FileInfo, error) {
	const downloadTimeout = time.Minute * 2 // TODO do not hardcode timeout, calculate it

	if err := retry.Do(
		func(attemptNum uint) error {
			key, keyErr := w.refreshTinyKey()
			if keyErr != nil {
				return errNoAvailableAPIKey
			}

			file, err := os.OpenFile(toFilePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
			if err != nil {
				return err
			}
			defer func() { _ = file.Close() }()

			if _, err = w.tiny.DownloadImage(url, file, downloadTimeout); err != nil {
				w.log.Warn("Compressed image downloading failed",
					zap.Error(err),
					zap.String("file", toFilePath),
					zap.Uint("attempt", attemptNum),
					zap.String("key", key),
				)

				return err
			}

			return nil
		},
		retry.WithContext(ctx),
		retry.WithAttempts(w.retryAttempts),
		retry.WithDelay(w.retryInterval),
		retry.WithLastErrorReturning(),
		retry.WithRetryStoppingErrors(errNoAvailableAPIKey, tinypng.ErrTooManyRequests, tinypng.ErrUnauthorized),
	); err != nil {
		return nil, err
	}

	stat, err := os.Stat(toFilePath)
	if err != nil {
		return nil, err
	}

	return &fInfo{
		size: uint64(stat.Size()),
		t:    "", // TODO detect temp file type
	}, nil
}

func (w *Worker) CopyContent(fromFilePath, toFilePath string) error {
	fromFile, err := os.OpenFile(fromFilePath, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer fromFile.Close()

	toFile, err := os.OpenFile(toFilePath, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	defer toFile.Close()

	_, copyErr := io.Copy(toFile, fromFile)

	return copyErr
}

func (w *Worker) RemoveFile(filePath string) error {
	return os.Remove(filePath)
}

func (w *Worker) refreshTinyKey() (string, error) {
	key, err := w.keeper.Get()
	if err != nil {
		return "", err
	}

	w.tiny.SetAPIKey(key)

	return key, nil
}

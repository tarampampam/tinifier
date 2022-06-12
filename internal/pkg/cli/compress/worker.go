package compress

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/tarampampam/tinifier/v3/internal/pkg/keys"
	"github.com/tarampampam/tinifier/v3/internal/pkg/pool"
	"github.com/tarampampam/tinifier/v3/internal/pkg/retry"
	"github.com/tarampampam/tinifier/v3/internal/pkg/validate"
	"github.com/tarampampam/tinifier/v3/pkg/tinypng"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var errNoAvailableAPIKey = errors.New("no one API key can be used")

type Worker struct {
	log    *zap.Logger
	keeper *keys.Keeper

	retryAttempts uint
	retryInterval time.Duration

	tinyHTTPClient interface { // make worker testable
		Do(*http.Request) (*http.Response, error)
	}
}

func newWorker(log *zap.Logger, keeper *keys.Keeper, retryAttempts uint, retryInterval time.Duration) *Worker {
	return &Worker{
		log:    log,
		keeper: keeper,

		retryAttempts: retryAttempts,
		retryInterval: retryInterval,

		tinyHTTPClient: new(http.Client),
	}
}

func (w *Worker) PreTaskRun(task pool.Task) {
	w.log.Info(fmt.Sprintf("[%d of %d] Compressing file \"%s\"", task.TaskNumber, task.TasksCount, task.FilePath))
}

func (w *Worker) Upload(ctx context.Context, filePath string) (string, pool.FileInfo, error) {
	const uploadTimeout = time.Minute * 4

	fInfo, err := newFileInfo(filePath)
	if err != nil {
		return "", nil, err
	}

	var (
		tiny = tinypng.NewClient("", tinypng.WithContext(ctx), tinypng.WithHTTPClient(w.tinyHTTPClient))
		info *tinypng.CompressionResult
	)

	if err := retry.Do(
		func(attemptNum uint) error {
			key, err := w.keeper.Get()
			if err != nil {
				return errNoAvailableAPIKey
			}

			tiny.SetAPIKey(key)

			file, openErr := w.openSourceFile(filePath)
			if openErr != nil {
				return openErr
			}
			defer func() { _ = file.Close() }()

			compResponse, uplErr := tiny.CompressImage(file, uploadTimeout)
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

			info = compResponse

			return nil
		},
		retry.WithContext(ctx),
		retry.WithAttempts(w.retryAttempts),
		retry.WithDelay(w.retryInterval),
		retry.WithLastErrorReturning(),
		retry.WithRetryStoppingErrors(errNoAvailableAPIKey),
	); err != nil {
		return "", nil, err
	}

	return info.Output.URL, fInfo, nil
}

func (w *Worker) openSourceFile(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}

	var keepFileOpened bool

	defer func() {
		if !keepFileOpened {
			_ = file.Close()
		}
	}()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if !stat.Mode().IsRegular() {
		return nil, errors.New("is not regular file")
	}

	if ok, valErr := validate.IsImage(file); !ok || valErr != nil {
		return nil, errors.New("wrong image file")
	}

	if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
		return nil, seekErr
	}

	keepFileOpened = true

	return file, nil
}

func (w *Worker) Download(ctx context.Context, url string, toFilePath string) (pool.FileInfo, error) {
	const downloadTimeout = time.Minute * 2

	var tiny = tinypng.NewClient("", tinypng.WithContext(ctx), tinypng.WithHTTPClient(w.tinyHTTPClient))

	if err := retry.Do(
		func(attemptNum uint) error {
			key, err := w.keeper.Get()
			if err != nil {
				return errNoAvailableAPIKey
			}

			tiny.SetAPIKey(key)

			file, err := os.OpenFile(toFilePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644) //nolint:gomnd
			if err != nil {
				return err
			}
			defer func() { _ = file.Close() }()

			if _, dlErr := tiny.DownloadImage(url, file, downloadTimeout); dlErr != nil {
				w.log.Warn("Compressed image downloading failed",
					zap.Error(err),
					zap.String("file", toFilePath),
					zap.Uint("attempt", attemptNum),
					zap.String("key", key),
				)

				if errors.Is(dlErr, tinypng.ErrTooManyRequests) || errors.Is(dlErr, tinypng.ErrUnauthorized) {
					w.keeper.Remove(key)
				}

				return dlErr
			}

			return nil
		},
		retry.WithContext(ctx),
		retry.WithAttempts(w.retryAttempts),
		retry.WithDelay(w.retryInterval),
		retry.WithLastErrorReturning(),
		retry.WithRetryStoppingErrors(errNoAvailableAPIKey),
	); err != nil {
		return nil, err
	}

	fInfo, err := newFileInfo(toFilePath)
	if err != nil {
		return nil, err
	}

	return fInfo, nil
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

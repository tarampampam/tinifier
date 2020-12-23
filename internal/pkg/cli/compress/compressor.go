package compress

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/tarampampam/tinifier/internal/pkg/pipeline"
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

type compressor struct {
	log    *zap.Logger
	keeper keysKeeper

	maxRetries    uint
	retryInterval time.Duration
}

// newCompressor creates new images compressor, that uses tinypng.com.
func newCompressor(log *zap.Logger, keeper keysKeeper, maxRetries uint, retryInterval time.Duration) compressor {
	return compressor{
		log:           log,
		keeper:        keeper,
		maxRetries:    maxRetries,
		retryInterval: retryInterval,
	}
}

// Compress reads file from passed task, compress them using tinypng client, and overwrite original file with
// compressed image content.
func (c compressor) Compress(ctx context.Context, t pipeline.Task) (*pipeline.TaskResult, error) { //nolint:funlen
	const (
		tinyDefaultTimeout  = time.Minute * 5
		tinyUploadTimeout   = time.Minute * 3 // TODO do not hardcode timeout, calculate it
		tinyDownloadTimeout = time.Minute * 2 // TODO do not hardcode timeout, calculate it
	)

	tiny := tinypng.NewClient("", tinypng.WithContext(ctx), tinypng.WithDefaultTimeout(tinyDefaultTimeout))

	var (
		info        *tinypng.CompressionResult
		srcFileMode os.FileMode
		apiKey      string
	)

	// STEP 1 - upload image to the tinypng.com side
	if err := retry.Do(func(attemptNum uint) (err error) {
		if apiKey, err = c.keeper.Get(); err != nil {
			return errors.New("no one key can be used")
		}
		tiny.SetAPIKey(apiKey)

		srcFile, stat, err := c.openSourceFile(t.FilePath)
		if err != nil {
			return err
		}
		defer func() { _ = srcFile.Close() }()

		srcFileMode = stat.Mode()

		if info, err = tiny.CompressImage(srcFile, tinyUploadTimeout); err != nil {
			c.log.Warn("File uploading failed",
				zap.Error(err),
				zap.String("file", t.FilePath),
				zap.Uint("attempt", attemptNum),
				zap.String("key", apiKey),
			)

			if errors.Is(err, tinypng.ErrTooManyRequests) || errors.Is(err, tinypng.ErrUnauthorized) {
				c.keeper.Remove(apiKey)
			}

			if _, seekErr := srcFile.Seek(0, io.SeekStart); seekErr != nil {
				return seekErr
			}

			return err
		}

		return nil // compressed successful
	}, retry.WithContext(ctx), retry.WithAttempts(c.maxRetries), retry.WithDelay(c.retryInterval)); err != nil {
		return nil, errors.New("image uploading failed")
	}

	// STEP 2 - download compressed image into temporary file
	tempFilePath := t.FilePath + ".tiny"

	defer func() { _ = os.Remove(tempFilePath) }() // remove temporary file anyway

	if err := retry.Do(func(attemptNum uint) error {
		tempFile, err := c.createTemporaryFile(tempFilePath, srcFileMode)
		if err != nil {
			return err
		}
		defer func() { _ = tempFile.Close() }()

		// TODO call `c.keeper.Get()` - invalid key does not allows to download the image

		if _, err = tiny.DownloadImage(info.Output.URL, tempFile, tinyDownloadTimeout); err != nil {
			c.log.Warn("Compressed file downloading failed",
				zap.Error(err),
				zap.String("file", t.FilePath),
				zap.String("temp file", tempFilePath),
				zap.Uint("attempt", attemptNum),
				zap.String("key", apiKey),
			)

			return err
		}

		return nil
	}, retry.WithContext(ctx), retry.WithAttempts(c.maxRetries), retry.WithDelay(c.retryInterval)); err != nil {
		return nil, errors.New("compressed image downloading failed")
	}

	// STEP 3 - replace original file with temporary
	if err := c.copyFileContent(tempFilePath, t.FilePath); err != nil {
		return nil, err
	}

	return &pipeline.TaskResult{
		FileType:       info.Output.Type,
		FilePath:       t.FilePath,
		OriginalSize:   info.Input.Size,
		CompressedSize: info.Output.Size,
		UsedQuota:      info.CompressionCount,
	}, nil
}

func (c compressor) openSourceFile(path string) (*os.File, os.FileInfo, error) {
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

	if ok, err := validator.IsImage(file); !ok || err != nil {
		return nil, nil, errors.New("wrong image file")
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, nil, err
	}

	keepFileOpened = true

	return file, stat, nil
}

func (c compressor) createTemporaryFile(path string, mode os.FileMode) (io.WriteCloser, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, mode)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (c compressor) copyFileContent(fromFilePath, toFilePath string) error {
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

	_, err = io.Copy(toFile, fromFile)

	return err
}

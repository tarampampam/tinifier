package compress

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/tarampampam/tinifier/internal/pkg/pipeline"
	"github.com/tarampampam/tinifier/pkg/tinypng"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	httpRequestTimeout    = time.Second * 90
	maxCompressionRetries = 600
	compressionRetryAfter = time.Millisecond * 500

	// Smallest possible PNG image size is 67 bytes <https://garethrees.org/2007/11/14/pngcrush/>
	//                   JPG image - 125 bytes <https://stackoverflow.com/a/24124454/2252921>
	minimalImageFileSize = 67 // bytes
)

type compressor struct {
	ctx    context.Context
	log    *zap.Logger
	keeper tinyKeysKeeper
}

type tinyKeysKeeper interface {
	Get() (string, error)
	ReportKeyError(key string, delta int) error
}

// newCompressor creates new tinypng images compressor.
func newCompressor(ctx context.Context, log *zap.Logger, keeper tinyKeysKeeper) compressor {
	return compressor{ctx: ctx, log: log, keeper: keeper}
}

// Compress reads file from passed task, compress them using tinypng client, and overwrite original file with
// compressed image content.
func (c compressor) Compress(t pipeline.Task) (*pipeline.TaskResult, error) {
	source, stat, err := c.readFile(t.FilePath)
	if err != nil {
		return nil, err
	}

	if len(source) < minimalImageFileSize {
		return nil, errors.New("original file size is too small")
	}

	tiny := tinypng.NewClient(tinypng.ClientConfig{RequestTimeout: httpRequestTimeout})

	var (
		resp          *tinypng.Result
		fallbackBreak uint
	)

retryLoop:
	for {
		if fallbackBreak++; fallbackBreak > maxCompressionRetries {
			return nil, errors.New("too many retries (REPORT ABOUT THIS ERROR TO DEVELOPERS)")
		}

		apiKey, err := c.keeper.Get()
		if err != nil {
			return nil, errors.Wrap(err, "no one key can be used")
		}

		tiny.SetAPIKey(apiKey)

		resp, err = tiny.Compress(c.ctx, bytes.NewBuffer(source))
		if err == nil {
			break retryLoop // compressed successful
		}

		// TODO handle error "InputMissing (Input file is empty)" - break this loop

		if err == tinypng.ErrTooManyRequests || err == tinypng.ErrUnauthorized {
			if reportingErr := c.keeper.ReportKeyError(apiKey, 1); reportingErr != nil {
				return nil, reportingErr
			}
		}

		c.log.Warn("Remote error occurred, retrying",
			zap.Error(err),
			zap.String("file", t.FilePath),
			zap.String("key", apiKey),
		)

		select {
		case <-c.ctx.Done():
			return nil, errors.New("compressing canceled")

		case <-time.After(compressionRetryAfter):
		}
	}

	if err := c.writeFile(t.FilePath, resp.Compressed, stat.Mode()); err != nil {
		return nil, err
	}

	return &pipeline.TaskResult{
		FileType:       resp.Output.Type,
		FilePath:       t.FilePath,
		OriginalSize:   resp.Input.Size,
		CompressedSize: resp.Output.Size,
		UsedQuota:      resp.CompressionCount,
	}, nil
}

func (c compressor) readFile(filePath string) ([]byte, os.FileInfo, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0) // open file for reading
	if err != nil {
		return nil, nil, err
	}

	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, nil, err
	}

	return buf, stat, nil
}

func (c compressor) writeFile(filePath string, content []byte, mode os.FileMode) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, mode) // open file for writing
	if err != nil {
		return err
	}

	defer file.Close()

	_, err = io.Copy(file, bytes.NewReader(content))
	if err != nil {
		return err
	}

	return nil
}

package compress

import (
	"context"
	"io"
	"os"

	"github.com/tarampampam/tinifier/internal/pkg/pipeline"
	"github.com/tarampampam/tinifier/pkg/tinypng"
)

type compressor struct {
	ctx  context.Context
	tiny *tinypng.Client
}

// newCompressor creates new tinypng images compressor.
func newCompressor(ctx context.Context, tiny *tinypng.Client) compressor {
	return compressor{ctx: ctx, tiny: tiny}
}

// Compress reads file from passed task, compress them using tinypng client, and overwrite original file with
// compressed image content.
func (c compressor) Compress(t pipeline.Task) (*pipeline.TaskResult, error) {
	fileRead, err := os.OpenFile(t.FilePath, os.O_RDONLY, 0) // open file for reading
	if err != nil {
		return nil, err
	}

	stat, err := fileRead.Stat()
	if err != nil {
		fileRead.Close() // do not forget to close file

		return nil, err
	}

	resp, err := c.tiny.Compress(c.ctx, fileRead)

	fileRead.Close() // file was compressed (successful or not), and must be closed

	if err != nil {
		return nil, err
	}

	defer resp.Compressed.Close()

	fileWrite, err := os.OpenFile(t.FilePath, os.O_WRONLY|os.O_TRUNC, stat.Mode()) // open file for writing
	if err != nil {
		return nil, err
	}

	defer fileWrite.Close()

	_, err = io.Copy(fileWrite, resp.Compressed)
	if err != nil {
		return nil, err
	}

	return &pipeline.TaskResult{
		FileType:       resp.Output.Type,
		FilePath:       t.FilePath,
		OriginalSize:   resp.Input.Size,
		CompressedSize: resp.Output.Size,
	}, nil
}

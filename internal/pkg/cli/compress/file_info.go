package compress

import (
	"errors"
	"io"
	"net/http"
	"os"
)

type fileInfo struct {
	size uint64
	t    string
}

func (i *fileInfo) Size() uint64 { return i.size }
func (i *fileInfo) Type() string { return i.t }

func newFileInfo(path string) (*fileInfo, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}

	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 32) // 32 bytes are enough for images

	if _, err := io.ReadFull(file, buf); err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, err
	}

	return &fileInfo{size: uint64(stat.Size()), t: http.DetectContentType(buf)}, nil
}

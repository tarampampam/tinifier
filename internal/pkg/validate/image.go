package validate

import (
	"errors"
	"io"
	"net/http"
	"strings"
)

// IsImage checks for passed content is image or not.
// Do not forget to reset the source (offset will be changed after this function calling).
func IsImage(src io.Reader) (bool, error) {
	buf := make([]byte, 32) // 32 bytes are enough for "first bytes" checking

	if _, err := io.ReadFull(src, buf); err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return false, err
	}

	return strings.HasPrefix(http.DetectContentType(buf), "image/"), nil
}

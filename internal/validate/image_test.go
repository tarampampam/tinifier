package validate_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"gh.tarampamp.am/tinifier/v4/internal/validate"
)

type brokenReader struct{}

func (r brokenReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("fake error")
}

func TestIsImage(t *testing.T) {
	fromFile := func(path string) io.Reader {
		t.Helper()

		data, err := os.ReadFile(path)
		assert.NoError(t, err)

		return bytes.NewReader(data)
	}

	var cases = []struct {
		name       string
		giveReader func() io.Reader
		wantResult bool
		wantErr    bool
	}{
		{
			name:       "empty reader",
			giveReader: func() io.Reader { return bytes.NewReader([]byte{}) },
			wantResult: false,
			wantErr:    false,
		},
		{
			name:       "fake string",
			giveReader: func() io.Reader { return bytes.NewReader([]byte("foo bar")) },
			wantResult: false,
			wantErr:    false,
		},
		{
			name:       "broken reader",
			giveReader: func() io.Reader { return brokenReader{} },
			wantResult: false,
			wantErr:    true,
		},
		{
			name:       "zip archive",
			giveReader: func() io.Reader { return fromFile("./testdata/zipped.zip") },
			wantResult: false,
			wantErr:    false,
		},
		{
			name:       "bmp file",
			giveReader: func() io.Reader { return fromFile("./testdata/image.bmp") },
			wantResult: true,
			wantErr:    false,
		},
		{
			name:       "gif file",
			giveReader: func() io.Reader { return fromFile("./testdata/image.gif") },
			wantResult: true,
			wantErr:    false,
		},
		{
			name:       "ico file",
			giveReader: func() io.Reader { return fromFile("./testdata/image.ico") },
			wantResult: true,
			wantErr:    false,
		},
		{
			name:       "jpg file",
			giveReader: func() io.Reader { return fromFile("./testdata/image.jpg") },
			wantResult: true,
			wantErr:    false,
		},
		{
			name:       "png file",
			giveReader: func() io.Reader { return fromFile("./testdata/image.png") },
			wantResult: true,
			wantErr:    false,
		},
		{
			name:       "webp file",
			giveReader: func() io.Reader { return fromFile("./testdata/image.webp") },
			wantResult: true,
			wantErr:    false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := validate.IsImage(tt.giveReader())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantResult, res)
		})
	}
}

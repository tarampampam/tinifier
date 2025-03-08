package validate_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"gh.tarampamp.am/tinifier/v5/internal/validate"
)

type brokenReader struct{}

func (r brokenReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("fake error")
}

func TestIsImage(t *testing.T) {
	fromFile := func(path string) io.Reader {
		t.Helper()

		data, err := os.ReadFile(path)
		assertNoError(t, err)

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
				assertError(t, err)
			} else {
				assertNoError(t, err)
			}

			assertEqual(t, tt.wantResult, res)
		})
	}
}

func assertEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()

	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func assertError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

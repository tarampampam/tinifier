package tinypng_test

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/tinifier/v4/pkg/tinypng"
)

func ExampleClient_Compress_and_Download() {
	c := tinypng.NewClient("YOUR-API-KEY")

	srcFile, err := os.OpenFile("/tmp/image.png", os.O_RDONLY, 0)
	if err != nil {
		panic(err)
	}
	defer srcFile.Close()

	info, err := c.Compress(context.TODO(), srcFile)
	if err != nil {
		panic(err)
	}

	destFile, err := os.OpenFile("/tmp/image_compressed.png", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	defer destFile.Close()

	if err = info.Download(context.TODO(), destFile); err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", info)
}

type httpClientFunc func(*http.Request) (*http.Response, error)

func (f httpClientFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func authHeaderValue(t *testing.T, apiKey string) string {
	t.Helper()

	return "Basic " + base64.StdEncoding.EncodeToString([]byte("api:"+apiKey))
}

func TestClient_UsedQuota_Success(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "https://api.tinify.com/shrink", req.URL.String())
		require.Equal(t, authHeaderValue(t, "foo-key"), req.Header.Get("Authorization"))

		return &http.Response{
			Header: http.Header{
				"Compression-Count": {"123454321", "any", "values"},
			},
			Body: io.NopCloser(bytes.NewReader([]byte{})),
		}, nil
	}

	c := tinypng.NewClient("wrong key", tinypng.WithHTTPClient(httpMock))
	c.SetAPIKey("foo-key")

	count, err := c.UsedQuota(context.TODO())

	assert.Equal(t, uint64(123454321), count)
	assert.NoError(t, err)
}

func TestClient_UsedQuota_WrongHeaderValue(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header: http.Header{
				"Compression-Count": {"foo bar"}, // <-- wrong value
			},
			Body: io.NopCloser(bytes.NewReader([]byte{})),
		}, nil
	}

	count, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).UsedQuota(context.TODO())

	assert.Equal(t, uint64(0), count)
	require.Error(t, err)
	require.ErrorContains(t, err, "wrong HTTP header")
}

func TestClient_UsedQuota_MissingHeader(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header: http.Header{
				// "Compression-Count": {"123454321", "any", "values"}, // <-- nothing is here
			},
			Body: io.NopCloser(bytes.NewReader([]byte{})),
		}, nil
	}

	count, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).UsedQuota(context.TODO())

	assert.Equal(t, uint64(0), count)
	assert.Error(t, err)
	assert.Regexp(t, `(?is)tinypng\.com.+header.+not found`, err.Error())
}

func TestClient_UsedQuota_HttpClientError(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("foo bar")
	}

	count, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).UsedQuota(context.TODO())

	assert.Equal(t, uint64(0), count)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "foo bar")
}

func TestClient_UsedQuota_Unauthorized(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader([]byte{})),
			StatusCode: http.StatusUnauthorized, // <-- important
		}, nil
	}

	count, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).UsedQuota(context.TODO())

	assert.Equal(t, uint64(0), count)
	assert.ErrorIs(t, err, tinypng.ErrUnauthorized)
}

var (
	//go:embed testdata/image.png
	srcImage []byte

	//go:embed testdata/image_compressed.png
	compressedImage []byte
)

func TestClient_Compress_Successful(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	var srcBuff = bytes.NewBuffer(srcImage)

	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "https://api.tinify.com/shrink", req.URL.String())
		require.Equal(t, authHeaderValue(t, "bar-key"), req.Header.Get("Authorization"))

		body, _ := io.ReadAll(req.Body)
		assert.Equal(t, srcImage, body)

		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     http.Header{"Compression-Count": {"123454321"}},
			Body: io.NopCloser(bytes.NewReader([]byte(`{
				"input":{
					"size":4633,
					"type":"image/png"
				},
				"output":{
					"size":1636,
					"type":"image/png",
					"width":123,
					"height":321,
					"ratio":0.3531,
					"url":"https://api.tinify.com/output/someRandomResultImageHash"
				}
			}`))),
		}, nil
	}

	info, err := tinypng.NewClient("bar-key", tinypng.WithHTTPClient(httpMock)).Compress(ctx, srcBuff)

	require.NoError(t, err)
	assert.Equal(t, uint64(1636), info.Size())
	assert.Equal(t, "image/png", info.Type())

	w, h := info.Dimensions()
	assert.Equal(t, uint32(123), w)
	assert.Equal(t, uint32(321), h)
	assert.Equal(t, "https://api.tinify.com/output/someRandomResultImageHash", info.URL())

	assert.Equal(t, uint64(123454321), info.UsedQuota())
}

func TestClient_Compress_WrongJsonResponse(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     http.Header{"Compression-Count": {"123454321"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"broken json']`))),
		}, nil
	}

	info, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).
		Compress(context.TODO(), bytes.NewBuffer(srcImage))

	assert.Nil(t, info)
	assert.Error(t, err)
	assert.Regexp(t, `(?is)tinypng\.com.+decoding.+failed`, err.Error())
}

func TestClient_Compress_Unauthorized(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader([]byte{})),
			StatusCode: http.StatusUnauthorized, // <-- important
		}, nil
	}

	info, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).
		Compress(context.TODO(), bytes.NewBuffer(srcImage))

	assert.Nil(t, info)
	assert.ErrorIs(t, err, tinypng.ErrUnauthorized)
}

func TestClient_Compress_TooManyRequests(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader([]byte{})),
			StatusCode: http.StatusTooManyRequests, // <-- important
		}, nil
	}

	info, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).
		Compress(context.TODO(), bytes.NewBuffer(srcImage))

	assert.Nil(t, info)
	assert.ErrorIs(t, err, tinypng.ErrTooManyRequests)
}

func TestClient_Compress_BadRequests(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader([]byte{})),
			StatusCode: http.StatusBadRequest, // <-- important
		}, nil
	}

	info, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).
		Compress(context.TODO(), bytes.NewBuffer(srcImage))

	assert.Nil(t, info)
	assert.ErrorIs(t, err, tinypng.ErrBadRequest)
}

func TestClient_Compress_HTTPErrorAbove599(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader([]byte{})),
			StatusCode: 600, // <-- important
		}, nil
	}

	info, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).
		Compress(context.TODO(), bytes.NewBuffer(srcImage))

	assert.Nil(t, info)
	assert.Error(t, err)
	assert.Regexp(t, `(?is)tinypng\.com.+unexpected.+code`, err.Error())
}

func TestClient_Compress_4xxError(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{"Compression-Count": {"123"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"Foo","message":"bar baz."}`))),
			StatusCode: http.StatusTeapot, // <-- can be any (instead 401, 429 and 400)
		}, nil
	}

	info, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).
		Compress(context.TODO(), bytes.NewBuffer(srcImage))

	assert.Nil(t, info)
	assert.Error(t, err)
	assert.Regexp(t, `(?is)tinypng\.com.+Foo \(bar baz\)$`, err.Error())
}

func TestClient_Compress_4xxErrorWithWrongJson(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{"Compression-Count": {"123"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"broken json']`))),
			StatusCode: http.StatusLocked, // <-- can be any (instead 401, 429 and 400)
		}, nil
	}

	info, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).
		Compress(context.TODO(), bytes.NewBuffer(srcImage))

	assert.Nil(t, info)
	assert.Error(t, err)
	assert.Regexp(t, `(?is)tinypng\.com.+decoding.+failed`, err.Error())
}

func TestClient_Compress_HttpClientError(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("foo bar")
	}

	info, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).
		Compress(context.TODO(), bytes.NewBuffer(srcImage))

	assert.Nil(t, info)
	assert.Error(t, err)
	assert.Equal(t, "foo bar", err.Error())
}

func TestClient_Compressed_Download_Success(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case "https://api.tinify.com/shrink": //nolint:goconst
			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     http.Header{"Compression-Count": {"123454321"}},
				Body: io.NopCloser(bytes.NewReader([]byte(`{
				"input":{
					"size":4633,
					"type":"image/png"
				},
				"output":{
					"size":1636,
					"type":"image/png",
					"width":123,
					"height":321,
					"ratio":0.3531,
					"url":"https://api.tinify.com/output/someRandomResultImageHash"
				}
			}`))),
			}, nil

		case "https://api.tinify.com/output/someRandomResultImageHash":
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(compressedImage)),
			}, nil

		default:
			return nil, errors.New("unexpected request")
		}
	}

	info, err := tinypng.NewClient("bar-key", tinypng.WithHTTPClient(httpMock)).Compress(ctx, bytes.NewBuffer(srcImage))
	require.NoError(t, err)

	out := bytes.NewBuffer(nil)
	err = info.Download(ctx, out)

	assert.NoError(t, err)
	assert.NotEmpty(t, out)
	assert.Equal(t, compressedImage, out.Bytes())
}

func TestClient_Compressed_Download_Unauthorized(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case "https://api.tinify.com/shrink":
			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     http.Header{"Compression-Count": {"123454321"}},
				Body: io.NopCloser(bytes.NewReader([]byte(`{
				"output":{
					"url":"https://api.tinify.com/output/someRandomResultImageHash123"
				}
			}`))),
			}, nil

		case "https://api.tinify.com/output/someRandomResultImageHash123":
			return &http.Response{
				StatusCode: http.StatusUnauthorized, // <-- important
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
			}, nil

		default:
			return nil, errors.New("unexpected request")
		}
	}

	info, err := tinypng.NewClient("bar-key", tinypng.WithHTTPClient(httpMock)).Compress(ctx, bytes.NewBuffer(srcImage))
	require.NoError(t, err)

	out := bytes.NewBuffer(nil)
	err = info.Download(ctx, out)

	assert.Empty(t, out)
	assert.ErrorIs(t, err, tinypng.ErrUnauthorized)
}

func TestClient_Compressed_Download_TooManyRequests(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case "https://api.tinify.com/shrink":
			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     http.Header{"Compression-Count": {"123454321"}},
				Body: io.NopCloser(bytes.NewReader([]byte(`{
				"output":{
					"url":"https://api.tinify.com/output/someRandomResultImageHash321"
				}
			}`))),
			}, nil

		case "https://api.tinify.com/output/someRandomResultImageHash321":
			return &http.Response{
				StatusCode: http.StatusTooManyRequests, // <-- important
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
			}, nil

		default:
			return nil, errors.New("unexpected request")
		}
	}

	info, err := tinypng.NewClient("bar-key", tinypng.WithHTTPClient(httpMock)).Compress(ctx, bytes.NewBuffer(srcImage))
	require.NoError(t, err)

	out := bytes.NewBuffer(nil)
	err = info.Download(ctx, out)

	assert.Empty(t, out)
	assert.ErrorIs(t, err, tinypng.ErrTooManyRequests)
}

func TestClient_DownloadImage4xxError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case "https://api.tinify.com/shrink":
			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     http.Header{"Compression-Count": {"123454321"}},
				Body: io.NopCloser(bytes.NewReader([]byte(`{
				"output":{
					"url":"https://api.tinify.com/output/someRandomResultImageHash111"
				}
			}`))),
			}, nil

		case "https://api.tinify.com/output/someRandomResultImageHash111":
			return &http.Response{
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"Foo","message":"bar baz."}`))),
				StatusCode: http.StatusTeapot, // <-- can be any 4xx error
			}, nil

		default:
			return nil, errors.New("unexpected request")
		}
	}

	info, err := tinypng.NewClient("bar-key", tinypng.WithHTTPClient(httpMock)).Compress(ctx, bytes.NewBuffer(srcImage))
	require.NoError(t, err)

	out := bytes.NewBuffer(nil)
	err = info.Download(ctx, out)

	assert.Empty(t, out)
	assert.Regexp(t, `(?is)tinypng\.com.+Foo \(bar baz\)$`, err.Error())
}

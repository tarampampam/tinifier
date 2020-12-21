package tinypng

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ExampleNewClient() {
	myContext := context.TODO()

	NewClient("YOUR-API-KEY", WithContext(myContext), WithDefaultTimeout(time.Second*60))
}

func ExampleClient_SetAPIKey() {
	c := NewClient("WRONG-KEY")

	c.SetAPIKey("CORRECT-KEY")
}

func ExampleClient_Compress() {
	c := NewClient("YOUR-API-KEY")

	srcFile, err := os.OpenFile("/tmp/image.png", os.O_RDONLY, 0)
	if err != nil {
		panic(err)
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile("/tmp/image_compressed.png", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	defer destFile.Close()

	info, err := c.Compress(srcFile, destFile, time.Second*60, time.Second*30)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", info)
}

func ExampleClient_CompressImage() {
	c := NewClient("YOUR-API-KEY")

	srcFile, err := os.OpenFile("/tmp/image.png", os.O_RDONLY, 0)
	if err != nil {
		panic(err)
	}
	defer srcFile.Close()

	info, err := c.CompressImage(srcFile, time.Second*60)
	if err != nil {
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

func TestClient_SetAPIKey(t *testing.T) {
	c := NewClient("")

	c.SetAPIKey("YOUR-API-KEY")

	assert.Equal(t, "YOUR-API-KEY", c.apiKey)
}

func TestClient_CompressionCountSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://api.tinify.com/shrink", req.URL.String())
		assert.Equal(t, authHeaderValue(t, "foo-key"), req.Header.Get("Authorization"))

		return &http.Response{
			Header: http.Header{
				"Compression-Count": {"123454321", "any", "values"},
			},
			Body: ioutil.NopCloser(bytes.NewReader([]byte{})),
		}, nil
	}

	count, err := NewClient("foo-key", WithHTTPClient(httpMock), WithContext(ctx)).CompressionCount(time.Second)

	assert.Equal(t, uint64(123454321), count)
	assert.NoError(t, err)
	assert.NoError(t, ctx.Err())
}

func TestClient_CompressionCountWrongHeaderValue(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header: http.Header{
				"Compression-Count": {"foo bar"}, // <-- wrong value
			},
			Body: ioutil.NopCloser(bytes.NewReader([]byte{})),
		}, nil
	}

	count, err := NewClient("", WithHTTPClient(httpMock)).CompressionCount()

	assert.Equal(t, uint64(0), count)
	assert.Error(t, err)
	assert.Regexp(t, `(?is)tinypng\.com.+wrong.+value`, err.Error())
}

func TestClient_CompressionCountMissingHeader(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header: http.Header{
				// "Compression-Count": {"123454321", "any", "values"}, // <-- nothing is here
			},
			Body: ioutil.NopCloser(bytes.NewReader([]byte{})),
		}, nil
	}

	count, err := NewClient("", WithHTTPClient(httpMock)).CompressionCount()

	assert.Equal(t, uint64(0), count)
	assert.Error(t, err)
	assert.Regexp(t, `(?is)tinypng\.com.+header.+not found`, err.Error())
}

func TestClient_CompressionCountHttpClientError(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("foo bar")
	}

	count, err := NewClient("", WithHTTPClient(httpMock)).CompressionCount()

	assert.Equal(t, uint64(0), count)
	assert.Error(t, err)
	assert.Equal(t, "foo bar", err.Error())
}

func TestClient_CompressionCountUnauthorized(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{},
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			StatusCode: http.StatusUnauthorized, // <-- important
		}, nil
	}

	count, err := NewClient("", WithHTTPClient(httpMock)).CompressionCount()

	assert.Equal(t, uint64(0), count)
	assert.Equal(t, ErrUnauthorized, err)
}

func TestClient_CompressImageSuccessful(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	file, err := os.OpenFile("./testdata/image.png", os.O_RDONLY, 0)
	assert.NoError(t, err)

	defer file.Close()

	fileBody, err := ioutil.ReadAll(file) // read file for future asserting
	assert.NoError(t, err)
	_, err = file.Seek(0, 0) // reset file "cursor"
	assert.NoError(t, err)

	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://api.tinify.com/shrink", req.URL.String())
		assert.Equal(t, authHeaderValue(t, "bar-key"), req.Header.Get("Authorization"))

		body, _ := ioutil.ReadAll(req.Body)
		assert.Equal(t, fileBody, body)

		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     http.Header{"Compression-Count": {"123454321"}},
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
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

	info, err := NewClient("bar-key", WithHTTPClient(httpMock), WithContext(ctx)).CompressImage(file, time.Second)

	assert.NoError(t, err)
	assert.Equal(t, uint64(4633), info.Input.Size)
	assert.Equal(t, "image/png", info.Input.Type)
	assert.Equal(t, uint64(1636), info.Output.Size)
	assert.Equal(t, "image/png", info.Output.Type)
	assert.Equal(t, uint64(123), info.Output.Width)
	assert.Equal(t, uint64(321), info.Output.Height)
	assert.Equal(t, float32(0.3531), info.Output.Ratio)
	assert.Equal(t, "https://api.tinify.com/output/someRandomResultImageHash", info.Output.URL)

	assert.Equal(t, uint64(123454321), info.CompressionCount)
}

func TestClient_CompressImageWrongJsonResponse(t *testing.T) {
	file, err := os.OpenFile("./testdata/image.png", os.O_RDONLY, 0)
	assert.NoError(t, err)

	defer file.Close()

	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     http.Header{"Compression-Count": {"123454321"}},
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(`{"broken json']`))),
		}, nil
	}

	info, err := NewClient("", WithHTTPClient(httpMock)).CompressImage(file)

	assert.Nil(t, info)
	assert.Error(t, err)
	assert.Regexp(t, `(?is)tinypng\.com.+decoding.+failed`, err.Error())
}

func TestClient_CompressImageUnauthorized(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{},
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			StatusCode: http.StatusUnauthorized, // <-- important
		}, nil
	}

	info, err := NewClient("", WithHTTPClient(httpMock)).CompressImage(bytes.NewReader([]byte{}))

	assert.Nil(t, info)
	assert.Equal(t, ErrUnauthorized, err)
}

func TestClient_CompressImageTooManyRequests(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{},
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			StatusCode: http.StatusTooManyRequests, // <-- important
		}, nil
	}

	info, err := NewClient("", WithHTTPClient(httpMock)).CompressImage(bytes.NewReader([]byte{}))

	assert.Nil(t, info)
	assert.Equal(t, ErrTooManyRequests, err)
}

func TestClient_CompressImageBadRequests(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{},
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			StatusCode: http.StatusBadRequest, // <-- important
		}, nil
	}

	info, err := NewClient("", WithHTTPClient(httpMock)).CompressImage(bytes.NewReader([]byte{}))

	assert.Nil(t, info)
	assert.Equal(t, ErrBadRequest, err)
}

func TestClient_CompressImageHTTPErrorAbove599(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{},
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			StatusCode: 600, // <-- important
		}, nil
	}

	info, err := NewClient("", WithHTTPClient(httpMock)).CompressImage(bytes.NewReader([]byte{}))

	assert.Nil(t, info)
	assert.Error(t, err)
	assert.Regexp(t, `(?is)tinypng\.com.+unexpected.+code`, err.Error())
}

func TestClient_CompressImage4xxError(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{"Compression-Count": {"123"}},
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(`{"error":"Foo","message":"bar baz."}`))),
			StatusCode: http.StatusTeapot, // <-- can be any (instead 401, 429 and 400)
		}, nil
	}

	info, err := NewClient("", WithHTTPClient(httpMock)).CompressImage(bytes.NewReader([]byte{}))

	assert.Nil(t, info)
	assert.Error(t, err)
	assert.Regexp(t, `(?is)tinypng\.com.+Foo \(bar baz\)$`, err.Error())
}

func TestClient_CompressImage4xxErrorWithWrongJson(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{"Compression-Count": {"123"}},
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(`{"error":"broken json']`))),
			StatusCode: http.StatusLocked, // <-- can be any (instead 401, 429 and 400)
		}, nil
	}

	info, err := NewClient("", WithHTTPClient(httpMock)).CompressImage(bytes.NewReader([]byte{}))

	assert.Nil(t, info)
	assert.Error(t, err)
	assert.Regexp(t, `(?is)tinypng\.com.+decoding.+failed`, err.Error())
}

func TestClient_CompressImageHttpClientError(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("foo bar")
	}

	info, err := NewClient("", WithHTTPClient(httpMock)).CompressImage(bytes.NewReader([]byte{}))

	assert.Nil(t, info)
	assert.Error(t, err)
	assert.Equal(t, "foo bar", err.Error())
}

func TestClient_DownloadImageSuccessful(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	file, err := os.OpenFile("./testdata/image_compressed.png", os.O_RDONLY, 0)
	assert.NoError(t, err)

	defer file.Close()

	fileBody, err := ioutil.ReadAll(file) // read file for future asserting
	assert.NoError(t, err)
	_, err = file.Seek(0, 0) // reset file "cursor"
	assert.NoError(t, err)

	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "https://api.tinify.com/output/someRandomResultImageHash", req.URL.String())
		assert.Equal(t, authHeaderValue(t, "baz-key"), req.Header.Get("Authorization"))

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       file,
		}, nil
	}

	dest := bytes.NewBuffer([]byte{})

	written, err := NewClient("baz-key", WithHTTPClient(httpMock), WithContext(ctx)).
		DownloadImage("https://api.tinify.com/output/someRandomResultImageHash", dest, time.Second)

	assert.NoError(t, err)
	assert.Equal(t, fileBody, dest.Bytes())
	assert.Equal(t, written, int64(len(fileBody)))
}

func TestClient_DownloadImageUnauthorized(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "https://example.com/foo", req.URL.String())

		return &http.Response{
			Header:     http.Header{},
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			StatusCode: http.StatusUnauthorized, // <-- important
		}, nil
	}

	written, err := NewClient("", WithHTTPClient(httpMock)).
		DownloadImage("https://example.com/foo", bytes.NewBuffer([]byte{}))

	assert.Zero(t, written)
	assert.Equal(t, ErrUnauthorized, err)
}

func TestClient_DownloadImage4xxError(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{},
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(`{"error":"Foo","message":"bar baz."}`))),
			StatusCode: http.StatusTeapot, // <-- can be any (instead 401)
		}, nil
	}

	written, err := NewClient("", WithHTTPClient(httpMock)).
		DownloadImage("https://example.com/foo", bytes.NewBuffer([]byte{}))

	assert.Zero(t, written)
	assert.Error(t, err)
	assert.Regexp(t, `(?is)tinypng\.com.+Foo \(bar baz\)$`, err.Error())
}

func TestClient_DownloadImage4xxErrorWithWrongJson(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{},
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(`{"error":"broken json']`))),
			StatusCode: http.StatusLocked, // <-- can be any (instead 401)
		}, nil
	}

	written, err := NewClient("", WithHTTPClient(httpMock)).
		DownloadImage("https://example.com/foo", bytes.NewBuffer([]byte{}))

	assert.Zero(t, written)
	assert.Error(t, err)
	assert.Regexp(t, `(?is)tinypng\.com.+decoding.+failed`, err.Error())
}

func TestClient_DownloadImageHTTPErrorAbove599(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Header:     http.Header{},
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			StatusCode: 600, // <-- important
		}, nil
	}

	written, err := NewClient("", WithHTTPClient(httpMock)).
		DownloadImage("https://example.com/foo", bytes.NewBuffer([]byte{}))

	assert.Zero(t, written)
	assert.Error(t, err)
	assert.Regexp(t, `(?is)tinypng\.com.+unexpected.+code`, err.Error())
}

func TestClient_DownloadImageHttpClientError(t *testing.T) {
	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("foo bar")
	}

	written, err := NewClient("", WithHTTPClient(httpMock)).
		DownloadImage("https://example.com/foo", bytes.NewBuffer([]byte{}))

	assert.Zero(t, written)
	assert.Error(t, err)
	assert.Equal(t, "foo bar", err.Error())
}

func TestClient_CompressSuccessful(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	origFile, err := os.OpenFile("./testdata/image.png", os.O_RDONLY, 0)
	assert.NoError(t, err)

	defer origFile.Close()

	origFileBody, err := ioutil.ReadAll(origFile) // read file for future asserting
	assert.NoError(t, err)
	_, err = origFile.Seek(0, 0) // reset file "cursor"
	assert.NoError(t, err)

	var (
		reqCounter         uint
		compressedFileBody []byte
	)

	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		defer func() { reqCounter++ }()

		switch reqCounter {
		case 0:
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "https://api.tinify.com/shrink", req.URL.String())
			assert.Equal(t, authHeaderValue(t, "blah-key"), req.Header.Get("Authorization"))

			body, _ := ioutil.ReadAll(req.Body)
			assert.Equal(t, origFileBody, body)

			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     http.Header{"Compression-Count": {"123454321"}},
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
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
						"url":"https://api.tinify.com/output/foobar"
					}
				}`))),
			}, nil

		case 1:
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "https://api.tinify.com/output/foobar", req.URL.String())
			assert.Equal(t, authHeaderValue(t, "blah-key"), req.Header.Get("Authorization"))

			compressedFile, err := os.OpenFile("./testdata/image_compressed.png", os.O_RDONLY, 0)
			assert.NoError(t, err)

			compressedFileBody, err = ioutil.ReadAll(compressedFile) // read file for future asserting
			assert.NoError(t, err)
			_, err = compressedFile.Seek(0, 0) // reset file "cursor"
			assert.NoError(t, err)

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       compressedFile,
			}, nil

		default:
			t.Fatal("unexpected request detected:", req)

			return nil, nil
		}
	}

	dest := bytes.NewBuffer([]byte{})

	info, err := NewClient("blah-key", WithHTTPClient(httpMock), WithContext(ctx)).
		Compress(origFile, dest, time.Second, time.Second)

	assert.NoError(t, err)
	assert.Equal(t, uint64(4633), info.Input.Size)
	assert.Equal(t, "image/png", info.Input.Type)
	assert.Equal(t, uint64(1636), info.Output.Size)
	assert.Equal(t, "image/png", info.Output.Type)
	assert.Equal(t, uint64(123), info.Output.Width)
	assert.Equal(t, uint64(321), info.Output.Height)
	assert.Equal(t, float32(0.3531), info.Output.Ratio)
	assert.Equal(t, "https://api.tinify.com/output/foobar", info.Output.URL)

	assert.Equal(t, uint64(123454321), info.CompressionCount)

	assert.Equal(t, compressedFileBody, dest.Bytes())
}

func TestClient_CompressImageCompressingFailed(t *testing.T) {
	origFile, err := os.OpenFile("./testdata/image.png", os.O_RDONLY, 0)
	assert.NoError(t, err)

	defer origFile.Close()

	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://api.tinify.com/shrink", req.URL.String())

		return nil, errors.New("foo bar")
	}

	info, err := NewClient("blah-key", WithHTTPClient(httpMock)).
		Compress(origFile, bytes.NewBuffer([]byte{}), time.Second, time.Second)

	assert.Nil(t, info)
	assert.Error(t, err)
	assert.Equal(t, "foo bar", err.Error())
}

func TestClient_CompressImageDownloadingFailed(t *testing.T) {
	origFile, err := os.OpenFile("./testdata/image.png", os.O_RDONLY, 0)
	assert.NoError(t, err)

	defer origFile.Close()

	var reqCounter uint

	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
		defer func() { reqCounter++ }()

		switch reqCounter {
		case 0:
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "https://api.tinify.com/shrink", req.URL.String())

			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     http.Header{"Compression-Count": {"123454321"}},
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
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
						"url":"https://api.tinify.com/output/foobar"
					}
				}`))),
			}, nil

		case 1:
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "https://api.tinify.com/output/foobar", req.URL.String())

			return nil, errors.New("bar baz")

		default:
			t.Fatal("unexpected request detected:", req)

			return nil, nil
		}
	}

	info, err := NewClient("blah-key", WithHTTPClient(httpMock)).
		Compress(origFile, bytes.NewBuffer([]byte{}), time.Second, time.Second)

	assert.Nil(t, info)
	assert.Error(t, err)
	assert.Equal(t, "bar baz", err.Error())
}

package tinypng_test

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"gh.tarampamp.am/tinifier/v5/pkg/tinypng"
)

//go:embed testdata/image.png
var srcImage []byte

//go:embed testdata/image_compressed.png
var compressedImage []byte

type httpClientFunc func(*http.Request) (*http.Response, error)

func (f httpClientFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func authHeaderValue(t *testing.T, apiKey string) string {
	t.Helper()

	return "Basic " + base64.StdEncoding.EncodeToString([]byte("api:"+apiKey))
}

func TestClient_UsedQuota(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			assertEqual(t, http.MethodPost, req.Method)
			assertEqual(t, "https://api.tinify.com/shrink", req.URL.String())
			assertEqual(t, authHeaderValue(t, "foo-key"), req.Header.Get("Authorization"))

			return &http.Response{
				Header: http.Header{
					"Compression-Count": {"123454321", "any", "values"},
				},
				Body: io.NopCloser(bytes.NewReader([]byte{})),
			}, nil
		}

		count, err := tinypng.
			NewClient("foo-key", tinypng.WithHTTPClient(httpMock)).
			UsedQuota(t.Context())

		assertEqual(t, uint64(123454321), count)
		assertNoError(t, err)
	})

	t.Run("wrong header value", func(t *testing.T) {
		t.Parallel()

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Header: http.Header{
					"Compression-Count": {"foo bar"}, // <-- wrong value
				},
				Body: io.NopCloser(bytes.NewReader([]byte{})),
			}, nil
		}

		count, err := tinypng.
			NewClient("", tinypng.WithHTTPClient(httpMock)).
			UsedQuota(t.Context())

		assertEqual(t, uint64(0), count)
		assertError(t, err)
		assertErrorContains(t, err, "tinypng:", "wrong HTTP header")
	})

	t.Run("missing header", func(t *testing.T) {
		t.Parallel()

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Header: http.Header{
					// "Compression-Count": {"123454321", "any", "values"}, // <-- nothing is here
				},
				Body: io.NopCloser(bytes.NewReader([]byte{})),
			}, nil
		}

		count, err := tinypng.
			NewClient("", tinypng.WithHTTPClient(httpMock)).
			UsedQuota(t.Context())

		assertEqual(t, uint64(0), count)
		assertError(t, err)
		assertErrorContains(t, err, "tinypng:", "header 'Compression-Count' was not found")
	})

	t.Run("http client error", func(t *testing.T) {
		t.Parallel()

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("foo bar")
		}

		count, err := tinypng.
			NewClient("", tinypng.WithHTTPClient(httpMock)).
			UsedQuota(t.Context())

		assertEqual(t, uint64(0), count)
		assertError(t, err)
		assertErrorContains(t, err, "foo bar")
	})

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
				StatusCode: http.StatusUnauthorized, // <-- important
			}, nil
		}

		count, err := tinypng.
			NewClient("", tinypng.WithHTTPClient(httpMock)).
			UsedQuota(t.Context())

		assertEqual(t, uint64(0), count)
		assertErrorIs(t, err, tinypng.ErrUnauthorized)
	})
}

func TestClient_Compress(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		var srcBuff = bytes.NewBuffer(srcImage)

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			assertEqual(t, http.MethodPost, req.Method)
			assertEqual(t, "https://api.tinify.com/shrink", req.URL.String())
			assertEqual(t, authHeaderValue(t, "bar-key"), req.Header.Get("Authorization"))

			body, _ := io.ReadAll(req.Body)
			assertSlicesEqual(t, srcImage, body)

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

		info, err := tinypng.
			NewClient("bar-key", tinypng.WithHTTPClient(httpMock)).
			Compress(ctx, srcBuff)

		assertNoError(t, err)
		assertEqual(t, uint64(1636), info.Size)
		assertEqual(t, "image/png", info.Type)

		assertEqual(t, uint32(123), info.Width)
		assertEqual(t, uint32(321), info.Height)
		assertEqual(t, "https://api.tinify.com/output/someRandomResultImageHash", info.URL)

		assertEqual(t, uint64(123454321), info.UsedQuota)
	})

	t.Run("wrong json response", func(t *testing.T) {
		t.Parallel()

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     http.Header{"Compression-Count": {"123454321"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"broken json']`))),
			}, nil
		}

		info, err := tinypng.
			NewClient("", tinypng.WithHTTPClient(httpMock)).
			Compress(t.Context(), bytes.NewBuffer(srcImage))

		assertNil(t, info)
		assertError(t, err)
		assertErrorContains(t, err, "tinypng: ", "decoding failed")
	})

	t.Run("wrong json response", func(t *testing.T) {
		t.Parallel()

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     http.Header{"Compression-Count": {"123454321"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"broken json']`))),
			}, nil
		}

		info, err := tinypng.
			NewClient("", tinypng.WithHTTPClient(httpMock)).
			Compress(t.Context(), bytes.NewBuffer(srcImage))

		assertNil(t, info)
		assertError(t, err)
		assertErrorContains(t, err, "tinypng: ", "decoding failed")
	})

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
				StatusCode: http.StatusUnauthorized, // <-- important
			}, nil
		}

		info, err := tinypng.
			NewClient("", tinypng.WithHTTPClient(httpMock)).
			Compress(t.Context(), bytes.NewBuffer(srcImage))

		assertNil(t, info)
		assertErrorIs(t, err, tinypng.ErrUnauthorized)
	})

	t.Run("too many requests", func(t *testing.T) {
		t.Parallel()

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
				StatusCode: http.StatusTooManyRequests, // <-- important
			}, nil
		}

		info, err := tinypng.
			NewClient("", tinypng.WithHTTPClient(httpMock)).
			Compress(t.Context(), bytes.NewBuffer(srcImage))

		assertNil(t, info)
		assertErrorIs(t, err, tinypng.ErrTooManyRequests)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
				StatusCode: http.StatusBadRequest, // <-- important
			}, nil
		}

		info, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).
			Compress(t.Context(), bytes.NewBuffer(srcImage))

		assertNil(t, info)
		assertErrorIs(t, err, tinypng.ErrBadRequest)
	})

	t.Run("http error above 599", func(t *testing.T) {
		t.Parallel()

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
				StatusCode: 600, // <-- important
			}, nil
		}

		info, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).
			Compress(t.Context(), bytes.NewBuffer(srcImage))

		assertNil(t, info)
		assertError(t, err)
		assertErrorContains(t, err, "tinypng: ", "unexpected HTTP response status code", "600")
	})

	t.Run("4xx error", func(t *testing.T) {
		t.Parallel()

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Header:     http.Header{"Compression-Count": {"123"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"Foo","message":"bar baz."}`))),
				StatusCode: http.StatusTeapot, // <-- can be any (instead 401, 429 and 400)
			}, nil
		}

		info, err := tinypng.
			NewClient("", tinypng.WithHTTPClient(httpMock)).
			Compress(t.Context(), bytes.NewBuffer(srcImage))

		assertNil(t, info)
		assertError(t, err)
		assertErrorContains(t, err, "tinypng: Foo (bar baz)")
	})

	t.Run("4xx error with wrong json", func(t *testing.T) {
		t.Parallel()

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Header:     http.Header{"Compression-Count": {"123"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"broken json']`))),
				StatusCode: http.StatusLocked, // <-- can be any (instead 401, 429 and 400)
			}, nil
		}

		info, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).
			Compress(t.Context(), bytes.NewBuffer(srcImage))

		assertNil(t, info)
		assertError(t, err)
		assertErrorContains(t, err, "tinypng", "decoding failed")
	})

	t.Run("http client error", func(t *testing.T) {
		t.Parallel()

		var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("foo bar")
		}

		info, err := tinypng.NewClient("", tinypng.WithHTTPClient(httpMock)).
			Compress(t.Context(), bytes.NewBuffer(srcImage))

		assertNil(t, info)
		assertError(t, err)
		assertEqual(t, "tinypng: foo bar", err.Error())
	})
}

func TestCompressed_Download(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

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

		info, err := tinypng.
			NewClient("bar-key", tinypng.WithHTTPClient(httpMock)).
			Compress(t.Context(), bytes.NewBuffer(srcImage))
		assertNoError(t, err)

		out := bytes.NewBuffer(nil)
		err = info.Download(t.Context(), out)

		assertNoError(t, err)
		assertNotEmpty(t, out)
		assertSlicesEqual(t, compressedImage, out.Bytes())
	})

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()

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

		info, err := tinypng.
			NewClient("bar-key", tinypng.WithHTTPClient(httpMock)).
			Compress(t.Context(), bytes.NewBuffer(srcImage))
		assertNoError(t, err)

		out := bytes.NewBuffer(nil)
		err = info.Download(t.Context(), out)

		assertSlicesEqual(t, nil, out.Bytes())
		assertErrorIs(t, err, tinypng.ErrUnauthorized)
	})

	t.Run("too many requests", func(t *testing.T) {
		t.Parallel()

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

		info, err := tinypng.
			NewClient("bar-key", tinypng.WithHTTPClient(httpMock)).
			Compress(t.Context(), bytes.NewBuffer(srcImage))
		assertNoError(t, err)

		out := bytes.NewBuffer(nil)
		err = info.Download(t.Context(), out)

		assertSlicesEqual(t, nil, out.Bytes())
		assertErrorIs(t, err, tinypng.ErrTooManyRequests)
	})

	t.Run("4xx error", func(t *testing.T) {
		t.Parallel()

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

		info, err := tinypng.
			NewClient("bar-key", tinypng.WithHTTPClient(httpMock)).
			Compress(t.Context(), bytes.NewBuffer(srcImage))
		assertNoError(t, err)

		out := bytes.NewBuffer(nil)
		err = info.Download(t.Context(), out)

		assertSlicesEqual(t, nil, out.Bytes())
		assertEqual(t, "tinypng: Foo (bar baz)", err.Error())
	})
}

func assertEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()

	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func assertNotEmpty[T comparable](t *testing.T, v T) {
	t.Helper()

	if v == *new(T) {
		t.Fatalf("expected not empty, got %v", v)
	}
}

func assertSlicesEqual[T comparable](t *testing.T, expected, actual []T) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}

	for i := range expected {
		if expected[i] != actual[i] {
			t.Fatalf("expected %v, got %v", expected, actual)
		}
	}
}

func assertError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func assertErrorIs(t *testing.T, err, target error) {
	t.Helper()

	if !errors.Is(err, target) {
		t.Fatalf("expected error to be %v, got %v", target, err)
	}
}

func assertErrorContains(t *testing.T, err error, substr ...string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	for _, s := range substr {
		var got = err.Error()

		if !strings.Contains(got, s) {
			t.Fatalf("expected error to contain %q, got %q", s, got)
		}
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertNil(t *testing.T, v any) {
	t.Helper()

	if ref := reflect.ValueOf(v); ref.Kind() == reflect.Ptr && !ref.IsNil() {
		t.Fatalf("expected nil, got %v", v)
	}
}

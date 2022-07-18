// Package tinypng is `tinypng.com` API client implementation.
package tinypng

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

const shrinkEndpoint = "https://api.tinify.com/shrink" // API endpoint for images shrinking.

type httpClient interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	httpClient httpClient // HTTP client for requests making

	apiKeyMu sync.RWMutex
	apiKey   string // API key for requests making (get own on <https://tinypng.com/developers>)
}

type (
	CompressionResult struct {
		Input struct {
			Size uint64 `json:"size"` // eg.: 37745
			Type string `json:"type"` // eg.: image/png
		} `json:"input"`
		Output struct {
			Size   uint64  `json:"size"`   // eg.: 35380
			Type   string  `json:"type"`   // eg.: image/png
			Width  uint64  `json:"width"`  // eg.: 512
			Height uint64  `json:"height"` // eg.: 512
			Ratio  float32 `json:"ratio"`  // eg.: 0.9373
			URL    string  `json:"url"`    // eg.: https://api.tinify.com/output/foobar
		} `json:"output"`
		CompressionCount uint64 // used quota value
	}
)

// NewClient creates new tinypng client instance. Options can be used to fine client tuning.
func NewClient(apiKey string, options ...ClientOption) *Client {
	c := &Client{apiKey: apiKey, httpClient: new(http.Client)}

	for _, opt := range options {
		opt(c)
	}

	return c
}

// SetAPIKey sets API key for requests making.
func (c *Client) SetAPIKey(key string) {
	c.apiKeyMu.Lock()
	c.apiKey = key
	c.apiKeyMu.Unlock()
}

// Compress reads image from passed source and compress them on tinypng side. Compressed result will be written to the
// passed destination (additional information about compressed image will be returned too).
// If the provided src is also an io.Closer - it will be closed automatically by HTTP client (if default HTTP client is
// used).
func (c *Client) Compress(ctx context.Context, src io.Reader, dest io.Writer) (*CompressionResult, error) {
	result, err := c.CompressImage(ctx, src)
	if err != nil {
		return nil, err
	}

	_, err = c.DownloadImage(ctx, result.Output.URL, dest)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CompressionCount returns compressions count for current API key (used quota value). By default, for free API keys
// quota is equals to 500.
func (c *Client) CompressionCount(ctx context.Context) (uint64, error) {
	// make a "fake" image uploading attempt for "Compression-Count" response header value reading.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, shrinkEndpoint, http.NoBody)
	if err != nil {
		return 0, err
	}

	c.setupRequestAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}

	_ = resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return 0, ErrUnauthorized

	default:
		return c.extractCompressionCount(resp.Header)
	}
}

// extractCompressionCount extracts `compression-count` header value from HTTP response headers.
func (c *Client) extractCompressionCount(headers http.Header) (uint64, error) {
	const headerName = "Compression-Count"

	if val, ok := headers[headerName]; ok {
		count, err := strconv.ParseUint(val[0], 10, 64) //nolint:gomnd
		if err == nil {
			return count, nil
		}

		return 0, fmt.Errorf("%s wrong HTTP header '%s' value: %w", errorsPrefix, headerName, err)
	}

	return 0, fmt.Errorf("%s HTTP header '%s' was not found", errorsPrefix, headerName)
}

// CompressImage uploads image content from passed source to the tinypng server for compression. When process is done -
// compression result (just information, not compressed image content) will be returned.
// If the provided src is also an io.Closer - it will be closed automatically by HTTP client (if default HTTP client is
// used).
func (c *Client) CompressImage(ctx context.Context, src io.Reader) (*CompressionResult, error) {
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, shrinkEndpoint, src)
	if reqErr != nil {
		return nil, reqErr
	}

	c.setupRequestAuth(req)
	req.Header.Set("Accept", "application/json") // is not necessary, but looks correct

	resp, respErr := c.httpClient.Do(req)
	if respErr != nil {
		return nil, respErr
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	switch code := resp.StatusCode; {
	case code == http.StatusCreated:
		var result CompressionResult

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, errors.New(errorsPrefix + " response decoding failed: " + err.Error())
		}

		if count, err := c.extractCompressionCount(resp.Header); err == nil { // error will be ignored
			result.CompressionCount = count
		}

		return &result, nil

	case code >= 400 && code < 599:
		switch code {
		case http.StatusUnauthorized:
			return nil, ErrUnauthorized

		case http.StatusTooManyRequests:
			return nil, ErrTooManyRequests

		case http.StatusBadRequest:
			return nil, ErrBadRequest

		default:
			return nil, errors.New(errorsPrefix + " " + c.parseServerError(resp.Body).Error())
		}

	default:
		return nil, fmt.Errorf("%s unexpected HTTP response code (%d)", errorsPrefix, code)
	}
}

// DownloadImage from remote server and write to the passed destination. It returns the number of written bytes.
func (c *Client) DownloadImage(ctx context.Context, url string, dest io.Writer) (int64, error) {
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if reqErr != nil {
		return 0, reqErr
	}

	c.setupRequestAuth(req)

	resp, respErr := c.httpClient.Do(req)
	if respErr != nil {
		return 0, respErr
	}

	defer func() { _ = resp.Body.Close() }()

	switch code := resp.StatusCode; {
	case code == http.StatusOK:
		written, writingErr := io.Copy(dest, resp.Body)
		if writingErr != nil {
			return written, writingErr
		}

		return written, nil

	case code >= 400 && code < 599:
		switch code {
		case http.StatusUnauthorized:
			return 0, ErrUnauthorized

		case http.StatusTooManyRequests:
			return 0, ErrTooManyRequests

		default:
			return 0, errors.New(errorsPrefix + " " + c.parseServerError(resp.Body).Error())
		}

	default:
		return 0, fmt.Errorf("%s unexpected HTTP response code (%d)", errorsPrefix, code)
	}
}

// setupRequestAuth sets all required properties for HTTP request (eg.: API key).
func (c *Client) setupRequestAuth(request *http.Request) {
	c.apiKeyMu.RLock()
	request.SetBasicAuth("api", c.apiKey)
	c.apiKeyMu.RUnlock()
}

// parseServerError reads HTTP response content as a JSON-string, parse them and converts into go-error. This function
// should never returns nil!
func (c *Client) parseServerError(content io.Reader) error {
	var e struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(content).Decode(&e); err != nil {
		return fmt.Errorf("error decoding failed: %w", err)
	}

	return fmt.Errorf("%s (%s)", e.Error, strings.Trim(e.Message, ". "))
}

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
	"time"
)

// WithoutTimeout is special value for timeouts disabling.
const WithoutTimeout = time.Duration(0)

const shrinkEndpoint = "https://api.tinify.com/shrink" // API endpoint for images shrinking.

// var DefaultClient *Client = NewClient("")

type httpClient interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	// Client context is used for requests making. It allows to cancel any "long" requests or limit them with timeout.
	ctx context.Context

	// Mutex is used for apiKey concurrent access protection.
	mu sync.Mutex

	// API key for requests making (get own on <https://tinypng.com/developers>).
	apiKey string

	// This timeout will be used for requests execution time limiting by default. Set WithoutTimeout (or simply `0`)
	// for timeouts disabling (is used by default).
	defaultTimeout time.Duration

	// HTTP client for requests making.
	httpClient httpClient
}

type (
	compressionInput struct {
		Size uint64 `json:"size"` // eg.: 37745
		Type string `json:"type"` // eg.: image/png
	}

	compressionOutput struct {
		Size   uint64  `json:"size"`   // eg.: 35380
		Type   string  `json:"type"`   // eg.: image/png
		Width  uint64  `json:"width"`  // eg.: 512
		Height uint64  `json:"height"` // eg.: 512
		Ratio  float32 `json:"ratio"`  // eg.: 0.9373
		URL    string  `json:"url"`    // eg.: https://api.tinify.com/output/foobar
	}

	CompressionResult struct {
		Input            compressionInput  `json:"input"`
		Output           compressionOutput `json:"output"`
		CompressionCount uint64            // used quota value
	}
)

// NewClient creates new tinypng client instance. Options can be used to fine client tuning.
func NewClient(apiKey string, options ...ClientOption) *Client {
	c := &Client{
		apiKey:         apiKey,
		defaultTimeout: WithoutTimeout,
	}

	for i := 0; i < len(options); i++ {
		options[i](c)
	}

	if c.ctx == nil {
		c.ctx = context.Background()
	}

	if c.httpClient == nil {
		c.httpClient = new(http.Client)
	}

	return c
}

// SetAPIKey sets API key for requests making.
func (c *Client) SetAPIKey(key string) {
	c.mu.Lock()
	c.apiKey = key
	c.mu.Unlock()
}

// Compress reads image from passed source and compress them on tinypng side. Compressed result will be wrote to the
// passed destination (additional information about compressed image will be returned too).
// You can use two timeouts - first for image uploading and response waiting, and second - for image downloading.
// If the provided src is also an io.Closer - it will be closed automatically by HTTP client (if default HTTP client is
// used).
func (c *Client) Compress(src io.Reader, dest io.Writer, timeouts ...time.Duration) (*CompressionResult, error) {
	var compressTimeout, downloadTimeout = c.defaultTimeout, c.defaultTimeout // setup defaults

	if len(timeouts) > 0 {
		compressTimeout = timeouts[0]
	}

	result, err := c.compressImage(src, compressTimeout)
	if err != nil {
		return nil, err
	}

	if len(timeouts) > 1 {
		downloadTimeout = timeouts[1]
	}

	_, err = c.downloadImage(result.Output.URL, dest, downloadTimeout)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CompressionCount returns compressions count for current API key (used quota value). By default, for free API keys
// quota is equals to 500.
func (c *Client) CompressionCount(timeout ...time.Duration) (uint64, error) {
	var t = c.defaultTimeout

	if len(timeout) > 0 {
		t = timeout[0]
	}

	return c.compressionCount(t)
}

// CompressImage uploads image content from passed source to the tinypng server for compression. When process is done -
// compression result (just information, not compressed image content) will be returned.
// If the provided src is also an io.Closer - it will be closed automatically by HTTP client (if default HTTP client is
// used).
func (c *Client) CompressImage(src io.Reader, timeout ...time.Duration) (*CompressionResult, error) {
	var t = c.defaultTimeout

	if len(timeout) > 0 {
		t = timeout[0]
	}

	return c.compressImage(src, t)
}

// DownloadImage from remote server and write to the passed destination. It returns the number of written bytes.
func (c *Client) DownloadImage(url string, dest io.Writer, timeout ...time.Duration) (int64, error) {
	var t = c.defaultTimeout

	if len(timeout) > 0 {
		t = timeout[0]
	}

	return c.downloadImage(url, dest, t)
}

// requestCtx creates context (with cancellation function) for request making. Do not forget to call cancellation
// function in your code anyway.
func (c *Client) requestCtx(timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == WithoutTimeout {
		return c.ctx, func() {
			// do nothing
		}
	}

	return context.WithTimeout(c.ctx, timeout)
}

// compressImage reads image content from src and sends them to the tinypng server with request timeout limitation.
// If the provided src is also an io.Closer - it will be closed automatically by HTTP client (if default HTTP client is
// used).
func (c *Client) compressImage(src io.Reader, timeout time.Duration) (*CompressionResult, error) {
	var ctx, cancel = c.requestCtx(timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, shrinkEndpoint, src)
	if err != nil {
		return nil, err
	}

	c.setupRequestAuth(req)
	req.Header.Set("Accept", "application/json") // is not necessary, but looks correct

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
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

// compressionCount makes "fake" image uploading attempt for "Compression-Count" response header value reading.
func (c *Client) compressionCount(timeout time.Duration) (uint64, error) {
	var ctx, cancel = c.requestCtx(timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, shrinkEndpoint, nil)
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

// setupRequestAuth sets all required properties for HTTP request (eg.: API key).
func (c *Client) setupRequestAuth(request *http.Request) {
	c.mu.Lock()
	k := c.apiKey
	c.mu.Unlock()

	request.SetBasicAuth("api", k)
}

// extractCompressionCount extracts `compression-count` header value from HTTP response headers.
func (c *Client) extractCompressionCount(headers http.Header) (uint64, error) {
	const headerName = "Compression-Count"

	if val, ok := headers[headerName]; ok {
		count, err := strconv.ParseUint(val[0], 10, 64)
		if err == nil {
			return count, nil
		}

		return 0, fmt.Errorf("%s wrong HTTP header '%s' value: %w", errorsPrefix, headerName, err)
	}

	return 0, fmt.Errorf("%s HTTP header '%s' was not found", errorsPrefix, headerName)
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

// downloadImage downloads image by passed URL (usually from tinypng remote server) with request timeout limitation.
func (c *Client) downloadImage(url string, dest io.Writer, timeout time.Duration) (int64, error) {
	var ctx, cancel = c.requestCtx(timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	c.setupRequestAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	switch code := resp.StatusCode; {
	case code == http.StatusOK:
		written, err := io.Copy(dest, resp.Body)
		if err != nil {
			return 0, err
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

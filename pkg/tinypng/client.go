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
)

var (
	ErrUnauthorized    = errors.New("unauthorized (invalid credentials)")
	ErrTooManyRequests = errors.New("too many requests (limit has been exceeded)")
	ErrBadRequest      = errors.New("bad request (empty file or wrong format)")
)

const shrinkEndpoint = "https://api.tinify.com/shrink" // API endpoint for images shrinking

// httpClient is an HTTP client used for requests.
type httpClient interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(*http.Request) (*http.Response, error)
}

// ClientOption allows to set up some internal client properties from outside.
type ClientOption func(*Client)

// WithHTTPClient setups allows to pass custom HTTP client implementation.
func WithHTTPClient(httpClient httpClient) ClientOption {
	return func(c *Client) { c.httpClient = httpClient }
}

// Client is a tinypng client implementation.
type Client struct {
	httpClient httpClient // HTTP client for requests making
	apiKey     string     // API key (string) for requests making (get own on <https://tinypng.com/developers>)
}

// NewClient creates a new tinypng client instance. Options can be used to fine client tuning.
func NewClient(apiKey string, options ...ClientOption) *Client {
	c := &Client{
		httpClient: new(http.Client),
		apiKey:     apiKey,
	}

	for _, opt := range options {
		opt(c)
	}

	return c
}

// UsedQuota returns compressions count for current API key (used quota value). By default, for free API keys quota
// is equals to 500.
func (c *Client) UsedQuota(ctx context.Context) (_ uint64, outErr error) {
	defer func() { // wrap error with package-specific error
		if outErr != nil {
			outErr = fmt.Errorf("tinypng: %w", outErr)
		}
	}()

	// make a "fake" image uploading attempt for "Compression-Count" response header value reading
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, shrinkEndpoint, http.NoBody)
	if reqErr != nil {
		return 0, reqErr
	}

	req.SetBasicAuth("api", c.apiKey)

	resp, respErr := c.httpClient.Do(req)
	if respErr != nil {
		return 0, respErr
	}

	_ = resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return 0, ErrUnauthorized
	}

	count, cErr := c.extractCompressionCount(resp.Header)
	if cErr != nil {
		return 0, cErr
	}

	return count, nil
}

// Compress uploads image content from the provided source to the tinypng server for compression. When the process
// is done, the compression result (just information, not compressed image content) will be returned. If the provided
// source is also an io.Closer - it will be closed automatically by the HTTP client (if the default HTTP client is
// used).
func (c *Client) Compress(ctx context.Context, src io.Reader) (_ *Compressed, outErr error) {
	defer func() { // wrap error with package-specific error
		if outErr != nil {
			outErr = fmt.Errorf("tinypng: %w", outErr)
		}
	}()

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, shrinkEndpoint, src)
	if reqErr != nil {
		return nil, reqErr
	}

	req.SetBasicAuth("api", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, respErr := c.httpClient.Do(req)
	if respErr != nil {
		return nil, respErr
	}

	defer func() { _ = resp.Body.Close() }()

	switch code := resp.StatusCode; {
	case code == http.StatusCreated:
		var p struct { // payload
			Input struct {
				Size uint64 `json:"size"` // eg.: 37745
				Type string `json:"type"` // eg.: image/png
			} `json:"input"`
			Output struct {
				Size   uint64  `json:"size"`   // eg.: 35380
				Type   string  `json:"type"`   // eg.: image/png
				Width  uint32  `json:"width"`  // eg.: 512
				Height uint32  `json:"height"` // eg.: 512
				Ratio  float32 `json:"ratio"`  // eg.: 0.9373
				URL    string  `json:"url"`    // eg.: https://api.tinify.com/output/foobar
			} `json:"output"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
			return nil, fmt.Errorf("response decoding failed: %w", err)
		}

		var result = Compressed{
			client: c,
			Type:   p.Output.Type,
			Size:   p.Output.Size,
			URL:    p.Output.URL,
			Width:  p.Output.Width,
			Height: p.Output.Height,
		}

		if count, err := c.extractCompressionCount(resp.Header); err == nil {
			result.UsedQuota = count
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
			return nil, parseRemoteError(resp.Body)
		}
	default:
		return nil, fmt.Errorf("unexpected HTTP response status code (%d)", code)
	}
}

// Compressed represents tinypng compression result.
type Compressed struct {
	client *Client // reference to the Client instance

	UsedQuota     uint64 // used quota value, eg.: 123
	Type          string // type of the compressed image, eg.: image/png
	Size          uint64 // the size (in bytes) of the compressed image, eg.: 35380
	URL           string // eg.: https://api.tinify.com/output/foobar
	Width, Height uint32 // the size of result image, eg.: 512, 512
}

// Download image from remote server and write to the passed destination.
//
// If the provided source is also an io.Closer - it will be closed automatically by the HTTP client (if the
// default HTTP client is used).
func (c Compressed) Download(ctx context.Context, to io.Writer) (outErr error) {
	defer func() { // wrap error with package-specific error
		if outErr != nil {
			outErr = fmt.Errorf("tinypng: %w", outErr)
		}
	}()

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, http.NoBody)
	if reqErr != nil {
		return reqErr
	}

	req.SetBasicAuth("api", c.client.apiKey)

	resp, respErr := c.client.httpClient.Do(req)
	if respErr != nil {
		return respErr
	}

	defer func() { _ = resp.Body.Close() }()

	switch code := resp.StatusCode; {
	case code == http.StatusOK:
		if _, err := io.Copy(to, resp.Body); err != nil {
			return err
		}

		return nil
	case code >= 400 && code < 599:
		switch code {
		case http.StatusUnauthorized:
			return ErrUnauthorized
		case http.StatusTooManyRequests:
			return ErrTooManyRequests
		default:
			return parseRemoteError(resp.Body)
		}

	default:
		return fmt.Errorf("unexpected HTTP response status code (%d)", code)
	}
}

// parseRemoteError reads HTTP response content as a JSON-string, parse them and converts into go-error.
//
// This function should never return nil!
func parseRemoteError(content io.Reader) error {
	var e struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(content).Decode(&e); err != nil {
		return fmt.Errorf("error decoding failed: %w", err)
	}

	return fmt.Errorf("%s (%s)", e.Error, strings.Trim(e.Message, ". "))
}

// extractCompressionCount extracts `compression-count` header value from HTTP response headers.
func (c *Client) extractCompressionCount(headers http.Header) (uint64, error) {
	const headerName = "Compression-Count"

	if val, ok := headers[headerName]; ok && len(val) > 0 {
		count, err := strconv.ParseUint(val[0], 10, 64)
		if err == nil {
			return count, nil
		}

		return 0, fmt.Errorf("wrong HTTP header '%s' value (%w)", headerName, err)
	}

	return 0, fmt.Errorf("HTTP header '%s' was not found", headerName)
}

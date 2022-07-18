// Package tinypng is `tinypng.com` API client implementation.
package tinypng

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
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
	httpClient httpClient   // HTTP client for requests making
	apiKey     atomic.Value // API key (string) for requests making (get own on <https://tinypng.com/developers>)
}

// NewClient creates a new tinypng client instance. Options can be used to fine client tuning.
func NewClient(apiKey string, options ...ClientOption) *Client {
	c := &Client{httpClient: new(http.Client)}

	c.apiKey.Store(apiKey)

	for _, opt := range options {
		opt(c)
	}

	return c
}

// SetAPIKey sets the API key for the requests making.
func (c *Client) SetAPIKey(key string) { c.apiKey.Store(key) }

// UsedQuota returns compressions count for current API key (used quota value). By default, for free API keys quota
// is equals to 500.
func (c *Client) UsedQuota(ctx context.Context) (uint64, error) {
	// make a "fake" image uploading attempt for "Compression-Count" response header value reading
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, shrinkEndpoint, http.NoBody)
	if err != nil {
		return 0, err
	}

	authHTTPRequest(req, c.apiKey.Load().(string))

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

// Compress uploads image content from the provided source to the tinypng server for compression. When the process
// is done, the compression result (just information, not compressed image content) will be returned. If the provided
// source is also an io.Closer - it will be closed automatically by the HTTP client (if the default HTTP client is
// used).
func (c *Client) Compress(ctx context.Context, src io.Reader) (*Compressed, error) { //nolint:funlen
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, shrinkEndpoint, src)
	if reqErr != nil {
		return nil, reqErr
	}

	authHTTPRequest(req, c.apiKey.Load().(string))
	req.Header.Set("Accept", "application/json") // is not necessary, but looks correct

	resp, respErr := c.httpClient.Do(req)
	if respErr != nil {
		return nil, respErr
	}

	defer func() { _ = resp.Body.Close() }()

	switch code := resp.StatusCode; {
	case code == http.StatusCreated:
		var (
			p struct { // payload
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

			result = Compressed{client: c}
		)

		if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
			return nil, newErrorf("response decoding failed: %s", err.Error())
		}

		if count, err := c.extractCompressionCount(resp.Header); err == nil {
			result.usedQuota = count
		}

		result.imgType = p.Output.Type
		result.size = p.Output.Size
		result.url = p.Output.URL
		result.width, result.height = p.Output.Width, p.Output.Height

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
			return nil, newError(parseRemoteError(resp.Body).Error())
		}

	default:
		return nil, newErrorf("unexpected HTTP response code (%d)", code)
	}
}

// Compressed represents tinypng compression result.
type Compressed struct {
	client *Client // reference to the Client instance

	usedQuota     uint64 // eg.: 123
	imgType       string // eg.: image/png
	size          uint64 // eg.: 35380
	url           string // eg.: https://api.tinify.com/output/foobar
	width, height uint32 // eg.: 512, 512
}

// Type returns the type of the compressed image.
func (c Compressed) Type() string { return c.imgType }

// UsedQuota returns the used quota value.
func (c Compressed) UsedQuota() uint64 { return c.usedQuota }

// Dimensions returns the dimensions of the compressed image.
func (c Compressed) Dimensions() (width, height uint32) { return c.width, c.height }

// Size returns the size (in bytes) of the compressed image.
func (c Compressed) Size() uint64 { return c.size }

// URL returns the URL of the compressed image.
func (c Compressed) URL() string { return c.url }

// Download image from remote server and write to the passed destination.
func (c Compressed) Download(ctx context.Context, to io.Writer) error {
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, c.url, http.NoBody)
	if reqErr != nil {
		return reqErr
	}

	authHTTPRequest(req, c.client.apiKey.Load().(string))

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
			return newError(parseRemoteError(resp.Body).Error())
		}

	default:
		return newErrorf("unexpected HTTP response code (%d)", code)
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

		return 0, newErrorf("wrong HTTP header '%s' value (%w)", headerName, err)
	}

	return 0, newErrorf("HTTP header '%s' was not found", headerName)
}

// authHTTPRequest sets the Authorization header to the request.
func authHTTPRequest(req *http.Request, apiKey string) {
	const authUserName = "api"

	req.SetBasicAuth(authUserName, apiKey)
}

// Package tinypng provides a client implementation for the `tinypng.com` API.
package tinypng

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrUnauthorized indicates that the provided API credentials are invalid.
	ErrUnauthorized = errors.New("unauthorized (invalid credentials)")

	// ErrTooManyRequests indicates that the API request limit has been exceeded.
	ErrTooManyRequests = errors.New("too many requests (limit exceeded)")

	// ErrBadRequest indicates an issue with the request, such as an empty file or an unsupported format.
	ErrBadRequest = errors.New("bad request (empty file or unsupported format)")
)

const shrinkEndpoint = "https://api.tinify.com/shrink" // API endpoint for image compression requests

// httpClient defines an interface for making HTTP requests.
type httpClient interface {
	// Do send an HTTP request and returns the corresponding HTTP response.
	Do(*http.Request) (*http.Response, error)
}

// ClientOption is a functional option used to configure the Client instance.
type ClientOption func(*Client)

// WithHTTPClient allows the use of a custom HTTP client implementation.
func WithHTTPClient(httpClient httpClient) ClientOption {
	return func(c *Client) { c.httpClient = httpClient }
}

// Client represents the TinyPNG API client.
type Client struct {
	httpClient httpClient // HTTP client used for making API requests
	apiKey     string     // API key for authentication (obtain from <https://tinypng.com/developers>)
}

// NewClient creates a new TinyPNG client instance with the specified API key.
// Additional options can be provided to customize the client.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	var c = Client{apiKey: apiKey}

	for _, opt := range opts {
		opt(&c)
	}

	if c.httpClient == nil { // set default HTTP client
		c.httpClient = &http.Client{
			Timeout:   60 * time.Second,                         //nolint:mnd
			Transport: &http.Transport{ForceAttemptHTTP2: true}, // use HTTP/2 (why not?)
		}
	}

	return &c
}

// ApiKey returns the API key used by the client.
func (c *Client) ApiKey() string { return c.apiKey }

// UsedQuota retrieves the number of compression requests made using the current API key.
// Free-tier accounts are limited to 500 requests per month.
func (c *Client) UsedQuota(ctx context.Context) (_ uint64, outErr error) {
	defer func() { // Wrap the error with a package-specific prefix.
		if outErr != nil {
			outErr = fmt.Errorf("tinypng: %w", outErr)
		}
	}()

	// Make a dummy image upload request to obtain the "Compression-Count" header from the response.
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, shrinkEndpoint, http.NoBody)
	if reqErr != nil {
		return 0, reqErr
	}

	req.SetBasicAuth("api", c.apiKey)

	resp, respErr := c.httpClient.Do(req)
	if respErr != nil {
		return 0, respErr
	}

	_ = resp.Body.Close() // We only need the headers, so we can safely discard the response body.

	if resp.StatusCode == http.StatusUnauthorized {
		return 0, ErrUnauthorized
	}

	// Extract the compression count from the response headers.
	count, cErr := c.extractCompressionCount(resp.Header)
	if cErr != nil {
		return 0, cErr
	}

	return count, nil
}

// Compress uploads an image to TinyPNG for compression.
// The function returns metadata about the compressed image, but not the image content itself.
// If the provided source implements io.Closer, it will be closed automatically by the HTTP client.
func (c *Client) Compress(ctx context.Context, src io.Reader) (_ *Compressed, outErr error) { //nolint:funlen
	defer func() { // Wrap the error with a package-specific prefix.
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
		// Response payload structure
		var p struct {
			Input struct {
				Size uint64 `json:"size"` // Example: 37745
				Type string `json:"type"` // Example: image/png
			} `json:"input"`
			Output struct {
				Size   uint64  `json:"size"`   // Example: 35380
				Type   string  `json:"type"`   // Example: image/png
				Width  uint32  `json:"width"`  // Example: 512
				Height uint32  `json:"height"` // Example: 512
				Ratio  float32 `json:"ratio"`  // Example: 0.9373
				URL    string  `json:"url"`    // Example: https://api.tinify.com/output/foobar
			} `json:"output"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
			return nil, fmt.Errorf("response decoding failed: %w", err)
		}

		result := Compressed{
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

// Compressed represents the result of a TinyPNG compression operation.
type Compressed struct {
	client *Client // Reference to the client instance

	UsedQuota     uint64 // Number of compressions used in the current billing period.
	Type          string // MIME type of the compressed image, e.g., "image/png".
	Size          uint64 // Size of the compressed image in bytes.
	URL           string // URL of the compressed image.
	Width, Height uint32 // Dimensions of the compressed image.
}

type (
	downloadOptions struct {
		// specific metadata will be copied from the uploaded image to the compressed version. the following values
		// are supported:
		//	- `copyright` - copyright information
		//	- `location` - GPS location
		//	- `creation` - creation date
		Preserve []string
	}

	DownloadOption func(*downloadOptions)
)

// WithDownloadPreserveCopyright specifies that the copyright information should be preserved.
func WithDownloadPreserveCopyright() DownloadOption {
	return func(o *downloadOptions) { o.Preserve = append(o.Preserve, "copyright") }
}

// WithDownloadPreserveLocation specifies that the GPS location should be preserved.
func WithDownloadPreserveLocation() DownloadOption {
	return func(o *downloadOptions) { o.Preserve = append(o.Preserve, "location") }
}

// WithDownloadPreserveCreation specifies that the creation date should be preserved.
func WithDownloadPreserveCreation() DownloadOption {
	return func(o *downloadOptions) { o.Preserve = append(o.Preserve, "creation") }
}

// Download retrieves the compressed image from the TinyPNG servers and writes it to the specified destination.
// If the provided destination implements io.Closer, it will be closed automatically by the HTTP client.
func (c Compressed) Download(ctx context.Context, to io.Writer, opt ...DownloadOption) (outErr error) { //nolint:funlen
	defer func() { // Wrap the error with a package-specific prefix.
		if outErr != nil {
			outErr = fmt.Errorf("tinypng: %w", outErr)
		}
	}()

	var opts downloadOptions
	for _, o := range opt {
		o(&opts)
	}

	var req *http.Request

	switch {
	case len(opts.Preserve) > 0:
		j, err := json.Marshal(struct {
			Preserve []string `json:"preserve"`
		}{
			Preserve: opts.Preserve,
		})
		if err != nil {
			return err
		}

		req, err = http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(j))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
	default:
		var err error

		req, err = http.NewRequestWithContext(ctx, http.MethodGet, c.URL, http.NoBody)
		if err != nil {
			return err
		}
	}

	req.SetBasicAuth("api", c.client.apiKey)

	resp, respErr := c.client.httpClient.Do(req)
	if respErr != nil {
		return respErr
	}

	defer func() { _ = resp.Body.Close() }()

	switch code := resp.StatusCode; {
	case code == http.StatusOK:
		_, err := io.Copy(to, resp.Body)

		return err
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

// parseRemoteError decodes an error response from TinyPNG and converts it into a Go error.
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

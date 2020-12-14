// Package tinypng is `tinypng.com` API client implementation.
package tinypng

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ENDPOINT is `tinypng.com` API endpoint. Docs can be found here: <https://tinypng.com/developers/reference>
const ENDPOINT string = "https://api.tinify.com/shrink"

type (
	// Client describes `tinypng.com` API client
	Client struct {
		apiKey     string
		httpClient *http.Client
	}

	ClientConfig struct {
		APIKey         string
		RequestTimeout time.Duration
	}
)

type (
	// Input follows `input` response object structure.
	Input struct {
		Size uint64 `json:"size"` // eg.: `5851`
		Type string `json:"type"` // eg.: `image/png`
	}

	// Output follows `output` response object structure.
	Output struct {
		Size   uint64  `json:"size"`   // eg.: `5851`
		Type   string  `json:"type"`   // eg.: `image/png`
		Width  uint64  `json:"width"`  // eg.: `512`
		Height uint64  `json:"height"` // eg.: `512`
		Ratio  float32 `json:"ratio"`  // eg.: `0.9058`
		URL    string  `json:"url"`    // eg.: `https://api.tinify.com/output/foobar`
	}

	// Result is compression result object with some additional properties.
	Result struct {
		Input            Input         `json:"input"`
		Output           Output        `json:"output"`
		Error            *string       `json:"error"`
		Message          *string       `json:"message"`
		CompressionCount uint64        // used quota value
		Compressed       io.ReadCloser // IMPORTANT: must be closed on receiver side
	}
)

var ErrCompressionCountHeaderNotFound = errors.New("header \"Compression-Count\" was not found in HTTP response")

// NewClient creates new `tinypng.com` API client instance.
func NewClient(config ClientConfig) *Client {
	return &Client{
		apiKey: config.APIKey,
		httpClient: &http.Client{
			Timeout: config.RequestTimeout, // Set request timeout
		},
	}
}

// Compress takes image body and compress it using `tinypng.com`. You should do not forget to close `result.Compressed`.
func (c *Client) Compress(ctx context.Context, body io.Reader) (*Result, error) {
	sentResponse, sentErr := c.sendImage(ctx, body)
	if sentErr != nil {
		return nil, sentErr
	}

	defer sentResponse.Body.Close()

	var result = Result{}

	// read response json
	if err := json.NewDecoder(sentResponse.Body).Decode(&result); err != nil {
		return nil, err
	}

	// making sure that error is missing in response
	if result.Error != nil {
		var details = ""

		if result.Error != nil {
			details += *result.Error
		}

		if result.Message != nil {
			details += " (" + strings.Trim(*result.Message, ". ") + ")"
		}

		return nil, errors.New("tinypng.com: " + details)
	}

	// extract `compression-count` value
	if count, err := c.extractCompressionCountFromResponse(sentResponse); err == nil {
		result.CompressionCount = count
	}

	compressed, err := c.downloadImage(ctx, result.Output.URL)
	if err != nil {
		return nil, err
	}

	// attach compressed content into result
	result.Compressed = compressed

	return &result, nil
}

// GetCompressionCount returns used quota value.
func (c *Client) GetCompressionCount(ctx context.Context) (uint64, error) {
	// If you know better way for getting current quota usage - please, make an issue in current repository
	resp, err := c.sendImage(ctx, nil)
	if err != nil {
		return 0, err
	}

	_ = resp.Body.Close()

	return c.extractCompressionCountFromResponse(resp)
}

// extractCompressionCountFromResponse extracts `compression-count` value from HTTP response.
func (c *Client) extractCompressionCountFromResponse(resp *http.Response) (uint64, error) {
	const headerName string = "Compression-Count"

	if val, ok := resp.Header[headerName]; ok {
		count, err := strconv.ParseUint(val[0], 10, 32)
		if err == nil {
			return count, nil
		}

		return 0, err
	}

	return 0, ErrCompressionCountHeaderNotFound
}

// sendImage sends image to the remote server.
func (c *Client) sendImage(ctx context.Context, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, ENDPOINT, body)
	if err != nil {
		return nil, err
	}

	// setup request API key
	request.SetBasicAuth("api", c.apiKey)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// downloadImage downloads image by passed URL from remote server.
func (c *Client) downloadImage(ctx context.Context, url string) (io.ReadCloser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// setup request API key
	request.SetBasicAuth("api", c.apiKey)

	response, err := c.httpClient.Do(request) //nolint:bodyclose
	if err != nil {
		return nil, err
	}

	// FIXME do NOT return http response body (convert it into "something readable", use "some buffer" from function
	//       params or something else). Also we should copy response into memory (for processing "downloading errors"
	//       inside this function (not on caller side))

	return response.Body, nil
}

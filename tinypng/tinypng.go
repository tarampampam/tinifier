// `tinypng.com` API client implementation.
package tinypng

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

// ENDPOINT is `tinypng.com` API endpoint. Docs can be found here: <https://tinypng.com/developers/reference>
const ENDPOINT string = "https://api.tinify.com/shrink"

// Client describes `tinypng.com` API client
type Client struct {
	apiKey     string
	httpClient *http.Client
	json       jsoniter.API
}

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

// NewClient creates new `tinypng.com` API client instance.
func NewClient(apiKey string, requestTimeout time.Duration) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: requestTimeout, // Set request timeout
		},
		json: jsoniter.ConfigFastest,
	}
}

// Compress takes image body and compress it using `tinypng.com`. You should do not forget to close `result.Compressed`.
func (c *Client) Compress(ctx context.Context, body io.Reader) (*Result, error) {
	sentResponse, sentErr := c.sendImage(ctx, body)
	if sentErr != nil {
		return nil, sentErr
	}

	defer func() {
		_ = sentResponse.Body.Close()
	}()

	var result = Result{}

	// read response json
	if err := c.json.NewDecoder(sentResponse.Body).Decode(&result); err != nil {
		return nil, err
	}

	// making sure that error is missing in response
	if result.Error != nil {
		var details = ""
		if result.Message != nil {
			details = " (" + strings.Trim(*result.Message, ". ") + ")"
		}

		return nil, errors.New("tinypng.com: " + *result.Error + details)
	}

	// extract `compression-count` value
	if count, countErr := c.extractCompressionCountFromResponse(sentResponse); countErr == nil {
		result.CompressionCount = count
	}

	compressed, downloadingErr := c.downloadImage(ctx, result.Output.URL)
	if downloadingErr != nil {
		return nil, downloadingErr
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

	return 0, fmt.Errorf("header %s was not found in HTTP response", headerName)
}

// sendImage sends image to the remote server.
func (c *Client) sendImage(ctx context.Context, body io.Reader) (*http.Response, error) {
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, ENDPOINT, body)
	if requestErr != nil {
		return nil, requestErr
	}

	// setup request API key
	request.SetBasicAuth("api", c.apiKey)

	response, responseErr := c.httpClient.Do(request)
	if responseErr != nil {
		return nil, responseErr
	}

	return response, nil
}

// downloadImage downloads image by passed URL from remote server.
func (c *Client) downloadImage(ctx context.Context, url string) (io.ReadCloser, error) {
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if requestErr != nil {
		return nil, requestErr
	}

	// setup request API key
	request.SetBasicAuth("api", c.apiKey)

	response, responseErr := c.httpClient.Do(request) //nolint:bodyclose
	if responseErr != nil {
		return nil, responseErr
	}

	return response.Body, nil
}

// Package tinypng is `tinypng.com` API client implementation.
package tinypng

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ENDPOINT is `tinypng.com` API endpoint. Docs can be found here: <https://tinypng.com/developers/reference>
const ENDPOINT string = "https://api.tinify.com/shrink"

type (
	// Client describes `tinypng.com` API client
	Client struct {
		mu         sync.Mutex
		apiKey     string
		httpClient *http.Client
	}

	ClientConfig struct {
		APIKey         string
		RequestTimeout time.Duration
	}
)

type (
	// Result is compression result object with some additional properties.
	Result struct {
		Input            Input   `json:"input"`
		Output           Output  `json:"output"`
		Error            *string `json:"error"`
		Message          *string `json:"message"`
		CompressionCount uint64  // used quota value
		Compressed       []byte
	}

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
)

const errorsPrefix = "tinypng.com: "

var (
	ErrCompressionCountHeaderNotFound = errors.New(errorsPrefix + "HTTP header \"Compression-Count\" was not found")
	ErrTooManyRequests                = errors.New(errorsPrefix + "too many requests (limit has been exceeded)")
	ErrUnauthorized                   = errors.New(errorsPrefix + "unauthorized (invalid credentials)")
)

// NewClient creates new `tinypng.com` API client instance.
func NewClient(config ClientConfig) *Client {
	return &Client{
		apiKey: config.APIKey,
		httpClient: &http.Client{
			Timeout: config.RequestTimeout, // Set request timeout
		},
	}
}

// SetAPIKey updates client API key.
func (c *Client) SetAPIKey(key string) {
	c.mu.Lock()
	c.apiKey = key
	c.mu.Unlock()
}

// Compress takes image body and compress it using `tinypng.com`. You should do not forget to close `result.Compressed`.
func (c *Client) Compress(ctx context.Context, body io.Reader) (*Result, error) {
	sentResponse, sentErr := c.sendImage(ctx, body)
	if sentErr != nil {
		return nil, sentErr
	}

	defer sentResponse.Body.Close()

	switch sentResponse.StatusCode {
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized

	case http.StatusTooManyRequests:
		return nil, ErrTooManyRequests
	}

	var result = Result{}

	// read response json
	if err := json.NewDecoder(sentResponse.Body).Decode(&result); err != nil {
		return nil, err
	}

	// making sure that error is missing in response
	if result.Error != nil {
		return nil, c.formatResponseError(result)
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

func (c *Client) formatResponseError(result Result) error {
	var details string

	if result.Error != nil {
		details += *result.Error
	}

	if result.Message != nil {
		details += " (" + strings.Trim(*result.Message, ". ") + ")"
	}

	if details == "" {
		details = "error does not contains information"
	}

	return errors.New(errorsPrefix + details)
}

// GetCompressionCount returns used quota value.
func (c *Client) GetCompressionCount(ctx context.Context) (uint64, error) {
	// If you know better way for getting current quota usage - please, make an issue in current repository
	resp, err := c.sendImage(ctx, nil)
	if err != nil {
		return 0, err
	}

	_ = resp.Body.Close()

	count, err := c.extractCompressionCountFromResponse(resp)

	if err != nil {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return 0, ErrUnauthorized

		case http.StatusTooManyRequests:
			return 0, ErrTooManyRequests
		}
	}

	return count, err
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
func (c *Client) downloadImage(ctx context.Context, url string) ([]byte, error) {
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
	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return content, nil
}

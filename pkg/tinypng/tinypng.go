// Package tinypng is `tinypng.com` API methods implementation.
package tinypng

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const (
	shrinkEndpoint = "https://api.tinify.com/shrink"
	errorsPrefix   = "tinypng.com: "
)

var (
	ErrTooManyRequests = errors.New(errorsPrefix + "too many requests (limit has been exceeded)")
	ErrUnauthorized    = errors.New(errorsPrefix + "unauthorized (invalid credentials)")
	ErrBadRequest      = errors.New(errorsPrefix + "bad request (empty file or wrong format)")
)

type (
	remoteError struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

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

func Compress(ctx context.Context, httpClient *http.Client, apiKey string, image io.Reader) (*CompressionResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, shrinkEndpoint, image)
	if err != nil {
		return nil, err
	}

	setupRequestAPIKey(req, apiKey)

	resp, err := httpClient.Do(req)
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
			return nil, errors.New(errorsPrefix + "response decoding failed: " + err.Error())
		}

		if count, err := extractCompressionCount(resp.Header); err == nil { // error will be ignored
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
			var e remoteError

			if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
				return nil, errors.New(errorsPrefix + "error decoding failed: " + err.Error())
			}

			return nil, errors.New(errorsPrefix + e.Error + " (" + strings.Trim(e.Message, ". ") + ")")
		}

	default:
		return nil, errors.New(errorsPrefix + "unexpected HTTP response code")
	}
}

func setupRequestAPIKey(request *http.Request, apiKey string) {
	request.SetBasicAuth("api", apiKey)
}

const compressionCountHeaderName = "Compression-Count"

// extractCompressionCount extracts `compression-count` header value from HTTP response.
func extractCompressionCount(headers http.Header) (uint64, error) {
	if val, ok := headers[compressionCountHeaderName]; ok {
		count, err := strconv.ParseUint(val[0], 10, 64)
		if err == nil {
			return count, nil
		}

		return 0, err
	}

	return 0, errors.New(errorsPrefix + "HTTP header \"Compression-Count\" was not found")
}

func CompressionCount(ctx context.Context, httpClient *http.Client, apiKey string) (uint64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, shrinkEndpoint, nil)
	if err != nil {
		return 0, err
	}

	setupRequestAPIKey(req, apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}

	_ = resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return 0, ErrUnauthorized

	default:
		return extractCompressionCount(resp.Header)
	}
}

func DownloadImage(ctx context.Context, httpClient *http.Client, apiKey, url string) (int64, io.Reader, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return -1, nil, err
	}

	setupRequestAPIKey(req, apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return -1, nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var bufferSize int64 // is used for content buffer pre-allocation
	if resp.ContentLength > 0 {
		bufferSize = resp.ContentLength
	}

	var buf = bytes.NewBuffer(make([]byte, 0, bufferSize))

	written, err := io.Copy(buf, resp.Body)
	if err != nil {
		return -1, nil, err
	}

	return written, buf, nil
}

//type (
//	// Client describes `tinypng.com` API client
//	Client struct {
//		mu         sync.Mutex
//		apiKey     string
//		httpClient *http.Client
//	}
//
//	ClientConfig struct {
//		APIKey         string
//		RequestTimeout time.Duration
//	}
//)
//
//type (
//	// Result is compression result object with some additional properties.
//	Result struct {
//		Input            Input   `json:"input"`
//		Output           Output  `json:"output"`
//		Error            *string `json:"error"`
//		Message          *string `json:"message"`
//		CompressionCount uint64  // used quota value
//		Compressed       []byte
//	}
//
//	// Input follows `input` response object structure.
//	Input struct {
//		Size uint64 `json:"size"` // eg.: `5851`
//		Type string `json:"type"` // eg.: `image/png`
//	}
//
//	// Output follows `output` response object structure.
//	Output struct {
//		Size   uint64  `json:"size"`   // eg.: `5851`
//		Type   string  `json:"type"`   // eg.: `image/png`
//		Width  uint64  `json:"width"`  // eg.: `512`
//		Height uint64  `json:"height"` // eg.: `512`
//		Ratio  float32 `json:"ratio"`  // eg.: `0.9058`
//		URL    string  `json:"url"`    // eg.: `https://api.tinify.com/output/foobar`
//	}
//)
//
//const errorsPrefix = "tinypng.com: "
//
//var (
//	ErrCompressionCountHeaderNotFound = errors.New(errorsPrefix + "HTTP header \"Compression-Count\" was not found")
//	ErrTooManyRequests                = errors.New(errorsPrefix + "too many requests (limit has been exceeded)")
//	ErrUnauthorized                   = errors.New(errorsPrefix + "unauthorized (invalid credentials)")
//	ErrBadRequest                     = errors.New(errorsPrefix + "bad request (empty file or wrong format)")
//)
//
//// NewClient creates new `tinypng.com` API client instance.
//func NewClient(config ClientConfig) *Client {
//	return &Client{
//		apiKey: config.APIKey,
//		httpClient: &http.Client{
//			Timeout: config.RequestTimeout, // Set request timeout
//		},
//	}
//}
//
//// SetAPIKey updates client API key.
//func (c *Client) SetAPIKey(key string) {
//	c.mu.Lock()
//	c.apiKey = key
//	c.mu.Unlock()
//}
//
//// Compress takes image body and compress it using `tinypng.com`. You should do not forget to close `result.Compressed`.
//func (c *Client) Compress(ctx context.Context, body io.Reader) (*Result, error) {
//	sentResponse, sentErr := c.sendImage(ctx, body)
//	if sentErr != nil {
//		return nil, sentErr
//	}
//
//	defer sentResponse.Body.Close()
//
//	switch sentResponse.StatusCode {
//	case http.StatusUnauthorized:
//		return nil, ErrUnauthorized
//
//	case http.StatusTooManyRequests:
//		return nil, ErrTooManyRequests
//
//	case http.StatusBadRequest:
//		return nil, ErrBadRequest
//	}
//
//	var result = Result{}
//
//	// read response json
//	if err := json.NewDecoder(sentResponse.Body).Decode(&result); err != nil {
//		return nil, err
//	}
//
//	// making sure that error is missing in response
//	if result.Error != nil {
//		return nil, c.formatResponseError(result)
//	}
//
//	// extract `compression-count` value
//	if count, err := c.extractCompressionCountFromResponse(sentResponse); err == nil {
//		result.CompressionCount = count
//	}
//
//	compressed, err := c.downloadImage(ctx, result.Output.URL)
//	if err != nil {
//		return nil, err
//	}
//
//	// attach compressed content into result
//	result.Compressed = compressed
//
//	return &result, nil
//}
//
//func (c *Client) formatResponseError(result Result) error {
//	var details string
//
//	if result.Error != nil {
//		details += *result.Error
//	}
//
//	if result.Message != nil {
//		details += " (" + strings.Trim(*result.Message, ". ") + ")"
//	}
//
//	if details == "" {
//		details = "error does not contains information"
//	}
//
//	return errors.New(errorsPrefix + details)
//}
//
//// GetCompressionCount returns used quota value.
//func (c *Client) GetCompressionCount(ctx context.Context) (uint64, error) {
//	// If you know better way for getting current quota usage - please, make an issue in current repository
//	resp, err := c.sendImage(ctx, nil)
//	if err != nil {
//		return 0, err
//	}
//
//	_ = resp.Body.Close()
//
//	count, err := c.extractCompressionCountFromResponse(resp)
//
//	if err != nil {
//		switch resp.StatusCode {
//		case http.StatusUnauthorized:
//			return 0, ErrUnauthorized
//
//		case http.StatusTooManyRequests:
//			return 0, ErrTooManyRequests
//		}
//	}
//
//	return count, err
//}
//
//// extractCompressionCountFromResponse extracts `compression-count` value from HTTP response.
//func (c *Client) extractCompressionCountFromResponse(resp *http.Response) (uint64, error) {
//	const headerName = "Compression-Count"
//
//	if val, ok := resp.Header[headerName]; ok {
//		count, err := strconv.ParseUint(val[0], 10, 64)
//		if err == nil {
//			return count, nil
//		}
//
//		return 0, err
//	}
//
//	return 0, ErrCompressionCountHeaderNotFound
//}
//
//// sendImage sends image to the remote server.
//func (c *Client) sendImage(ctx context.Context, body io.Reader) (*http.Response, error) {
//	request, err := http.NewRequestWithContext(ctx, http.MethodPost, Endpoint, body)
//	if err != nil {
//		return nil, err
//	}
//
//	// setup request API key
//	request.SetBasicAuth("api", c.apiKey)
//
//	response, err := c.httpClient.Do(request)
//	if err != nil {
//		return nil, err
//	}
//
//	return response, nil
//}
//
//// downloadImage by passed URL from remote server.
//func (c *Client) downloadImage(ctx context.Context, url string) ([]byte, error) {
//	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
//	if err != nil {
//		return nil, err
//	}
//
//	// setup request API key
//	request.SetBasicAuth("api", c.apiKey)
//
//	response, err := c.httpClient.Do(request)
//	if err != nil {
//		return nil, err
//	}
//	defer response.Body.Close()
//
//	content, err := ioutil.ReadAll(response.Body) // TODO(jetexe) return new buffer mb?
//	if err != nil {
//		return nil, err
//	}
//
//	return content, nil
//}

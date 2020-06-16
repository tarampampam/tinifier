package tinypng

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	jsoniter "github.com/json-iterator/go"
)

const ENDPOINT string = "https://api.tinify.com/shrink"

type Client struct {
	apiKey     string
	httpClient *http.Client
	json       jsoniter.API
}

type (
	Input struct {
		Size uint64 `json:"size"` // eg.: `5851`
		Type string `json:"type"` // eg.: `image/png`
	}

	Output struct {
		Size   uint64  `json:"size"`   // eg.: `5851`
		Type   string  `json:"type"`   // eg.: `image/png`
		Width  uint64  `json:"width"`  // eg.: `512`
		Height uint64  `json:"height"` // eg.: `512`
		Ratio  float32 `json:"ratio"`  // eg.: `0.9058`
		URL    string  `json:"url"`    // eg.: `https://api.tinify.com/output/foobar`
	}

	Result struct {
		Input            Input   `json:"input"`
		Output           Output  `json:"output"`
		Error            *string `json:"error"`
		CompressionCount uint64  // used quota value
		Compressed       io.ReadCloser
	}
)

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
func (c *Client) Compress(body io.Reader) (*Result, error) {
	sentResponse, sentErr := c.sendImage(body)
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
		return nil, errors.New(*result.Error)
	}

	// extract `compression-count` value
	if count, countErr := c.extractCompressionCountFromResponse(sentResponse); countErr == nil {
		result.CompressionCount = count
	}

	compressed, downloadingErr := c.downloadImage(result.Output.URL)
	if downloadingErr != nil {
		return nil, downloadingErr
	}

	// attach compressed content into result
	result.Compressed = compressed

	return &result, nil
}

func (c *Client) GetCompressionCount() (uint64, error) {
	// If you know better way for getting current quota usage - please, make an issue in current repository
	resp, err := c.sendImage(nil)
	if err != nil {
		return 0, err
	}

	_ = resp.Body.Close()

	return c.extractCompressionCountFromResponse(resp)
}

// extract `compression-count` value
func (c *Client) extractCompressionCountFromResponse(resp *http.Response) (uint64, error) {
	const headerName string = "Compression-Count"

	if val, ok := resp.Header[headerName]; ok {
		if count, err := strconv.ParseUint(val[0], 10, 32); err == nil {
			return count, nil
		} else {
			return 0, err
		}
	}

	return 0, fmt.Errorf("header %s was not found in HTTP response", headerName)
}

func (c *Client) sendImage(body io.Reader) (*http.Response, error) {
	request, requestErr := http.NewRequest(http.MethodPost, ENDPOINT, body)
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

func (c *Client) downloadImage(url string) (io.ReadCloser, error) {
	request, requestErr := http.NewRequest(http.MethodGet, url, nil)
	if requestErr != nil {
		return nil, requestErr
	}

	// setup request API key
	request.SetBasicAuth("api", c.apiKey)

	response, responseErr := c.httpClient.Do(request)
	if responseErr != nil {
		return nil, responseErr
	}

	return response.Body, nil
}

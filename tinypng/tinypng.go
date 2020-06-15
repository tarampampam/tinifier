package tinypng

import (
	"errors"
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
	if val, ok := sentResponse.Header["Compression-Count"]; ok {
		if count, err := strconv.ParseUint(val[0], 10, 32); err == nil {
			result.CompressionCount = count
		}
	}

	compressed, downloadingErr := c.downloadImage(result.Output.URL)
	if downloadingErr != nil {
		return nil, downloadingErr
	}

	// attach compressed content into result
	result.Compressed = compressed

	return &result, nil
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

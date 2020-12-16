package tinypng

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	ctx context.Context

	mu      sync.Mutex
	apiKey  string
	timeout time.Duration // default HTTP request timeout
}

type ClientOption func(*Client)

func WithContext(ctx context.Context) ClientOption {
	return func(c *Client) { c.ctx = ctx }
}

func WithDefaultTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) { c.timeout = timeout }
}

func NewClient(apiKey string, options ...ClientOption) *Client {
	const defaultTimeout = time.Second * 60

	c := &Client{
		apiKey:  apiKey,
		timeout: defaultTimeout,
	}

	for i := 0; i < len(options); i++ {
		options[i](c)
	}

	if c.ctx == nil {
		c.ctx = context.Background()
	}

	return c
}

func (c *Client) SetAPIKey(key string) {
	c.mu.Lock()
	c.apiKey = key
	c.mu.Unlock()
}

func (c *Client) Compress(source io.Reader, timeouts ...time.Duration) (*CompressionResult, io.Reader, error) {
	var t time.Duration

	if len(timeouts) > 0 {
		t = timeouts[0] // first timeout for compressing operation
	} else {
		t = c.timeout // default
	}

	httpClient := &http.Client{Timeout: t}

	result, err := Compress(c.ctx, httpClient, c.apiKey, source)
	if err != nil {
		return nil, nil, err
	}

	if len(timeouts) > 1 {
		httpClient.Timeout = timeouts[1] // second - for downloading
	}

	_, compressed, err := DownloadImage(c.ctx, httpClient, c.apiKey, result.Output.URL)
	if err != nil {
		return nil, nil, err
	}

	return result, compressed, nil
}

func (c *Client) GetCompressionCount(timeout ...time.Duration) (uint64, error) {
	var t time.Duration

	if len(timeout) > 0 {
		t = timeout[0]
	} else {
		t = c.timeout // default
	}

	httpClient := &http.Client{Timeout: t}

	return CompressionCount(c.ctx, httpClient, c.apiKey)
}

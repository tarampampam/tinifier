package tinypng

import (
	"context"
	"time"
)

// ClientOption allows to setup some internal client properties from outside.
type ClientOption func(*Client)

// WithContext setups client context.
func WithContext(ctx context.Context) ClientOption {
	return func(c *Client) { c.ctx = ctx }
}

// WithDefaultTimeout setups default HTTP request timeouts.
func WithDefaultTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) { c.defaultTimeout = timeout }
}

// WithHTTPClient setups allows to pass custom HTTP client implementation.
func WithHTTPClient(httpClient httpClient) ClientOption {
	return func(c *Client) { c.httpClient = httpClient }
}

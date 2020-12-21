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

// WithContext setups default HTTP request timeouts.
func WithDefaultTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) { c.defaultTimeout = timeout }
}

// WithContext setups allows to pass custom HTTP client implementation.
func WithHTTPClient(httpClient httpClient) ClientOption {
	return func(c *Client) { c.httpClient = httpClient }
}

package tinypng

// ClientOption allows to set up some internal client properties from outside.
type ClientOption func(*Client)

// WithHTTPClient setups allows to pass custom HTTP client implementation.
func WithHTTPClient(httpClient httpClient) ClientOption {
	return func(c *Client) { c.httpClient = httpClient }
}

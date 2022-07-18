package tinypng

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ExampleWithHTTPClient() {
	NewClient("YOUR-API-KEY", WithHTTPClient(&http.Client{Timeout: time.Second * 5}))
}

func TestWithHTTPClient(t *testing.T) {
	hc := new(http.Client)

	c := NewClient("", WithHTTPClient(hc))

	assert.Equal(t, hc, c.httpClient)
}

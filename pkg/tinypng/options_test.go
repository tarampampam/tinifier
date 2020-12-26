package tinypng

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ExampleWithContext() {
	NewClient("YOUR-API-KEY", WithContext(context.TODO()))
}

func TestWithContext(t *testing.T) {
	myCtx := context.TODO()

	c := NewClient("", WithContext(myCtx))

	assert.Same(t, myCtx, c.ctx)
}

func ExampleWithDefaultTimeout() {
	NewClient("YOUR-API-KEY", WithDefaultTimeout(time.Second*60))
}

func TestWithDefaultTimeout(t *testing.T) {
	d := time.Second * 1234

	c := NewClient("", WithDefaultTimeout(d))

	assert.Equal(t, d, c.defaultTimeout)
}

func ExampleWithHTTPClient() {
	NewClient("YOUR-API-KEY", WithHTTPClient(&http.Client{Timeout: time.Second * 5}))
}

func TestWithHTTPClient(t *testing.T) {
	hc := new(http.Client)

	c := NewClient("", WithHTTPClient(hc))

	assert.Equal(t, hc, c.httpClient)
}

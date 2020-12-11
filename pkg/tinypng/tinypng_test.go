package tinypng

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClientConstants(t *testing.T) {
	assert.Equal(t, "https://api.tinify.com/shrink", ENDPOINT)
}

func TestNewClient(t *testing.T) {
	const requestTimeout = time.Second * 123

	fakeAPIKey := genRandAPIkey()
	c := NewClient(ClientConfig{fakeAPIKey, requestTimeout})

	assert.Equal(t, fakeAPIKey, c.apiKey)
	assert.Equal(t, requestTimeout, c.httpClient.Timeout)
}

func TestClient_CompressNilBodyReturnsInputMissingError(t *testing.T) {
	fakeAPIKey := genRandAPIkey()
	c := NewClient(ClientConfig{fakeAPIKey, time.Second * 10})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check requested URI path
		assert.Equal(t, "/shrink", r.URL.Path)
		// check auth header
		assert.Equal(t, generateAuthHeaderValue(fakeAPIKey), r.Header.Get("Authorization"))

		w.Header().Add("Compression-Count", "123")
		_, _ = w.Write([]byte(`{"error":"InputMissing","message":"Input file is empty."}`))
	})

	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	c.httpClient = httpClient

	res, err := c.Compress(context.Background(), nil)

	assert.Nil(t, res)
	assert.Error(t, err, "tinypng.com: InputMissing (Input file is empty)")
}

func TestClient_CompressRealImageSuccess(t *testing.T) {
	fakeAPIKey := genRandAPIkey()
	c := NewClient(ClientConfig{fakeAPIKey, time.Second * 10})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/shrink":
			// check auth header
			assert.Equal(t, generateAuthHeaderValue(fakeAPIKey), r.Header.Get("Authorization"))

			// check passed content
			file, _ := ioutil.ReadFile("./image_test.png")
			reqBody, _ := ioutil.ReadAll(r.Body)
			assert.Equal(t, file, reqBody)

			w.Header().Add("Compression-Count", "666")
			_, _ = w.Write([]byte(`{
									"input":{
										"size":4633,
										"type":"image/png"
									},
									"output":{
										"size":1636,
										"type":"image/png",
										"width":128,
										"height":128,
										"ratio":0.3531,
										"url":"https://api.tinify.com/output/someRandomResultImageHash"
									}
								}`))

		case "/output/someRandomResultImageHash":
			file, _ := ioutil.ReadFile("./image_compressed_test.png")

			// check auth header
			assert.Equal(t, generateAuthHeaderValue(fakeAPIKey), r.Header.Get("Authorization"))

			_, _ = w.Write(file)

		default:
			t.Fatal("unexpected request")
		}
	})

	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	c.httpClient = httpClient

	file, fileErr := ioutil.ReadFile("./image_test.png")
	assert.Nil(t, fileErr)

	res, err := c.Compress(context.Background(), bytes.NewBuffer(file))

	assert.NotNil(t, res)
	assert.Nil(t, err)

	// check compressed image content
	compressed, _ := ioutil.ReadFile("./image_compressed_test.png")
	respBody, _ := ioutil.ReadAll(res.Compressed)
	assert.Equal(t, compressed, respBody)

	// check result values
	assert.Equal(t, uint64(666), res.CompressionCount)
	assert.Equal(t, uint64(4633), res.Input.Size)
	assert.Equal(t, "image/png", res.Input.Type)
	assert.Equal(t, uint64(1636), res.Output.Size)
	assert.Equal(t, "image/png", res.Output.Type)
	assert.Equal(t, uint64(128), res.Output.Width)
	assert.Equal(t, uint64(128), res.Output.Height)
	assert.Equal(t, float32(0.3531), res.Output.Ratio)
	assert.Equal(t, "https://api.tinify.com/output/someRandomResultImageHash", res.Output.URL)
	assert.Nil(t, res.Message)
	assert.Nil(t, res.Error)

	assert.Nil(t, res.Compressed.Close())
}

func TestClient_GetCompressionCount(t *testing.T) {
	fakeAPIKey := genRandAPIkey()

	c := NewClient(ClientConfig{fakeAPIKey, time.Second * 10})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check requested URI path
		assert.Equal(t, "/shrink", r.URL.Path)
		// check auth header
		assert.Equal(t, generateAuthHeaderValue(fakeAPIKey), r.Header.Get("Authorization"))
		w.Header().Add("Compression-Count", "444")

		_, _ = w.Write([]byte(`{"error":"InputMissing","message":"Input file is empty."}`))
	})

	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	c.httpClient = httpClient

	res, err := c.GetCompressionCount(context.Background())

	assert.Equal(t, uint64(444), res)
	assert.Nil(t, err)
}

func generateAuthHeaderValue(apiKey string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte("api:"+apiKey))
}

func testingHTTPClient(handler http.Handler) (*http.Client, func()) {
	s := httptest.NewTLSServer(handler)

	cli := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec
			},
		},
	}

	return cli, s.Close
}

func genRandAPIkey() string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	l, b := len(letterBytes), make([]byte, 16)

	for i := range b {
		b[i] = letterBytes[rand.Intn(l)] //nolint:gosec
	}

	return string(b)
}

package tinypng

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const apiKeyForTests = "foo"

func TestClient_CompressNilBody(t *testing.T) {
	c := NewClient(apiKeyForTests, time.Second*10)

	//res, err := c.Compress(context.Background(), nil)
	//
	//assert.Nil(t, err)
	//
	//fmt.Println(res)

	assert.Equal(t, apiKeyForTests, c.apiKey)
}

func TestClient_CompressImage(t *testing.T) {
	c := NewClient(apiKeyForTests, time.Second*10)

	//file, fileErr := ioutil.ReadFile("./image_test.png")
	//assert.Nil(t, fileErr)
	//
	//res, err := c.Compress(context.Background(), bytes.NewBuffer(file))
	//
	//assert.Nil(t, err)
	//
	//fmt.Println(res)

	assert.Equal(t, apiKeyForTests, c.apiKey)
}

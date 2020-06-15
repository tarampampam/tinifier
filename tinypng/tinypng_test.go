package tinypng

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
	"time"
)

const apiKeyForTests = "foo"

func TestClient_CompressNilBody(t *testing.T) {
	c := NewClient(apiKeyForTests, time.Second*10)

	res, err := c.Compress(nil)

	assert.Nil(t, err)

	fmt.Println(res)
}

func TestClient_CompressImage(t *testing.T) {
	c := NewClient(apiKeyForTests, time.Second*10)

	file, fileErr := ioutil.ReadFile("./image_test.png")
	assert.Nil(t, fileErr)

	res, err := c.Compress(bytes.NewBuffer(file))

	assert.Nil(t, err)

	fmt.Println(res)
}

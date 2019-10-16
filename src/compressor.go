package main

import (
	"errors"
	tinypng "github.com/gwpp/tinify-go/tinify"
	"net/http"
	"strconv"
)

type Compressor struct {}

var compressor = Compressor{}

// Set tinypng.com service API key.
func (c *Compressor) SetKey(key string) {
	tinypng.SetKey(key)
}

// Get tinypng.com used quota value.
func (c *Compressor) GetQuotaUsage() (int, error) {
	// If you know better way for getting current quota usage - please, make an issue in current repository
	if response, err := tinypng.GetClient().Request(http.MethodPost, "/shrink", nil); err == nil {
		if val, ok := response.Header["Compression-Count"]; ok {
			if count, err := strconv.Atoi(val[0]); err == nil {
				return count, nil
			} else {
				return -1, err
			}
		} else {
			return -1, errors.New("header 'Compression-Count' does not exists in service response")
		}
	} else {
		return -1, err
	}
}

// Compress image content using tinypng.com service and save result to the file.
func (c *Compressor) CompressBuffer(buffer *[]byte, out string) error {
	if source, err := tinypng.FromBuffer(*buffer); err != nil {
		return err
	} else {
		if err := source.ToFile(out); err != nil {
			return err
		}
	}

	return nil
}

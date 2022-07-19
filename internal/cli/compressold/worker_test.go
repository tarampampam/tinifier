package compressold

// import (
// 	"bytes"
// 	"context"
// 	"errors"
// 	"io/ioutil"
// 	"net/http"
// 	"os"
// 	"path"
// 	"path/filepath"
// 	"testing"
// 	"time"
//
// 	"github.com/tarampampam/tinifier/v4/internal/pkg/pool"
//
// 	"github.com/tarampampam/tinifier/v4/internal/keys"
// 	"github.com/tarampampam/tinifier/v4/pkg/tinypng"
//
// 	"github.com/stretchr/testify/assert"
// 	"go.uber.org/zap"
// )
//
// func TestWorker_PreTaskRun(t *testing.T) {
// 	w := newWorker(zap.NewNop(), new(keys.Keeper), 0, time.Duration(0))
//
// 	assert.NotPanics(t, func() { w.PreTaskRun(pool.Task{}) })
// }
//
// type httpClientFunc func(*http.Request) (*http.Response, error)
//
// func (f httpClientFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }
//
// func TestWorker_UploadSuccessful(t *testing.T) {
// 	file, err := os.OpenFile("./testdata/image.png", os.O_RDONLY, 0)
// 	assert.NoError(t, err)
//
// 	fileContent, err := ioutil.ReadAll(file)
// 	file.Close()
// 	assert.NoError(t, err)
//
// 	keeper := keys.NewKeeper()
// 	assert.NoError(t, keeper.Add("foo-key", "bar-key"))
//
// 	w := newWorker(zap.NewNop(), &keeper, 2, time.Duration(0))
//
// 	var httpDoCallCounter uint
//
// 	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
// 		httpDoCallCounter++
//
// 		body, readErr := ioutil.ReadAll(req.Body)
// 		assert.NoError(t, readErr)
// 		assert.Equal(t, fileContent, body)
//
// 		if httpDoCallCounter > 1 { // only second (and any next) call will be successful
// 			return &http.Response{
// 				StatusCode: http.StatusCreated,
// 				Header:     http.Header{"Compression-Count": {"123"}},
// 				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
// 					"input":{
// 						"size":4633,
// 						"type":"image/png"
// 					},
// 					"output":{
// 						"size":1636,
// 						"type":"image/png",
// 						"width":123,
// 						"height":321,
// 						"ratio":0.3531,
// 						"url":"https://example.com/foo"
// 					}
// 				}`))),
// 			}, nil
// 		}
//
// 		return nil, errors.New("fake http error")
// 	}
//
// 	w.tinyHTTPClient = httpMock // mock http client
//
// 	url, fi, err := w.Upload(context.Background(), "./testdata/image.png")
//
// 	assert.Equal(t, "https://example.com/foo", url)
// 	assert.NotNil(t, fi)
// 	assert.Equal(t, uint64(3836), fi.Size())
// 	assert.Equal(t, "image/png", fi.Type())
// 	assert.NoError(t, err)
// }
//
// func TestWorker_UploadWithoutKeyInKeeper(t *testing.T) {
// 	keeper := keys.NewKeeper()
//
// 	w := newWorker(zap.NewNop(), &keeper, 2, time.Duration(0))
//
// 	url, fi, err := w.Upload(context.Background(), "./testdata/image.png")
//
// 	assert.Empty(t, url)
// 	assert.Nil(t, fi)
// 	assert.EqualError(t, err, errNoAvailableAPIKey.Error())
// }
//
// func TestWorker_UploadInvalidKeysRemoval(t *testing.T) {
// 	keeper := keys.NewKeeper()
// 	assert.NoError(t, keeper.Add("aaa", "bbb"))
//
// 	_, err := keeper.Get()
// 	assert.NoError(t, err)
//
// 	w := newWorker(zap.NewNop(), &keeper, 2, time.Duration(0)) // <-- 2 attempts is important
//
// 	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
// 		return &http.Response{
// 			StatusCode: http.StatusTooManyRequests, // <-- important
// 			Header:     http.Header{},
// 			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
// 		}, nil
// 	}
//
// 	w.tinyHTTPClient = httpMock // mock http client
//
// 	url, fi, err := w.Upload(context.Background(), "./testdata/image.png")
//
// 	assert.Empty(t, url)
// 	assert.Nil(t, fi)
// 	assert.EqualError(t, err, tinypng.ErrTooManyRequests.Error())
//
// 	_, err = keeper.Get()
// 	assert.EqualError(t, err, keys.ErrKeyNotExists.Error())
// }
//
// func TestWorker_DownloadSuccessful(t *testing.T) {
// 	originalFile, err := os.OpenFile("./testdata/image.png", os.O_RDONLY, 0)
// 	assert.NoError(t, err)
// 	originalFileContent, err := ioutil.ReadAll(originalFile)
// 	originalFile.Close()
// 	assert.NoError(t, err)
//
// 	keeper := keys.NewKeeper()
// 	assert.NoError(t, keeper.Add("foo-key", "bar-key"))
//
// 	tmpDir, tmpDirErr := ioutil.TempDir("", "test-")
// 	assert.NoError(t, tmpDirErr)
//
// 	defer func(d string) { assert.NoError(t, os.RemoveAll(d)) }(tmpDir)
//
// 	w := newWorker(zap.NewNop(), &keeper, 2, time.Duration(0))
//
// 	var httpDoCallCounter uint
//
// 	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
// 		httpDoCallCounter++
//
// 		file, fileErr := os.OpenFile("./testdata/image.png", os.O_RDONLY, 0)
// 		assert.NoError(t, fileErr)
//
// 		if httpDoCallCounter > 1 { // only second (and any next) call will be successful
// 			return &http.Response{
// 				StatusCode: http.StatusOK,
// 				Header:     http.Header{"Compression-Count": {"123"}},
// 				Body:       file,
// 			}, nil
// 		}
//
// 		return nil, errors.New("fake http error")
// 	}
//
// 	w.tinyHTTPClient = httpMock // mock http client
//
// 	targetFilePath := path.Join(tmpDir, "foo.image")
// 	fi, err := w.Download(context.Background(), "https://example.com/foo", targetFilePath)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, fi)
// 	assert.Equal(t, uint64(3836), fi.Size())
// 	assert.Equal(t, "image/png", fi.Type())
//
// 	downloadedFile, err := os.OpenFile(targetFilePath, os.O_RDONLY, 0)
// 	assert.NoError(t, err)
//
// 	defer downloadedFile.Close()
//
// 	downloadedFileContent, err := ioutil.ReadAll(downloadedFile)
// 	assert.NoError(t, err)
//
// 	assert.Equal(t, originalFileContent, downloadedFileContent)
// }
//
// func TestWorker_DownloadWithoutKeyInKeeper(t *testing.T) {
// 	keeper := keys.NewKeeper()
//
// 	w := newWorker(zap.NewNop(), &keeper, 2, time.Duration(0))
//
// 	fi, err := w.Download(context.Background(), "https://example.com/foo", "/tmp/foo")
//
// 	assert.Nil(t, fi)
// 	assert.EqualError(t, err, errNoAvailableAPIKey.Error())
// }
//
// func TestWorker_DownloadInvalidKeysRemoval(t *testing.T) {
// 	keeper := keys.NewKeeper()
// 	assert.NoError(t, keeper.Add("aaa", "bbb"))
//
// 	_, err := keeper.Get()
// 	assert.NoError(t, err)
//
// 	w := newWorker(zap.NewNop(), &keeper, 2, time.Duration(0)) // <-- 2 attempts is important
//
// 	var httpMock httpClientFunc = func(req *http.Request) (*http.Response, error) {
// 		return &http.Response{
// 			StatusCode: http.StatusTooManyRequests, // <-- important
// 			Header:     http.Header{},
// 			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
// 		}, nil
// 	}
//
// 	w.tinyHTTPClient = httpMock // mock http client
//
// 	fi, err := w.Download(context.Background(), "https://example.com/foo", "/tmp/foo")
//
// 	assert.Nil(t, fi)
// 	assert.EqualError(t, err, tinypng.ErrTooManyRequests.Error())
//
// 	_, err = keeper.Get()
// 	assert.EqualError(t, err, keys.ErrKeyNotExists.Error())
// }
//
// func TestWorker_CopyContent(t *testing.T) {
// 	tmpDir, tmpDirErr := ioutil.TempDir("", "test-")
// 	assert.NoError(t, tmpDirErr)
//
// 	defer func(d string) { assert.NoError(t, os.RemoveAll(d)) }(tmpDir)
//
// 	fromFilePath, toFilePath := filepath.Join(tmpDir, "from"), filepath.Join(tmpDir, "to")
//
// 	testContent := []byte{1, 2, 3}
//
// 	fromFile, createErr := os.Create(fromFilePath)
// 	assert.NoError(t, createErr)
//
// 	_, fileWritingErr := fromFile.Write(testContent)
// 	assert.NoError(t, fileWritingErr)
// 	assert.NoError(t, fromFile.Close())
//
// 	toFile, createErr := os.Create(toFilePath)
// 	assert.NoError(t, createErr)
// 	assert.NoError(t, toFile.Close())
//
// 	w := newWorker(zap.NewNop(), new(keys.Keeper), 0, time.Duration(0))
//
// 	assert.NoError(t, w.CopyContent(fromFilePath, toFilePath))
//
// 	toFile, err := os.OpenFile(toFilePath, os.O_RDONLY, 0)
// 	assert.NoError(t, err)
//
// 	content, _ := ioutil.ReadAll(toFile)
// 	toFile.Close()
//
// 	assert.Equal(t, testContent, content)
// }
//
// func TestWorker_RemoveFile(t *testing.T) {
// 	tmpDir, tmpDirErr := ioutil.TempDir("", "test-")
// 	assert.NoError(t, tmpDirErr)
//
// 	defer func(d string) { assert.NoError(t, os.RemoveAll(d)) }(tmpDir)
//
// 	filePath := filepath.Join(tmpDir, "test_file")
//
// 	file, createErr := os.Create(filePath)
// 	assert.NoError(t, createErr)
// 	assert.NoError(t, file.Close())
//
// 	w := newWorker(zap.NewNop(), new(keys.Keeper), 0, time.Duration(0))
//
// 	assert.FileExists(t, filePath)
// 	assert.NoError(t, w.RemoveFile(filePath))
// 	assert.NoFileExists(t, filePath)
// }

package pipeline

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompressingPipeline_RunSimple(t *testing.T) {
	var successCount, errorsCount, preWorkerRunCounter uint32

	var comp CompressorFunc = func(task Task) (*TaskResult, error) {
		// emulate error when task filepath is "bar"
		if task.FilePath == "bar" {
			return nil, errors.New("fake error")
		}

		return &TaskResult{FilePath: task.FilePath}, nil
	}

	onResult := func(res TaskResult) {
		assert.Contains(t, []string{"foo", "baz"}, res.FilePath)

		atomic.AddUint32(&successCount, 1)
	}

	onError := func(err TaskError) {
		assert.Equal(t, "bar", err.Task.FilePath)
		assert.Equal(t, "fake error", err.Error.Error())

		atomic.AddUint32(&errorsCount, 1)
	}

	pipe := NewPipeline(
		context.Background(),
		[]Task{{FilePath: "foo"}, {FilePath: "bar"}, {FilePath: "baz"}},
		comp,
		onResult,
		onError,
	)

	pipe.PreWorkerRun = func(task Task) { atomic.AddUint32(&preWorkerRunCounter, 1) }

	<-pipe.Run(2)

	assert.Equal(t, uint32(3), preWorkerRunCounter)
	assert.Equal(t, uint32(2), successCount)
	assert.Equal(t, uint32(1), errorsCount)
}

func TestCompressingPipeline_RunWithCxtCancellation(t *testing.T) {
	var successCount, errorsCount, preWorkerRunCounter uint32

	var comp CompressorFunc = func(task Task) (*TaskResult, error) {
		t.Error("should not be executed")

		return &TaskResult{FilePath: task.FilePath}, nil
	}

	var (
		onResult    = func(TaskResult) { atomic.AddUint32(&successCount, 1) }
		onError     = func(TaskError) { atomic.AddUint32(&errorsCount, 1) }
		ctx, cancel = context.WithCancel(context.Background())
	)

	pipe := NewPipeline(
		ctx,
		[]Task{{FilePath: "foo"}, {FilePath: "bar"}},
		comp,
		onResult,
		onError,
	)

	pipe.PreWorkerRun = func(task Task) { atomic.AddUint32(&preWorkerRunCounter, 1) }

	cancel() // all tasks must be canceled

	<-pipe.Run(2)

	assert.Equal(t, uint32(0), successCount)
	assert.Equal(t, uint32(0), errorsCount)
	assert.Equal(t, uint32(0), preWorkerRunCounter)
}

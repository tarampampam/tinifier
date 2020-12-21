package pipeline

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithPreWorkerRun(t *testing.T) {
	var h TaskHandler = func(task Task) {
		// do nothing
	}

	p := NewCompressingPipeline(
		context.TODO(),
		nil,
		nil,
		nil,
		nil,
		WithPreWorkerRun(h),
	)

	assert.True(t, reflect.ValueOf(h) == reflect.ValueOf(p.preWorkerRun))
	assert.True(t, reflect.TypeOf(h) == reflect.TypeOf(p.preWorkerRun))
}

package compress_test

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tarampampam/tinifier/v4/internal/cli/compress"
)

func TestErrorsWatcher(t *testing.T) {
	var (
		watcher                        = make(compress.ErrorsWatcher, 1)
		onErrorHandled, onLimitHandled atomic.Value
		ctx, cancel                    = context.WithCancel(context.Background())

		testErr = errors.New("test error")
	)

	defer cancel()

	onErrorHandled.Store(false)
	onLimitHandled.Store(false)

	go watcher.Watch(ctx, 2, compress.WithOnErrorHandler(func(err error) {
		require.EqualValues(t, testErr, err)

		onErrorHandled.Store(true)
	}), compress.WithLimitExceededHandler(func() {
		onLimitHandled.Store(true)
	}))

	require.False(t, onErrorHandled.Load().(bool))
	require.False(t, onLimitHandled.Load().(bool))

	watcher <- testErr
	runtime.Gosched()

	require.True(t, onErrorHandled.Load().(bool))
	require.False(t, onLimitHandled.Load().(bool))

	watcher <- testErr
	runtime.Gosched()

	require.True(t, onErrorHandled.Load().(bool))
	require.True(t, onLimitHandled.Load().(bool))

	close(watcher)
}

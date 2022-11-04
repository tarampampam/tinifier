package compress

import (
	"context"
)

type ErrorsWatcher chan error

type (
	errorsWatcherOptions struct {
		onError         func(error)
		onLimitExceeded func()
	}

	ErrorsWatcherOption func(*errorsWatcherOptions)
)

func WithOnErrorHandler(h func(error)) ErrorsWatcherOption {
	return func(o *errorsWatcherOptions) { o.onError = h }
}

func WithLimitExceededHandler(h func()) ErrorsWatcherOption {
	return func(o *errorsWatcherOptions) { o.onLimitExceeded = h }
}

func (w ErrorsWatcher) Watch(ctx context.Context, errorsLimit uint, options ...ErrorsWatcherOption) {
	var (
		opt     = &errorsWatcherOptions{}
		counter uint
	)

	for _, o := range options {
		o(opt)
	}

	for {
		select {
		case <-ctx.Done():
			return

		case err, isOpened := <-w:
			if !isOpened {
				return
			}

			if opt.onError != nil {
				opt.onError(err)
			}

			counter++

			if counter >= errorsLimit {
				if opt.onLimitExceeded != nil {
					opt.onLimitExceeded()
				}

				return
			}
		}
	}
}

func (w ErrorsWatcher) Push(ctx context.Context, err error) {
	select {
	case <-ctx.Done():
	case w <- err:
	}
}

package errgroup_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"gh.tarampamp.am/tinifier/v5/internal/errgroup"
)

func TestGroup_Go_NothingToDo(t *testing.T) {
	t.Parallel()

	var (
		ctx, cancel = context.WithCancel(context.Background())
		eg, egCtx   = errgroup.New(ctx)
		flag        bool
	)

	defer cancel()

	eg.Go(func(ctx context.Context) error {
		defer func() { flag = true }()

		if ctx != egCtx {
			t.Error("unexpected context")
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !flag {
		t.Error("expected the function to be called")
	}
}

func TestGroup_Go_CancelOnFirstError(t *testing.T) {
	t.Parallel()

	var (
		ctx, cancel = context.WithCancel(context.Background())
		eg, _       = errgroup.New(ctx)
		counter     atomic.Uint32
	)

	defer cancel()

	eg.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
			counter.Add(1) // should not be called due to the context cancellation
		}

		return errors.New("long") // should be ignored due to another goroutine error
	})

	eg.Go(func(ctx context.Context) error {
		<-time.After(time.Millisecond)

		counter.Add(1)

		return errors.New("short")
	})

	var err = eg.Wait()

	if err == nil {
		t.Error("expected an error")

		return
	}

	if err.Error() != "short" {
		t.Errorf("unexpected error: %v", err)
	}

	if counter.Load() != 1 {
		t.Error("expected the second function to be called")
	}
}

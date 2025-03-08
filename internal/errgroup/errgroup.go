package errgroup

import (
	"context"
	"sync"
)

// Group is a collection of goroutines working on subtasks that are part of the same overall task.
type Group interface {
	// Wait blocks until all function calls from the Go method have returned, then returns the first non-nil
	// error (if any) from them.
	Wait() error

	// Go calls the given function in a new goroutine.
	// It blocks until the new goroutine can be added without the number of active goroutines in the group.
	//
	// The first call to return a non-nil error cancels the group's context, if the group was created by calling
	// WithContext. The error will be returned by Wait.
	Go(f func(context.Context) error)
}

// group implementation was taken from the stdlib package golang.org/x/sync@v0.11.0, but with the following changes:
//   - SetLimit method was removed
//   - WithContext function was renamed to New
//   - New function returns Group (interface) instead of *group (struct)
//   - Go method accepts a function with a context.Context argument
type group struct {
	ctx     context.Context
	cancel  func(error)
	wg      sync.WaitGroup
	sem     chan struct{}
	errOnce sync.Once
	err     error
}

var _ Group = (*group)(nil) // ensure that group implements Group

// New returns a new group and an associated Context derived from ctx.
//
// The derived Context is canceled the first time a function passed to Go returns a non-nil error or the first
// time Wait returns, whichever occurs first.
func New(ctx context.Context) (Group, context.Context) {
	ctx, cancel := context.WithCancelCause(ctx)

	return &group{ctx: ctx, cancel: cancel}, ctx
}

func (g *group) done() {
	if g.sem != nil {
		<-g.sem
	}

	g.wg.Done()
}

func (g *group) Wait() error {
	g.wg.Wait()

	if g.cancel != nil {
		g.cancel(g.err)
	}

	return g.err
}

func (g *group) Go(f func(context.Context) error) {
	if g.sem != nil {
		g.sem <- struct{}{}
	}

	g.wg.Add(1)

	go func() {
		defer g.done()

		if err := f(g.ctx); err != nil {
			g.errOnce.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel(g.err)
				}
			})
		}
	}()
}

package threadsafe

import (
	"sync"

	"github.com/pkg/errors"
)

type ErrorBag struct {
	mu  sync.RWMutex
	err error
}

func (b *ErrorBag) Set(err error) {
	b.mu.Lock()
	b.err = err
	b.mu.Unlock()
}

func (b *ErrorBag) Unset() {
	b.mu.Lock()
	b.err = nil
	b.mu.Unlock()
}

func (b *ErrorBag) Wrap(err error) {
	b.mu.Lock()
	if b.err == nil {
		b.err = err
	} else {
		b.err = errors.Wrap(b.err, err.Error())
	}
	b.mu.Unlock()
}

func (b *ErrorBag) Get() error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.err
}

package threadsafe

import (
	"sync"

	"github.com/pkg/errors"
)

// ErrorBag allows to get/set/wrap error concurrently.
type ErrorBag struct { // TODO(jetexe) rename to "Error"? bag for one?
	mu  sync.RWMutex
	err error
}

// Set sets the error.
func (b *ErrorBag) Set(err error) {
	b.mu.Lock()
	b.err = err
	b.mu.Unlock()
}

// Unset unsets the error.
func (b *ErrorBag) Unset() {
	b.mu.Lock()
	b.err = nil
	b.mu.Unlock()
}

// Wrap set error in a bag if it does not set, or wrap the error instead.
func (b *ErrorBag) Wrap(err error) {
	if err == nil {
		return
	}

	b.mu.Lock()
	if b.err == nil {
		b.err = err
	} else {
		b.err = errors.Wrap(b.err, err.Error())
	}
	b.mu.Unlock()
}

// Get returns an error from the bag.
func (b *ErrorBag) Get() error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.err
}

package threadsafe

import (
	"sync"

	"github.com/pkg/errors"
)

// Error allows to get/set/wrap error concurrently.
type Error struct {
	mu  sync.RWMutex
	err error
}

// Set sets the error.
func (b *Error) Set(err error) {
	b.mu.Lock()
	b.err = err
	b.mu.Unlock()
}

// Unset unsets the error.
func (b *Error) Unset() {
	b.mu.Lock()
	b.err = nil
	b.mu.Unlock()
}

// Wrap set error in a bag if it does not set, or wrap the error instead.
func (b *Error) Wrap(err error) {
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
func (b *Error) Get() error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.err
}

// Package keys provides Keeper struct for concurrent API keys management.
package keys

import (
	"errors"
	"sync"
)

// Keeper allows you to store some API keys in a one place.
type Keeper struct {
	mu    sync.RWMutex
	state map[string]struct{}
}

// ErrKeyNotExists occurred when keeper does not contains keys.
var ErrKeyNotExists = errors.New("key not exists")

// NewKeeper creates new keeper instance.
func NewKeeper() Keeper {
	return Keeper{
		state: make(map[string]struct{}),
	}
}

// Add new key in a keeper. Empty or/and key duplicates are not allowed.
func (k *Keeper) Add(keys ...string) error {
	if len(keys) > 0 {
		k.mu.Lock()
		defer k.mu.Unlock()

		for i := 0; i < len(keys); i++ {
			if len(keys[i]) == 0 {
				return errors.New("empty keys are not allowed")
			}

			if _, ok := k.state[keys[i]]; ok {
				return errors.New("key \"" + keys[i] + "\" already exists")
			}

			k.state[keys[i]] = struct{}{}
		}
	}

	return nil
}

// Remove rey from storage.
func (k *Keeper) Remove(keys ...string) {
	if len(keys) == 0 {
		return
	}

	k.mu.Lock()
	k.remove(keys...)
	k.mu.Unlock()
}

func (k *Keeper) remove(keys ...string) {
	for i := 0; i < len(keys); i++ {
		delete(k.state, keys[i])
	}
}

// Get the key which does not exceed the maximum count of errors. If none exists, ErrKeyNotExists will be returned.
func (k *Keeper) Get() (string, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	for key := range k.state {
		return key, nil
	}

	return "", ErrKeyNotExists
}

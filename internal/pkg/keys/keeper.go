package keys

import (
	"errors"
	"sync"
)

// Keeper allows you to store some API keys in a one place with very useful feature - "bad" keys reporting and only
// "good" keys usage. Maximal key errors means maximal key reports before key removal.
type Keeper struct {
	mu           sync.RWMutex
	state        map[string]int
	maxKeyErrors int
}

var (
	ErrKeyNotExists = errors.New("key not exists")
	ErrNoUsableKey  = errors.New("all kept keys has too many errors")
)

// NewKeeper creates new keeper instance.
func NewKeeper(maxKeyErrors int) Keeper {
	return Keeper{
		state:        make(map[string]int),
		maxKeyErrors: maxKeyErrors,
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

			k.state[keys[i]] = 0
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

// ReportKey allows to change key errors count (positive delta will increase key errors, and negative delta vice
// versa - reduces). If key errors count exceeds the maximum allowable value - key will be removed.
func (k *Keeper) ReportKey(key string, delta int) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if _, ok := k.state[key]; ok {
		k.state[key] += delta

		if k.state[key] >= k.maxKeyErrors {
			k.remove(key) // remove invalid key right now
		}

		return nil
	}

	return ErrKeyNotExists
}

// Get the key which does not exceed the maximum count of errors. If none exists, ErrNoUsableKey will be returned.
func (k *Keeper) Get() (string, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	for key := range k.state {
		return key, nil
	}

	return "", ErrNoUsableKey
}

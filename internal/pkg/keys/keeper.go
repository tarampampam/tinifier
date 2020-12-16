package keys

import (
	"errors"
	"sync"
)

type Keeper struct {
	mu           sync.RWMutex
	storage      map[string]*keyInfo
	maxKeyErrors int
}

type keyInfo struct {
	errorsCount int
}

var (
	ErrKeyNotExists     = errors.New("key not exists")
	ErrEmptyKeysStorage = errors.New("empty keys storage")
	ErrNoUsableKey      = errors.New("all kept keys has too many errors")
)

func NewKeeper(maxKeyErrors int) Keeper {
	return Keeper{
		storage:      make(map[string]*keyInfo),
		maxKeyErrors: maxKeyErrors,
	}
}

func (k *Keeper) Add(keys ...string) error {
	if len(keys) > 0 {
		k.mu.Lock()
		defer k.mu.Unlock()

		for i := 0; i < len(keys); i++ {
			if len(keys[i]) == 0 {
				return errors.New("empty keys are not allowed")
			}

			if _, ok := k.storage[keys[i]]; ok {
				return errors.New("key \"" + keys[i] + "\" already exists")
			}

			k.storage[keys[i]] = &keyInfo{}
		}
	}

	return nil
}

func (k *Keeper) Remove(keys ...string) {
	if len(keys) > 0 {
		k.mu.Lock()
		for i := 0; i < len(keys); i++ {
			delete(k.storage, keys[i])
		}
		k.mu.Unlock()
	}
}

func (k *Keeper) ReportKeyError(key string, delta int) error { // TODO delete invalid key right here
	k.mu.Lock()
	defer k.mu.Unlock()

	if v, ok := k.storage[key]; ok {
		v.errorsCount += delta

		return nil
	}

	return ErrKeyNotExists
}

func (k *Keeper) Get() (string, error) { // TODO randomize returned key
	k.mu.RLock()
	defer k.mu.RUnlock()

	if len(k.storage) == 0 {
		return "", ErrEmptyKeysStorage
	}

	for key, value := range k.storage {
		if value.errorsCount < k.maxKeyErrors {
			return key, nil
		}
	}

	return "", ErrNoUsableKey
}

package tinypng

import "sync"

type ClientsPool struct {
	opts []ClientOption

	mu   sync.Mutex
	keys map[string]*Client
}

// NewClientsPool initializes a new pool of clients using the given API keys.
// Additional options can be provided to customize the clients.
func NewClientsPool(apiKeys []string, opts ...ClientOption) *ClientsPool {
	var pool = ClientsPool{
		opts: opts,
		keys: make(map[string]*Client, len(apiKeys)), // initialize the map to avoid nil map assignment
	}

	for _, key := range apiKeys {
		pool.keys[key] = nil // prepopulate the map with keys and nil clients
	}

	return &pool
}

// Get retrieves a random client from the pool.
//
// If the pool is empty, it returns nil and false as the last return value. If the client for a key is
// uninitialized, it creates a new one and returns it.
// The returned cleanup function should be called when the client is no longer needed, allowing the key
// to be removed from the pool.
func (p *ClientsPool) Get() (*Client, func(), bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// iterate over the map to get a random key (order of map iteration is not guaranteed)
	for key, c := range p.keys {
		// if the client is not initialized, create a new one
		if c == nil {
			p.keys[key] = NewClient(key, p.opts...) // store in the pool
			c = p.keys[key]
		}

		// return the client along with a cleanup function that removes the key from the pool
		return c, func() {
			p.mu.Lock()
			defer p.mu.Unlock()

			// remove key from the pool when the client is no longer needed so any next Get() call will
			// create a new client but using another key
			delete(p.keys, key)
		}, true
	}

	return nil, func() {}, false
}

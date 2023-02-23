package compress

import (
	"errors"
	"sync"

	"gh.tarampamp.am/tinifier/v4/pkg/tinypng"
)

var errNoClients = errors.New("no clients")

type clientsPool struct {
	mu   sync.Mutex
	pool map[string]*tinypng.Client
}

func newClientsPool(apiKeys []string, options ...tinypng.ClientOption) *clientsPool {
	p := clientsPool{
		pool: make(map[string]*tinypng.Client),
	}

	for _, key := range apiKeys {
		p.pool[key] = tinypng.NewClient(key, options...)
	}

	return &p
}

// Remove removes client from pool.
func (p *clientsPool) Remove(apiKey string) {
	p.mu.Lock()
	delete(p.pool, apiKey)
	p.mu.Unlock()
}

// Get returns an arbitrary client.
func (p *clientsPool) Get() (string, *tinypng.Client) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for key, c := range p.pool {
		return key, c
	}

	return "", nil
}

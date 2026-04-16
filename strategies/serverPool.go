package strategies

import (
	"errors"
	"sync"
	"sync/atomic"
)

type Pool struct {
	mu      sync.RWMutex
	servers []*Server
	counter atomic.Uint64
	ring    *HashRing
}

var DefaultPool = &Pool{ring: newHashRing()}

// Register adds a server to the pool, marking it healthy.
// If already present, it is re-marked as healthy (useful after recovery).
func (p *Pool) Register(url string) {
	p.mu.Lock()
	found := false
	for _, s := range p.servers {
		if s.URL == url {
			s.Healthy = true
			found = true
			break
		}
	}
	if !found {
		p.servers = append(p.servers, &Server{URL: url, Healthy: true})
	}
	p.mu.Unlock()

	// update ring after releasing pool lock to avoid nested locking
	if !found {
		p.ring.Add(url)
	}
}

// Deregister removes a server from the pool entirely.
func (p *Pool) Deregister(url string) {
	p.mu.Lock()
	for i, s := range p.servers {
		if s.URL == url {
			p.servers = append(p.servers[:i], p.servers[i+1:]...)
			break
		}
	}
	p.mu.Unlock()

	p.ring.Remove(url)
}

// SetHealth updates the health status of a server.
func (p *Pool) SetHealth(url string, healthy bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, s := range p.servers {
		if s.URL == url {
			s.Healthy = healthy
			return
		}
	}
}

// Next returns the next healthy server URL using round-robin.
func (p *Pool) Next() (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var healthy []string
	for _, s := range p.servers {
		if s.Healthy {
			healthy = append(healthy, s.URL)
		}
	}

	if len(healthy) == 0 {
		return "", errors.New("no healthy servers available")
	}

	idx := p.counter.Add(1) - 1
	return healthy[idx%uint64(len(healthy))], nil
}

// ConsistentHashNext routes key to a healthy server via the hash ring.
// Builds the healthy set under the pool lock, then queries the ring separately
// to avoid nested locking between pool and ring mutexes.
func (p *Pool) ConsistentHashNext(key string) (string, error) {
	p.mu.RLock()
	healthy := make(map[string]bool, len(p.servers))
	for _, s := range p.servers {
		if s.Healthy {
			healthy[s.URL] = true
		}
	}
	p.mu.RUnlock()

	return p.ring.Get(key, healthy)
}

// List returns a snapshot of all registered servers and their health status.
func (p *Pool) List() []*Server {
	p.mu.RLock()
	defer p.mu.RUnlock()
	snapshot := make([]*Server, len(p.servers))
	copy(snapshot, p.servers)
	return snapshot
}

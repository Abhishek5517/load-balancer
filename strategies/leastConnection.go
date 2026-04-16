package strategies

import "errors"

// LeastConnectionServer routes to the healthy server with the fewest active connections.
func LeastConnectionServer() (string, error) {
	return DefaultPool.LeastConnNext()
}

func (p *Pool) LeastConnNext() (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var best *Server
	for _, s := range p.servers {
		if !s.Healthy {
			continue
		}
		if best == nil || s.ActiveConnections.Load() < best.ActiveConnections.Load() {
			best = s
		}
	}

	if best == nil {
		return "", errors.New("no healthy servers available")
	}
	return best.URL, nil
}

func (p *Pool) IncrementConn(url string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, s := range p.servers {
		if s.URL == url {
			s.ActiveConnections.Add(1)
			return
		}
	}
}

func (p *Pool) DecrementConn(url string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, s := range p.servers {
		if s.URL == url {
			s.ActiveConnections.Add(-1)
			return
		}
	}
}

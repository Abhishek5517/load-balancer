package strategies

import (
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"sync"
)

const replicas = 150

type HashRing struct {
	mu     sync.RWMutex
	points []uint32          // sorted ring positions
	ring   map[uint32]string // position → server URL
}

func newHashRing() *HashRing {
	return &HashRing{ring: make(map[uint32]string)}
}

func (h *HashRing) hash(key string) uint32 {
	digest := md5.Sum([]byte(key))
	return binary.BigEndian.Uint32(digest[:4])
}

// Add places a server's virtual nodes onto the ring.
func (h *HashRing) Add(url string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := 0; i < replicas; i++ {
		point := h.hash(fmt.Sprintf("%s#%d", url, i))
		h.ring[point] = url
		h.points = append(h.points, point)
	}
	sort.Slice(h.points, func(i, j int) bool { return h.points[i] < h.points[j] })
}

// Remove clears all virtual nodes for a server from the ring.
func (h *HashRing) Remove(url string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := 0; i < replicas; i++ {
		point := h.hash(fmt.Sprintf("%s#%d", url, i))
		delete(h.ring, point)
	}
	// rebuild points from remaining keys
	h.points = h.points[:0]
	for point := range h.ring {
		h.points = append(h.points, point)
	}
	sort.Slice(h.points, func(i, j int) bool { return h.points[i] < h.points[j] })
}

// Get returns the first healthy server clockwise from hash(key).
// healthy is a pre-built set passed in by the caller to avoid lock inversion.
func (h *HashRing) Get(key string, healthy map[string]bool) (string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.points) == 0 {
		return "", errors.New("no servers in ring")
	}

	hash := h.hash(key)

	// find first point >= hash (clockwise), wrap around if past the end
	idx := sort.Search(len(h.points), func(i int) bool { return h.points[i] >= hash })
	if idx == len(h.points) {
		idx = 0
	}

	// walk clockwise until a healthy server is found
	for i := 0; i < len(h.points); i++ {
		point := h.points[(idx+i)%len(h.points)]
		url := h.ring[point]
		if healthy[url] {
			return url, nil
		}
	}

	return "", errors.New("no healthy servers available")
}

// ConsistentHashServer routes based on the provided key (typically client IP).
func ConsistentHashServer(key string) (string, error) {
	return DefaultPool.ConsistentHashNext(key)
}

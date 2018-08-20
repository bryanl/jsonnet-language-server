package locate

import (
	"sync"
)

// LocatableCache stores locatables.
type LocatableCache struct {
	store map[string][]Locatable

	mu sync.Mutex
}

// NewLocatableCache creates an instance of LocatableCache.
func NewLocatableCache() *LocatableCache {
	return &LocatableCache{
		store: make(map[string][]Locatable),
	}
}

// Store stores a locatable in the cache.
func (lc *LocatableCache) Store(filename string, l []Locatable) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.store[filename] = l
	return nil
}

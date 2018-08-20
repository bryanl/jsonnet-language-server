package lexical

import (
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/pkg/errors"
)

// LocatableCache stores locatables.
type LocatableCache struct {
	store map[string][]locate.Locatable
}

// NewLocatableCache creates an instance of LocatableCache.
func NewLocatableCache() *LocatableCache {
	return &LocatableCache{
		store: make(map[string][]locate.Locatable),
	}
}

// Store stores a locatable in the cache.
func (lc *LocatableCache) Store(filename string, l *locate.Locatable) error {
	return errors.New("LocatableCache.Store not implemented")
}

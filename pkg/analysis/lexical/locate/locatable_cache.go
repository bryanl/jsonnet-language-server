package locate

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

// GetAtPosition gets a locatable from the cache by position. It selects the
// position with the smallest range. If it can't find a locatable, it will
// return an error.
func (lc *LocatableCache) GetAtPosition(filename string, pos ast.Location) (*Locatable, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if strings.HasPrefix(filename, "file://") {
		var err error
		filename, err = uri.ToPath(filename)
		if err != nil {
			return nil, err
		}
	}

	list, ok := lc.store[filename]
	if !ok {
		var files []string
		for k := range lc.store {
			files = append(files, k)

		}
		logrus.WithField("entries", strings.Join(files, ", ")).Info("existing entries")
		return nil, errors.Errorf("filename %q is unknown to the locatable cache", filename)
	}

	logrus.Infof("finding token at position %s", pos.String())

	var selected *Locatable
	for i := range list {
		l := list[i]
		if selected == nil && inRange(pos, l.Loc) {
			logrus.Debugf("setting %T as selected token because there was none (%s)",
				l.Token, l.Loc.String())
			selected = &l
		} else if selected != nil && inRange(pos, l.Loc) && isRangeSmaller(selected.Loc, l.Loc) {
			logrus.Debugf("setting %T as selected token because its range %s is smaller than %s from %T",
				l.Token, l.Loc.String(), selected.Loc.String(), selected.Token)
			selected = &l
		}
	}

	return selected, nil
}

// Store stores a locatable in the cache.
func (lc *LocatableCache) Store(filename string, l []Locatable) error {
	logrus.Infof("storing %s", filename)
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.store[filename] = l
	tokenField := fmt.Sprintf("cache.locatable.file[%s].tokens", filename)
	logrus.WithFields(logrus.Fields{
		"cache.locatable.files.count": len(lc.store),
		tokenField:                    len(lc.store[filename]),
	}).Info("locatable cache statistics")

	return nil
}

package lexical

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextDocumentWatcher_watch(t *testing.T) {
	lc := locate.NewLocatableCache()
	c := &fakeTextDocumentWatcherConfig{
		lc: lc,
	}
	_ = NewTextDocumentWatcher(c)

	td := config.NewTextDocumentFromItem(lsp.TextDocumentItem{
		Text: "{}",
		URI:  "file:///file.jsonnet",
	})

	c.watchFn(td)

	l, err := lc.GetAtPosition("/file.jsonnet", createLoc(1, 1))
	require.NoError(t, err)

	assert.NotNil(t, l)
}

type fakeTextDocumentWatcherConfig struct {
	lc *locate.LocatableCache

	watchFn config.DispatchFn
}

var _ TextDocumentWatcherConfig = (*fakeTextDocumentWatcherConfig)(nil)

func (c *fakeTextDocumentWatcherConfig) LocatableCache() *locate.LocatableCache {
	return c.lc
}

func (c *fakeTextDocumentWatcherConfig) Watch(k string, fn config.DispatchFn) config.DispatchCancelFn {
	c.watchFn = fn

	return func() {}
}

package lexical

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
)

func TestTextDocumentWatcher_watch(t *testing.T) {
	lc := locate.NewLocatableCache()
	c := &fakeTextDocumentWatcherConfig{
		lc: lc,
	}

	dp := &fakeDocumentProcessor{}

	tdw := NewTextDocumentWatcher(c, dp)
	conn := &fakeRPCConn{}
	tdw.SetConn(conn)

	td := config.NewTextDocumentFromItem(lsp.TextDocumentItem{
		Text: "{}",
		URI:  "file:///file.jsonnet",
	})

	c.watchFn(td)
}

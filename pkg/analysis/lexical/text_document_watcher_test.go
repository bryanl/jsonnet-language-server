package lexical

import (
	"testing"

	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
)

func TestTextDocumentWatcher_watch(t *testing.T) {
	c := &fakeTextDocumentWatcherConfig{}

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

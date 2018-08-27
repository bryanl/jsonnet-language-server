package lexical

import (
	"context"

	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/sourcegraph/jsonrpc2"
)

type fakeDocumentProcessor struct {
	processErr error
}

var _ DocumentProcessor = (*fakeDocumentProcessor)(nil)

func (dp *fakeDocumentProcessor) Process(td config.TextDocument, conn RPCConn) error {
	return dp.processErr
}

type fakeTextDocumentWatcherConfig struct {
	watchFn config.DispatchFn
}

var _ TextDocumentWatcherConfig = (*fakeTextDocumentWatcherConfig)(nil)

func (c *fakeTextDocumentWatcherConfig) Watch(k string, fn config.DispatchFn) config.DispatchCancelFn {
	c.watchFn = fn

	return func() {}
}

type fakeRPCConn struct {
	notifyErr error
}

func (c *fakeRPCConn) Notify(ctx context.Context, method string, params interface{}, opts ...jsonrpc2.CallOption) error {
	return c.notifyErr
}

package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/tracing"
	"github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/opentracing/opentracing-go/log"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/bryanl/jsonnet-language-server/pkg/langserver"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/uri"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
)

func textDocumentCompletion(ctx context.Context, r *request, c *config.Config) (interface{}, error) {
	var rp lsp.ReferenceParams
	if err := r.Decode(&rp); err != nil {
		return nil, err
	}

	cmpl, err := newComplete(rp, c)
	if err != nil {
		return nil, err
	}

	response, err := cmpl.handle(ctx)
	if err != nil {
		return nil, err
	}

	return response, nil
}

type complete struct {
	referenceParams   lsp.ReferenceParams
	config            *config.Config
	completionMatcher *langserver.CompletionMatcher
}

func newComplete(rp lsp.ReferenceParams, cfg *config.Config) (*complete, error) {
	c := &complete{
		referenceParams:   rp,
		config:            cfg,
		completionMatcher: langserver.NewCompletionMatcher(),
	}

	jpm := newJsonnetPathManager(cfg)
	mh := newMatchHandler(jpm, cfg.NodeCache())
	if err := mh.register(c.completionMatcher); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *complete) handle(ctx context.Context) (interface{}, error) {
	span, ctx := tracing.ChildSpan(ctx, "complete")
	defer span.Finish()

	uriStr := c.referenceParams.TextDocument.URI
	text, err := c.config.Text(ctx, uriStr)
	if err != nil {
		return nil, errors.Wrap(err, "loading current text")
	}

	span.LogFields(
		log.String("reference-params", spew.Sdump(c.referenceParams)),
	)

	path, err := uri.ToPath(uriStr)
	if err != nil {
		return nil, err
	}

	list := &lsp.CompletionList{
		Items: []lsp.CompletionItem{},
	}

	pos := position.FromLSPPosition(c.referenceParams.Position)
	editRange := position.NewRange(pos, pos)

	span.LogFields(
		log.String("truncate.to", pos.String()),
	)

	matchText, err := text.Truncate(pos)
	if err != nil {
		return nil, err
	}
	matchText = strings.TrimSpace(matchText)
	span.LogFields(
		log.String("truncate.text", matchText),
	)

	matchItems, err := c.completionMatcher.Match(ctx, pos, path, text.String())
	if err != nil {
		return nil, err
	}

	if len(matchItems) > 0 {
		return matchItems, nil
	}

	m, err := token.LocationScope(path, text.String(), pos, c.config.NodeCache())
	if err != nil {
		span.LogFields(
			log.Error(err),
		)
	} else {
		for _, k := range m.Keys() {
			e, err := m.Get(k)
			if err != nil {
				span.LogFields(
					log.Error(err),
				)
				break
			}

			ci := lsp.CompletionItem{
				Label:         k,
				Kind:          lsp.CIKVariable,
				Detail:        e.Detail,
				Documentation: e.Documentation,
				SortText:      fmt.Sprintf("0_%s", k),
				TextEdit: lsp.TextEdit{
					Range:   editRange.ToLSP(),
					NewText: k,
				},
			}

			list.Items = append(list.Items, ci)
		}
	}

	for _, k := range m.Keywords() {
		ci := lsp.CompletionItem{
			Label:    k,
			Kind:     lsp.CIKKeyword,
			SortText: fmt.Sprintf("1_%s", k),
			TextEdit: lsp.TextEdit{
				Range:   editRange.ToLSP(),
				NewText: k,
			},
		}

		list.Items = append(list.Items, ci)
	}

	return list, nil

}

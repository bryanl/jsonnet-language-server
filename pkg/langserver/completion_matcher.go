package langserver

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/tracing"
	"github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/bryanl/jsonnet-language-server/pkg/util/text"
	"github.com/opentracing/opentracing-go/log"
)

// CompletionAction is an action performed on a completion match.
type CompletionAction func(ctx context.Context, pos position.Position, path, source string) ([]lsp.CompletionItem, error)

// CompletionMatcher can register multiple terms to complete against.
type CompletionMatcher struct {
	store     map[*regexp.Regexp]CompletionAction
	mu        sync.Mutex
	nodeCache *token.NodeCache
}

// NewCompletionMatcher creates an instance of CompletionMatchers.
func NewCompletionMatcher() *CompletionMatcher {
	return &CompletionMatcher{
		store: make(map[*regexp.Regexp]CompletionAction),
	}
}

// Register registers terms for the matcher
func (cm *CompletionMatcher) Register(term string, fn CompletionAction) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	expr := fmt.Sprintf(`(%s)[;,\]\)\}]*$`, term)
	re, err := regexp.Compile(expr)
	if err != nil {
		return err
	}

	cm.store[re] = fn

	return nil
}

// Match matches at a point defined in the edit range.
func (cm *CompletionMatcher) Match(ctx context.Context, pos position.Position, path, source string) ([]lsp.CompletionItem, error) {
	span, ctx := tracing.ChildSpan(ctx, "completionMatcher")

	defer span.Finish()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	matched, err := text.Truncate(source, pos)
	if err != nil {
		return nil, err
	}

	for re, m := range cm.store {
		span.LogFields(
			log.String("match.text", matched),
			log.String("match.regex", re.String()),
		)
		match := re.FindStringSubmatch(matched)
		if match != nil {
			return m(ctx, pos, path, matched)
		}
	}

	return []lsp.CompletionItem{}, nil
}

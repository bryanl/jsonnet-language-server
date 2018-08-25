package langserver

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/sirupsen/logrus"
)

// CompletionAction is an action performed on a completion match.
type CompletionAction func(editRange lsp.Range, matched string) ([]lsp.CompletionItem, error)

// CompletionMatcher can register multiple terms to complete against.
type CompletionMatcher struct {
	store map[*regexp.Regexp]CompletionAction
	mu    sync.Mutex
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

	expr := fmt.Sprintf(`(%s)$`, term)
	re, err := regexp.Compile(expr)
	if err != nil {
		return err
	}

	cm.store[re] = fn

	return nil
}

// Match matches at a point definedi in the edit range.
func (cm *CompletionMatcher) Match(editRange lsp.Range, source string) ([]lsp.CompletionItem, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for re, m := range cm.store {
		logrus.Infof("trying to match %q to %s", source, re.String())
		match := re.FindStringSubmatch(source)
		if match != nil {
			return m(editRange, match[0])
		}
	}

	return []lsp.CompletionItem{}, nil
}

package langserver

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/bryanl/jsonnet-language-server/pkg/util/position"
	"github.com/sirupsen/logrus"
)

// CompletionAction is an action performed on a completion match.
type CompletionAction func(pos position.Position, path, source string) ([]lsp.CompletionItem, error)

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
func (cm *CompletionMatcher) Match(pos position.Position, path, source string) ([]lsp.CompletionItem, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	matched, err := truncateText(source, pos)
	if err != nil {
		return nil, err
	}

	for re, m := range cm.store {
		logrus.Debugf("trying to match %q to %s", matched, re.String())
		match := re.FindStringSubmatch(matched)
		if match != nil {
			return m(pos, path, matched)
		}
	}

	return []lsp.CompletionItem{}, nil
}

func (cm *CompletionMatcher) defaultMatcher(pos position.Position, path, source string) ([]lsp.CompletionItem, error) {
	var items []lsp.CompletionItem

	editRange := position.NewRange(pos, pos)

	m, err := token.LocationScope(path, source, pos, cm.nodeCache)
	if err != nil {
		logrus.WithError(err).WithField("loc", pos.String()).Debug("load scope")
	} else {
		for _, k := range m.Keys() {
			e, err := m.Get(k)
			if err != nil {
				logrus.WithError(err).Debugf("fetching %q from scope", k)
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

			items = append(items, ci)
		}
	}

	// for _, k := range m.Keywords() {
	// 	ci := lsp.CompletionItem{
	// 		Label:    k,
	// 		Kind:     lsp.CIKKeyword,
	// 		SortText: fmt.Sprintf("1_%s", k),
	// 		TextEdit: lsp.TextEdit{
	// 			Range:   editRange.ToLSP(),
	// 			NewText: k,
	// 		},
	// 	}

	// 	list.Items = append(list.Items, ci)
	// }

	return items, nil
}

// Truncate returns text truncated at a position.
func truncateText(source string, p position.Position) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(source))
	scanner.Split(bufio.ScanBytes)

	var buf bytes.Buffer

	c := 0
	l := 1

	for scanner.Scan() {
		c++

		t := scanner.Text()

		_, err := buf.WriteString(t)
		if err != nil {
			return "", err
		}

		if l == p.Line() && c == p.Column() {
			break
		}

		if t == "\n" {
			l++
			c = 0
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.TrimRight(buf.String(), "\n"), nil
}

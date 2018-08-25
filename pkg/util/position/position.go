package position

import (
	"fmt"

	"github.com/bryanl/jsonnet-language-server/pkg/lsp"
	"github.com/google/go-jsonnet/ast"
)

// Position in a position.
type Position struct {
	line   int
	column int
}

// New creates a Position.
func New(line, column int) Position {
	return Position{
		line:   line,
		column: column,
	}
}

// FromLSPPosition converts a LSP Position to a Position.
func FromLSPPosition(lspp lsp.Position) Position {
	return New(lspp.Line+1, lspp.Character+1)
}

// Line is the position line.
func (p *Position) Line() int {
	return p.line
}

// Column is the position column.
func (p *Position) Column() int {
	return p.column
}

// ToLSP converts to a LSP Position.
func (p *Position) ToLSP() lsp.Position {
	lspp := lsp.Position{
		Line:      p.Line() - 1,
		Character: p.Column() - 1,
	}

	return lspp
}

// ToJsonnet converts to a Jsonnet Location.
func (p *Position) ToJsonnet() ast.Location {
	l := ast.Location{
		Line:   p.Line(),
		Column: p.Column(),
	}

	return l
}

func (p *Position) String() string {
	return fmt.Sprintf("%v:%v", p.Line(), p.Column())
}

// Range is a range between two Positions.
type Range struct {
	Start Position
	End   Position
}

// NewRange creates a Range.
func NewRange(start, end Position) Range {
	return Range{
		Start: start,
		End:   end,
	}
}

// ToLSP converts a range to a LSP Range.
func (r *Range) ToLSP() lsp.Range {
	return lsp.Range{
		Start: r.Start.ToLSP(),
		End:   r.End.ToLSP(),
	}
}

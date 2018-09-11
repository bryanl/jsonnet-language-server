package position

import (
	"bytes"
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

// FromJsonnetLocation converts a Jsonnet location to a Postion.
func FromJsonnetLocation(loc ast.Location) Position {
	return New(loc.Line, loc.Column)
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

// IsInJsonnetRange returns true if the position is in a Jsonnet
// range.
// nolint: gocyclo
func (p *Position) IsInJsonnetRange(r ast.LocationRange) bool {
	if r.Begin.Line == p.line && p.line == r.End.Line &&
		r.Begin.Column <= p.column && p.column <= r.End.Column {
		return true
	} else if r.Begin.Line < p.line && p.line == r.End.Line &&
		p.column <= r.End.Column {
		return true
	} else if r.Begin.Line == p.line && p.line < r.End.Line &&
		p.column >= r.Begin.Column {
		return true
	} else if r.Begin.Line < p.line && p.line < r.End.Line {
		return true
	}

	return false
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

// NewRangeFromCoords creates a range from coordinates.
func NewRangeFromCoords(sl, sc, el, ec int) Range {
	return NewRange(
		New(sl, sc),
		New(el, ec))
}

// CombinedRange combines two ranges.
func CombinedRange(start, end Range) Range {
	return Range{
		Start: start.Start,
		End:   end.End,
	}
}

// ToLSP converts a range to a LSP Range.
func (r *Range) ToLSP() lsp.Range {
	return lsp.Range{
		Start: r.Start.ToLSP(),
		End:   r.End.ToLSP(),
	}
}

func (r *Range) String() string {
	return fmt.Sprintf("%s-%s", r.Start.String(), r.End.String())
}

// FromJsonnetRange converts a Jsonnet LocationRange to
// Range.
func FromJsonnetRange(r ast.LocationRange) Range {
	start := FromJsonnetLocation(r.Begin)
	end := FromJsonnetLocation(r.End)

	return NewRange(start, end)
}

// Location is a range within a URI.
type Location struct {
	uri string
	r   Range
}

// NewLocation creates a Location.
func NewLocation(uri string, r Range) Location {
	return Location{
		uri: uri,
		r:   r,
	}
}

// LocationFromJsonnet converts a Jsonnet LocationRange to
// Location.
func LocationFromJsonnet(r ast.LocationRange) Location {
	fileName := r.FileName
	return NewLocation(fileName, FromJsonnetRange(r))
}

// URI is the URI for the location.
func (l *Location) URI() string {
	return l.uri
}

// Range is range of the location.
func (l *Location) Range() Range {
	return l.r
}

// ToLSP converts the Location to a LSP Location.
func (l *Location) ToLSP() lsp.Location {
	return lsp.Location{
		URI:   fmt.Sprintf("file://%s", l.uri),
		Range: l.r.ToLSP(),
	}
}

// ToJsonnet converts the Location to a Jsonnet LocationRange.
func (l *Location) ToJsonnet() ast.LocationRange {
	return ast.LocationRange{
		FileName: l.uri,
		Begin:    l.r.Start.ToJsonnet(),
		End:      l.r.End.ToJsonnet(),
	}
}

func (l *Location) String() string {
	start := l.Range().Start
	end := l.Range().End
	return fmt.Sprintf("%s(%v)-(%v)", l.uri, start.String(), end.String())
}

// Locations is a set of locations.
type Locations struct {
	store map[Location]bool
}

// Add adds a location to the set.
func (ls *Locations) Add(l Location) {
	if ls.store == nil {
		ls.store = make(map[Location]bool)
	}

	ls.store[l] = true
}

func (l *Locations) String() string {
	var buf bytes.Buffer
	buf.WriteString("[")
	sl := l.Slice()
	for i := 0; i < len(sl); i++ {
		buf.WriteString(sl[i].String())
		if i < len(sl)-1 {
			buf.WriteString(", ")
		}
	}
	buf.WriteString("]")
	return buf.String()
}

// Equal returns true if this set of locations equals
// another set of locations.
func (ls *Locations) Equal(other *Locations) bool {
	if other == nil {
		return false
	}

	return locationsEqual(ls.Slice(), other.Slice())
}

// Slice converts the locations to a slice.
func (ls *Locations) Slice() []Location {
	var out []Location
	for k := range ls.store {
		out = append(out, k)
	}

	return out
}

func locationsEqual(a, b []Location) bool {
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].String() != b[i].String() {
			return false
		}
	}

	return true
}

package lexical

import (
	"github.com/google/go-jsonnet/ast"
)

type Locatable struct {
	Token interface{}
	Loc   ast.LocationRange
}

func inRange(l ast.Location, lr ast.LocationRange) bool {
	if lr.Begin.Line == l.Line {
		return lr.Begin.Column <= l.Column
	} else if lr.Begin.Line < l.Line && lr.End.Line >= l.Line {
		return true
	}

	return false
}

func isRangeSmaller(r1, r2 ast.LocationRange) bool {
	b := inRange(r2.Begin, r1) && inRange(r2.End, r1)
	return b
}

func afterRangeOrEqual(l ast.Location, lr ast.LocationRange) bool {
	end := lr.End
	if l.Line > end.Line {
		return true
	} else if l.Line == end.Line && l.Column >= end.Column {
		return true
	}

	return false
}

func beforeRange(l ast.Location, r ast.LocationRange) bool {
	begin := r.Begin
	if l.Line < begin.Line {
		return true
	} else if l.Line == begin.Line && l.Column < begin.Column {
		return true
	}

	return false
}

func afterRange(l ast.Location, lr ast.LocationRange) bool {
	end := lr.End
	if l.Line > end.Line {
		return true
	} else if l.Line == end.Line && l.Column > end.Column {
		return true
	}

	return false
}

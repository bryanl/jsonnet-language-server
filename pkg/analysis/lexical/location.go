package lexical

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

type Locatable struct {
	Token  interface{}
	Loc    ast.LocationRange
	Parent *Locatable
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

func localBindRange(source []byte, lb ast.LocalBind, parent interface{}) (ast.LocationRange, error) {
	data, err := ExtractUntil(source, lb.Body.Loc().Begin)
	if err != nil {
		return ast.LocationRange{}, err
	}

	re, err := regexp.Compile(fmt.Sprintf(`(?m)\s+%s\s*=\s*\z`, string(lb.Variable)))
	if err != nil {
		return ast.LocationRange{}, err
	}

	if string(lb.Variable) == "$" {
		return *lb.Body.Loc(), nil
	}

	match := re.FindSubmatch(data)
	if len(match) != 1 {
		return ast.LocationRange{}, errors.New("unable to find assignment in local bind")
	}

	addrStartIndex := strings.LastIndex(string(data), string(lb.Variable)) + 1
	addrEndIndex := addrStartIndex + len(string(lb.Variable))

	begin, err := FindLocation(data, addrStartIndex)
	if err != nil {
		return ast.LocationRange{}, err
	}
	end, err := FindLocation(data, addrEndIndex)
	if err != nil {
		return ast.LocationRange{}, err
	}

	r := ast.LocationRange{
		Begin: begin,
		End:   end,
	}

	return r, nil
}

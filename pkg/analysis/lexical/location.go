package lexical

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Locatable struct {
	Token  interface{}
	Loc    ast.LocationRange
	Parent *Locatable
	Env    Env
}

func (l *Locatable) Resolve() (ast.LocationRange, error) {
	switch t := l.Token.(type) {
	case *ast.Var:
		name, err := tokenName(t)
		if err != nil {
			return ast.LocationRange{}, err
		}

		if ref, ok := l.Env[string(t.Id)]; ok {
			pointerName, err := tokenName(ref.Token)
			if err != nil {
				return ast.LocationRange{}, err
			}

			logrus.Infof("found reference for %s. It is %s at %s", name, pointerName, ref.Loc.String())
			return ref.Loc, nil
		} else {
			logrus.Warn("did not find ref")
		}
	default:
		logrus.Warnf("unable to resolve %T", l.Token)
	}

	return ast.LocationRange{}, nil
}

func (l *Locatable) IsFunctionParam() bool {
	if _, isVar := l.Token.(*ast.Var); isVar {
		if _, isParentLocal := l.Parent.Token.(*ast.Local); isParentLocal {
			return true
		}
	}

	return false
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
	return beforeRangeOrEqual(r1.Begin, r2) &&
		afterRangeOrEqual(r1.End, r2)
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

func beforeRangeOrEqual(l ast.Location, r ast.LocationRange) bool {
	begin := r.Begin
	if l.Line < begin.Line {
		return true
	} else if l.Line == begin.Line && l.Column <= begin.Column {
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

func localBindRange(source []byte, lb ast.LocalBind, parent *Locatable) (ast.LocationRange, error) {
	// pStart := parent.Loc.Begin.Line
	// pEnd := parent.Loc.End.Line

	// data, err := ExtractLines(source, pStart, pEnd)
	data, err := ExtractUntil(source, lb.Body.Loc().Begin)
	if err != nil {
		return ast.LocationRange{}, err
	}

	if string(lb.Variable) == "$" {
		return *lb.Body.Loc(), nil
	}

	expression := fmt.Sprintf(`(?m)\b%s(\(.*?\))?\s*=\s*\z`, string(lb.Variable))
	re, err := regexp.Compile(expression)
	if err != nil {
		return ast.LocationRange{}, err
	}

	match := re.FindAll(data, 1)
	// match := re.FindSubmatch(data)
	if len(match) != 1 {
		logrus.WithFields(logrus.Fields{
			"expression": expression,
			"var":        string(lb.Variable),
			"source":     string(data),
			"match":      spew.Sdump(match),
			"parent":     spew.Sdump(lb.Body),
		}).Error("couldn't find assignment")
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

package lexical

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CursorVisitor finds a node whose range some cursor lies in, or the
// closest node to it.
type CursorVisitor struct {
	SourceVisitor *SourceVisitor
	Location      ast.Location

	enclosingNode            *Locatable
	terminalNode             *Locatable
	terminalNodeOnCursorLine *Locatable
}

// NewCursorVisitor creates an instance of CursorVisitor.
func NewCursorVisitor(filename string, r io.Reader, loc ast.Location) (*CursorVisitor, error) {
	cv := &CursorVisitor{
		Location: loc,
	}

	v, err := NewSourceVisitor(filename, r, cv.previsit)
	if err != nil {
		return nil, err
	}

	cv.SourceVisitor = v
	cv.terminalNode = &Locatable{Token: v.Node, Loc: *v.Node.Loc()}

	return cv, nil
}

func (cv *CursorVisitor) Visit() error {
	return cv.SourceVisitor.Visit()
}

func (this *CursorVisitor) TokenAtPosition() (*Locatable, error) {
	logrus.Info("finding token")
	if this.enclosingNode == nil {
		if beforeRange(this.Location, *this.SourceVisitor.Node.Loc()) {
			return nil, errors.Errorf("before doc start")
		} else if afterRange(this.Location, this.terminalNode.Loc) {
			return nil, errors.Errorf("after doc end")
		}

		return nil, errors.New("no wrapping identifer was found, but node didn't lie outside of document range")
	} else if !isIdentifier(this.enclosingNode) {
		if this.terminalNodeOnCursorLine != nil && afterRange(this.Location, this.terminalNodeOnCursorLine.Loc) {
			return nil, errors.Errorf("AfterLineEnd: %#v, %#v", this.enclosingNode, this.terminalNodeOnCursorLine)
		}

		return nil, errors.Errorf("NotIdentifier: %#v, %#v", this.enclosingNode, this.terminalNodeOnCursorLine)
	}

	return this.enclosingNode, nil
}

func (cv *CursorVisitor) previsit(token, parent interface{}, env Env) error {
	var r ast.LocationRange
	var err error
	switch t := token.(type) {
	case ast.Node:
		r = cv.nodeRange(t)
	case ast.LocalBind:
		r, err = cv.localBindRange(t, parent)
	default:
		return errors.Errorf("can't find range for %T", t)
	}

	if err != nil {
		return err
	}

	nodeEnd := r.End

	l := &Locatable{Token: token, Loc: r}

	if inRange(cv.Location, r) {
		if cv.enclosingNode == nil || isRangeSmaller(cv.enclosingNode.Loc, r) {
			cv.enclosingNode = l
		}
	}

	if afterRangeOrEqual(nodeEnd, cv.terminalNode.Loc) {
		cv.terminalNode = l
	}

	if nodeEnd.Line == cv.Location.Line {
		if cv.terminalNodeOnCursorLine == nil {
			cv.terminalNodeOnCursorLine = nil
		} else if afterRangeOrEqual(nodeEnd, cv.terminalNodeOnCursorLine.Loc) {
			cv.terminalNodeOnCursorLine = nil
		}
	}

	return nil
}

func (cv *CursorVisitor) nodeRange(node ast.Node) ast.LocationRange {
	return *node.Loc()
}

var reLocalBind = `(?m)\s+foo3\s*=\s*\Z`

func (cv *CursorVisitor) localBindRange(lb ast.LocalBind, parent interface{}) (ast.LocationRange, error) {
	data, err := ExtractUntil(cv.SourceVisitor.Source, lb.Body.Loc().Begin)
	if err != nil {
		return ast.LocationRange{}, err
	}

	reLocalBind, err := regexp.Compile(fmt.Sprintf(`(?m)\s+%s\s*=\s*\z`, string(lb.Variable)))
	if err != nil {
		return ast.LocationRange{}, err
	}

	match := reLocalBind.FindSubmatch(data)
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

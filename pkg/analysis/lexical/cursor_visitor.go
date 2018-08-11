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
	NodeVisitor *NodeVisitor
	Location    ast.Location

	enclosingNode            *Locatable
	terminalNode             *Locatable
	terminalNodeOnCursorLine *Locatable
}

// NewCursorVisitor creates an instance of CursorVisitor.
func NewCursorVisitor(filename string, r io.Reader, loc ast.Location) (*CursorVisitor, error) {
	cv := &CursorVisitor{
		Location: loc,
	}

	v, err := NewNodeVisitor(filename, r, cv.previsit)
	if err != nil {
		return nil, err
	}

	cv.NodeVisitor = v
	cv.terminalNode = &Locatable{Token: v.Node, Loc: *v.Node.Loc()}

	return cv, nil
}

func (cv *CursorVisitor) Visit() error {
	return cv.NodeVisitor.Visit()
}

func (this *CursorVisitor) TokenAtPosition() (*Locatable, error) {
	logrus.Debugf("finding token in a %T", this.enclosingNode.Token)
	if this.enclosingNode == nil {
		if beforeRange(this.Location, *this.NodeVisitor.Node.Loc()) {
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

// nolint: gocyclo
func (cv *CursorVisitor) previsit(token interface{}, parent *Locatable, env Env) error {
	var r ast.LocationRange
	var err error
	switch t := token.(type) {
	case RequiredParameter:
		r = ast.LocationRange{}
	case ast.DesugaredObjectField:
		r, err = cv.desugaredObjectFieldRange(t, parent)
	case ast.Identifier:
		r, err = cv.identifierRange(t, parent)
	case *ast.Identifier:
		if t == nil {
			return errors.Errorf("identifier is nil")
		}
		r, err = cv.identifierRange(*t, parent)
	case ast.LocalBind:
		r, err = cv.localBindRange(t, parent)
	case ast.Node:
		r, err = cv.nodeRange(t, parent)
	default:
		return errors.Errorf("can't find range for %T", t)
	}

	if err != nil {
		return err
	}

	nodeEnd := r.End

	l := &Locatable{Token: token, Loc: r, Parent: parent}

	if inRange(cv.Location, r) {
		if cv.enclosingNode == nil {
			cv.enclosingNode = l
		} else if isRangeSmaller(cv.enclosingNode.Loc, r) {
			logrus.Debugf("setting token %T as enclosing node because %s is smaller than %s (%T)",
				l.Token, r.String(), cv.enclosingNode.Loc.String(), cv.enclosingNode.Token)

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
			cv.terminalNodeOnCursorLine = &Locatable{
				Token: token,
				Loc:   r,
			}
		}
	}

	return nil
}

func (cv *CursorVisitor) desugaredObjectFieldRange(f ast.DesugaredObjectField, parent *Locatable) (ast.LocationRange, error) {
	if parent == nil {
		return ast.LocationRange{}, errors.New("field has not parent")
	}

	start := parent.Loc.Begin.Line
	end := parent.Loc.End.Line

	rangeText, err := ExtractLines(cv.NodeVisitor.Source, start, end)
	if err != nil {
		return ast.LocationRange{}, err
	}

	// TODO get value from a node
	fieldName := ""
	switch t := f.Name.(type) {
	case *ast.LiteralString:
		fieldName = t.Value
	default:
		return ast.LocationRange{}, errors.Errorf("unable to get desugared field name from type %T", t)
	}

	r, err := fieldRange(fieldName, string(rangeText))
	if err != nil {
		return ast.LocationRange{}, err
	}

	r.Begin.Line += start - 1
	r.End.Line += start - 1

	return r, nil
}

func (cv *CursorVisitor) identifierRange(id ast.Identifier, parent interface{}) (ast.LocationRange, error) {
	return ast.LocationRange{}, nil
}

func (cv *CursorVisitor) nodeRange(node ast.Node, parent *Locatable) (ast.LocationRange, error) {
	if node.Loc() == nil {
		return ast.LocationRange{}, errors.New("node range is nil")
	}
	return *node.Loc(), nil
}

func (cv *CursorVisitor) localBindRange(lb ast.LocalBind, parent interface{}) (ast.LocationRange, error) {
	data, err := ExtractUntil(cv.NodeVisitor.Source, lb.Body.Loc().Begin)
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

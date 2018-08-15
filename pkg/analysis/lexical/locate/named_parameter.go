package locate

import (
	"bytes"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NamedParameter locates an astext.RequiredParameter.
func NamedParameter(p ast.NamedParameter, parentRange ast.LocationRange, source string) (ast.LocationRange, error) {
	parentSource, err := extractRange(source, parentRange)
	if err != nil {
		return ast.LocationRange{}, err
	}

	if parentSource == "" {
		logrus.Debug(parentRange.String())
		return ast.LocationRange{}, errors.New("could not find source for named parameter parent")
	}

	var val bytes.Buffer
	if _, err = val.WriteString(string(p.Name)); err != nil {
		return ast.LocationRange{}, err
	}

	if p.DefaultArg != nil {
		da := astext.TokenValue(p.DefaultArg)
		if _, err = val.WriteString("=" + da); err != nil {
			return ast.LocationRange{}, err
		}
	}

	id := val.String()
	inArgs := false
	for i := 0; i < len(parentSource); i++ {
		s := parentSource[i]

		switch string(s) {
		case "(":
			inArgs = true
			continue
		case ")":
			inArgs = false
			continue
		}

		if inArgs {
			if len(parentSource) > i+len(id) {
				if parentSource[i:i+len(id)] == id {
					argLocation, err := findLocation(parentSource, i)
					if err != nil {
						return ast.LocationRange{}, err
					}

					r := createRange(
						parentRange.FileName,
						argLocation.Line+parentRange.Begin.Line-1,
						argLocation.Column+parentRange.Begin.Column-1,
						argLocation.Line+parentRange.Begin.Line-1,
						argLocation.Column+parentRange.Begin.Column+len(id)-1,
					)
					return r, nil
				}
			}
		}
	}

	return ast.LocationRange{}, errors.Errorf("unable to find optional parameter %q", string(p.Name))
}

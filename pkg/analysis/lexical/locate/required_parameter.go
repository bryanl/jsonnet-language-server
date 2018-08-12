package locate

import (
	"fmt"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/astext"
	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// RequiredParameter locates an astext.RequiredParameter.
func RequiredParameter(p astext.RequiredParameter, parentRange ast.LocationRange, source string) (ast.LocationRange, error) {
	parentSource, err := extractRange(source, parentRange)
	if err != nil {
		return ast.LocationRange{}, err
	}

	if parentSource == "" {
		logrus.Info(parentRange.String())
		return ast.LocationRange{}, errors.New("could not find source for parameter parent")
	}

	id := string(p.ID)
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
						argLocation.Line, argLocation.Column+parentRange.Begin.Column,
						argLocation.Line, argLocation.Column+parentRange.Begin.Column,
					)
					return r, nil
				}
			}
		}
	}

	fmt.Println(source, parentRange)
	return ast.LocationRange{}, errors.Errorf("unable to find parameter %q", string(p.ID))
}

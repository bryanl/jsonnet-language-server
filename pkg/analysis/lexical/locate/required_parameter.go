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

	inArgs := false
	for i, s := range parentSource {
		switch string(s) {
		case "(":
			inArgs = true
			continue
		case ")":
			inArgs = false
			continue
		}

		if inArgs {
			if string(p.ID) == string(s) {
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

	fmt.Println(source, parentRange)
	return ast.LocationRange{}, errors.Errorf("unable to find parameter %s", string(p.ID))
}

package locate

import (
	"fmt"

	"github.com/google/go-jsonnet/ast"
	"github.com/sirupsen/logrus"
)

func ForSpec(a ast.ForSpec, parentRange ast.LocationRange, source string) (ast.LocationRange, error) {
	logrus.Infof("parent range is %s", parentRange.String())
	parentSource, err := extractRange(source, parentRange)
	if err != nil {
		return ast.LocationRange{}, err
	}

	inFor := false
	start := 0
	for i := 0; i < len(parentSource); i++ {
		c := parentSource[i]
		switch string(c) {
		case "f":
			if !inFor && i+2 < len(parentSource) && parentSource[i:i+3] == "for" {
				start = i
				inFor = true
			}
		}
	}

	startLocation, err := findLocation2(parentSource, start)
	if err != nil {
		return ast.LocationRange{}, err
	}

	fmt.Println("startLocation:", start, startLocation.String())
	end := len(parentSource) - 2
	endLocation, err := findLocation2(parentSource, end)
	if err != nil {
		return ast.LocationRange{}, err
	}

	r := createRange(
		parentRange.FileName,
		startLocation.Line+parentRange.Begin.Line-1,
		startLocation.Column,
		endLocation.Line+parentRange.Begin.Line-1,
		endLocation.Column,
	)

	return r, nil
}

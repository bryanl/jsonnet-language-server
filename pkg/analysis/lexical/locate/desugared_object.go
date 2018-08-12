package locate

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

func DesugaredObjectField(field ast.DesugaredObjectField, parentRange ast.LocationRange, source string) (ast.LocationRange, error) {
	parentSource, err := extractRange(source, parentRange)
	if err != nil {
		return ast.LocationRange{}, err
	}

	// TODO get value from a node
	fieldName := ""
	switch t := field.Name.(type) {
	case *ast.LiteralString:
		fieldName = t.Value
	default:
		return ast.LocationRange{}, errors.Errorf("unable to get desugared field name from type %T", t)
	}

	fieldLocation, err := fieldRange(fieldName, parentSource)
	if err != nil {
		return ast.LocationRange{}, err
	}

	fieldLocation.FileName = parentRange.FileName
	fieldLocation.Begin.Line += parentRange.Begin.Line - 1
	fieldLocation.End.Line += parentRange.Begin.Line - 1

	return fieldLocation, nil
}

func extractRange(source string, r ast.LocationRange) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(source))
	scanner.Split(bufio.ScanRunes)

	var buf bytes.Buffer

	col := 1
	line := 1

	for scanner.Scan() {
		cur := scanner.Text()
		if cur == "\n" {
			line++
			col = 1
		}

		loc := ast.Location{Line: line, Column: col}
		if inRange(loc, r) {
			if _, err := buf.WriteString(cur); err != nil {
				return "", err
			}
		}

		col++
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func inRange(l ast.Location, r ast.LocationRange) bool {
	if r.Begin.Line == l.Line {
		return r.Begin.Column <= l.Column
	} else if r.Begin.Line <= l.Line && r.End.Line >= l.Line {
		return true
	}

	return false
}

// TODO move test from pkg/analysis/lexical to here
func fieldRange(fieldName, source string) (ast.LocationRange, error) {
	re, err := regexp.Compile(fmt.Sprintf(`(?m)\b%s\b.*?:{1,3}\s*`, fieldName))
	if err != nil {
		return ast.LocationRange{}, err
	}

	match := re.FindStringSubmatch(source)
	if len(match) != 1 {
		return ast.LocationRange{}, errors.Errorf("unable to find field %s", fieldName)
	}

	fieldStartIndex := strings.LastIndex(source, match[0])
	begin, err := findLocation(source, fieldStartIndex)
	if err != nil {
		return ast.LocationRange{}, err
	}

	valueStartIndex := fieldStartIndex + len(match[0])

	done := false
	inDoubleQuote := false
	inBlockQuote := false
	bracketLevel := 0
	braceLevel := 0
	count := 0
	for i := valueStartIndex; i < len(source); i++ {
		switch source[i] {
		case '"':
			if i > 1 && source[i-1] != '\\' {
				inDoubleQuote = !inDoubleQuote
			}
		case '|':
			// TODO this is not compliant with the full spec at https://jsonnet.org/ref/spec.html
			if source[i+1:i+3] == "||" {
				if !inBlockQuote {
					i = i + 2
				}
				inBlockQuote = !inBlockQuote
			}
		case '[':
			bracketLevel++
		case ']':
			bracketLevel--
		case '{':
			braceLevel++
		case '}':
			braceLevel--
		case ',':
			if inDoubleQuote ||
				inBlockQuote ||
				bracketLevel != 0 ||
				braceLevel != 0 {
				continue
			}
			count = i
			done = true
			break
		}

		if done {
			break
		}
	}

	end, err := findLocation(source, count)
	if err != nil {
		return ast.LocationRange{}, err
	}

	r := ast.LocationRange{
		Begin: begin,
		End:   end,
	}

	return r, nil
}

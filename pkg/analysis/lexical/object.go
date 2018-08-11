package lexical

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

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
	begin, err := FindLocation([]byte(source), fieldStartIndex)
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

	end, err := FindLocation([]byte(source), count)
	if err != nil {
		return ast.LocationRange{}, err
	}

	r := ast.LocationRange{
		Begin: begin,
		End:   end,
	}

	return r, nil
}

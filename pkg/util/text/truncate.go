package text

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/util/position"
)

// Truncate returns text truncated at a position.
func Truncate(source string, p position.Position) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(source))
	scanner.Split(bufio.ScanBytes)

	var buf bytes.Buffer

	c := 0
	l := 1

	for scanner.Scan() {
		c++

		t := scanner.Text()

		_, err := buf.WriteString(t)
		if err != nil {
			return "", err
		}

		if l == p.Line() && c == p.Column() {
			break
		}

		if t == "\n" {
			l++
			c = 0
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.TrimRight(buf.String(), "\n"), nil
}

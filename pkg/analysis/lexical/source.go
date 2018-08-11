package lexical

import (
	"bufio"
	"bytes"

	"github.com/google/go-jsonnet/ast"
	"github.com/pkg/errors"
)

// ExtractUntil extracts data from data until it gets to loc.
func ExtractUntil(data []byte, loc ast.Location) ([]byte, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Split(bufio.ScanRunes)

	var buf bytes.Buffer

	c := 0
	l := 1

	for scanner.Scan() {
		c++

		t := scanner.Text()

		if t == "\n" {
			l++
			c = 0
		}

		_, err := buf.WriteString(t)
		if err != nil {
			return nil, err
		}

		if l == loc.Line && c == loc.Column-1 {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ExtractCount extracts `count` runes from data. If count is negative,
// it works from the end of the data.
func ExtractCount(data []byte, count int) ([]byte, error) {
	if count == 0 {
		return []byte{}, nil
	} else if len(data) < count {
		return nil, errors.Errorf("count is larger than data length")
	}

	isReverse := false
	out := data
	if count < 0 {
		isReverse = true
		out = reverseBytes(data)
		count = 0 - count
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Split(bufio.ScanRunes)

	var buf bytes.Buffer
	i := 1

	for scanner.Scan() {
		_, err := buf.WriteString(scanner.Text())
		if err != nil {
			return nil, err
		}
		if i == count {
			break
		}
		i++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if isReverse {
		return reverseBytes(buf.Bytes()), nil
	}
	return buf.Bytes(), nil
}

// FindLocation finds the location count characters in data.
func FindLocation(data []byte, count int) (ast.Location, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Split(bufio.ScanRunes)

	c := 1
	l := 1

	i := 1

	for scanner.Scan() {
		t := scanner.Text()

		if t == "\n" {
			c = 1
			l++
		}

		if i == count {
			break
		}

		i++
		c++
	}

	if err := scanner.Err(); err != nil {
		return ast.Location{}, err
	}

	if count != i {
		return ast.Location{}, errors.Errorf("count didn't match index %d vs %d",
			count, i)
	}

	return ast.Location{
		Line:   l,
		Column: c,
	}, nil
}

func reverseBytes(data []byte) []byte {
	out := make([]byte, len(data))
	copy(out, data)

	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	return out
}

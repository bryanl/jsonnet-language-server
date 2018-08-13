package lexical

import (
	"fmt"
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/stretchr/testify/assert"
)

func Test_inRange(t *testing.T) {
	name := func(l ast.Location, r ast.LocationRange) string {
		return fmt.Sprintf("%s in %s-%s",
			l.String(),
			r.Begin.String(), r.End.String())
	}

	cases := []struct {
		loc      ast.Location
		lr       ast.LocationRange
		expected bool
	}{
		{
			loc:      createLoc(5, 3),
			lr:       createRange(5, 1, 5, 10),
			expected: true,
		},
		{
			loc:      createLoc(5, 3),
			lr:       createRange(1, 1, 9, 10),
			expected: true,
		},
		{
			loc:      createLoc(5, 3),
			lr:       createRange(1, 1, 5, 4),
			expected: true,
		},
		{
			loc:      createLoc(5, 3),
			lr:       createRange(1, 1, 2, 2),
			expected: false,
		},
		{
			loc:      createLoc(5, 3),
			lr:       createRange(1, 1, 2, 2),
			expected: false,
		},
		{
			loc:      createLoc(2, 17),
			lr:       createRange(2, 7, 2, 10),
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(name(tc.loc, tc.lr), func(t *testing.T) {
			got := inRange(tc.loc, tc.lr)
			assert.Equal(t, tc.expected, got)
		})
	}

}

func Test_isRangeSmaller(t *testing.T) {
	name := func(r1, r2 ast.LocationRange) string {
		return fmt.Sprintf("%s-%s %s-%s",
			r1.Begin.String(), r1.End.String(),
			r2.Begin.String(), r2.End.String())
	}

	cases := []struct {
		r1       ast.LocationRange
		r2       ast.LocationRange
		expected bool
	}{
		{
			r1:       createRange(1, 10, 1, 20),
			r2:       createRange(1, 15, 1, 17),
			expected: true,
		},
		{
			r1:       createRange(2, 7, 2, 10),
			r2:       createRange(2, 14, 2, 20),
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(name(tc.r1, tc.r2), func(t *testing.T) {
			got := isRangeSmaller(tc.r1, tc.r2)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func Test_afterRangeOrEqual(t *testing.T) {
	name := func(l ast.Location, r ast.LocationRange) string {
		return fmt.Sprintf("%s after %s-%s",
			l.String(),
			r.Begin.String(), r.End.String())
	}

	cases := []struct {
		l        ast.Location
		r        ast.LocationRange
		expected bool
	}{
		{
			l:        createLoc(3, 7),
			r:        createRange(2, 1, 2, 6),
			expected: true,
		},
		{
			l:        createLoc(1, 7),
			r:        createRange(1, 1, 1, 6),
			expected: true,
		},
		{
			l:        createLoc(1, 6),
			r:        createRange(1, 1, 1, 6),
			expected: true,
		},
		{
			l:        createLoc(1, 7),
			r:        createRange(2, 1, 2, 6),
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(name(tc.l, tc.r), func(t *testing.T) {
			got := afterRangeOrEqual(tc.l, tc.r)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func Test_beforeRange(t *testing.T) {
	name := func(l ast.Location, r ast.LocationRange) string {
		return fmt.Sprintf("%s before %s-%s",
			l.String(),
			r.Begin.String(), r.End.String())
	}

	cases := []struct {
		l        ast.Location
		r        ast.LocationRange
		expected bool
	}{
		{
			l:        createLoc(1, 7),
			r:        createRange(2, 1, 2, 6),
			expected: true,
		},
		{
			l:        createLoc(1, 4),
			r:        createRange(1, 5, 1, 6),
			expected: true,
		},
		{
			l:        createLoc(3, 7),
			r:        createRange(2, 1, 2, 6),
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(name(tc.l, tc.r), func(t *testing.T) {
			got := beforeRange(tc.l, tc.r)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func Test_afterRange(t *testing.T) {
	name := func(l ast.Location, r ast.LocationRange) string {
		return fmt.Sprintf("%s after %s-%s",
			l.String(),
			r.Begin.String(), r.End.String())
	}

	cases := []struct {
		l        ast.Location
		r        ast.LocationRange
		expected bool
	}{
		{
			l:        createLoc(3, 7),
			r:        createRange(2, 1, 2, 6),
			expected: true,
		},
		{
			l:        createLoc(1, 7),
			r:        createRange(1, 1, 1, 6),
			expected: true,
		},
		{
			l:        createLoc(1, 7),
			r:        createRange(2, 1, 2, 6),
			expected: false,
		},
		{
			l:        createLoc(2, 6),
			r:        createRange(2, 1, 2, 6),
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(name(tc.l, tc.r), func(t *testing.T) {
			got := afterRange(tc.l, tc.r)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func createRange(r1l, r1c, r2l, r2c int) ast.LocationRange {
	return ast.LocationRange{
		Begin: createLoc(r1l, r1c),
		End:   createLoc(r2l, r2c),
	}
}

func createLoc(line, column int) ast.Location {
	return ast.Location{Line: line, Column: column}
}

package main

import (
	"flag"
	"log"
	"os"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-jsonnet/ast"
)

func main() {
	filename := flag.String("filename", "", "filename")
	line := flag.Int("l", 0, "line")
	char := flag.Int("c", 0, "character")

	flag.Parse()

	if *filename == "" {
		log.Fatalf("invalid file name")
	}

	if *line == 0 || *char == 0 {
		log.Fatalf("invalid pos")
	}

	req := request{Filename: *filename, Line: *line, Char: *char}
	if err := run(req); err != nil {
		log.Fatalf(err.Error())
	}
}

type request struct {
	Filename string
	Line     int
	Char     int
}

func run(req request) error {
	f, err := os.Open(req.Filename)
	if err != nil {
		return err
	}

	loc := ast.Location{
		Line:   req.Line,
		Column: req.Char,
	}

	locatable, err := lexical.TokenAtLocation(req.Filename, f, loc)
	if err != nil {
		return err
	}

	spew.Dump(locatable)

	return nil
}

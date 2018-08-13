package main

import (
	"flag"
	"io/ioutil"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/parser"
)

func main() {
	filename := flag.String("filename", "", "filename")
	flag.Parse()

	if *filename == "" {
		log.Fatal("usage: jsonnet-dumper -filename <filename>")
	}

	data, err := ioutil.ReadFile(*filename)
	if err != nil {
		log.Fatal(err)
	}

	n, err := parse(*filename, string(data))
	if err != nil {
		log.Fatal(err)
	}

	spew.Dump(n)
}

func parse(filename, snippet string) (ast.Node, error) {
	tokens, err := parser.Lex(filename, snippet)
	if err != nil {
		return nil, err
	}
	node, err := parser.Parse(tokens)
	if err != nil {
		return nil, err
	}

	return node, nil
}

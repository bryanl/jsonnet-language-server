package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"
	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/token"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/parser"
)

func main() {
	filename := flag.String("filename", "", "filename")
	level := flag.Int("level", 1, "dump level: 1) lex 2) parse 3) desugar/analyze")
	flag.Parse()

	if *filename == "" {
		log.Fatal("usage: jsonnet-dumper -filename <filename>")
	}

	data, err := ioutil.ReadFile(*filename)
	if err != nil {
		log.Fatal(err)
	}

	switch *level {
	case 0:
		lex(*filename, string(data))
	case 1:
		n, err := parse(*filename, string(data))
		if err != nil {
			log.Fatal(err)
		}

		spew.Dump(n)
	case 2:
		m, err := token.NewMatch(*filename, string(data))
		if err != nil {
			log.Fatal(err)
		}

		n, err := m.Expr(0)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(m.Tokens[0 : n+1])

	default:
		log.Fatalf("unsupport option %d", *level)
	}

}

func lex(filename, snippet string) {
	tokens, err := locate.Lex(filename, snippet)
	if err != nil {
		log.Fatal(err)
	}

	for i, t := range tokens {
		fmt.Printf("%d %s: %s = %s\n", i, t.Loc.String(), t.Kind.String(), t.Data)
	}
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

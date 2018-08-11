package main

import (
	"flag"
	"io/ioutil"
	"log"

	"github.com/davecgh/go-spew/spew"
	jsonnet "github.com/google/go-jsonnet"
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

	n, err := jsonnet.SnippetToAST(*filename, string(data))
	if err != nil {
		log.Fatal(err)
	}

	spew.Dump(n)
}

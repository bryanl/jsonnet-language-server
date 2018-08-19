package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical/locate"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/profile"
	"github.com/sirupsen/logrus"
)

func main() {
	var jlibPaths arrayFlags
	flag.Var(&jlibPaths, "J", "jsonnet lib path")

	filename := flag.String("filename", "", "filename")
	line := flag.Int("l", 0, "line")
	char := flag.Int("c", 0, "character")
	debug := flag.Bool("d", false, "debug")
	cpuProf := flag.Bool("p", false, "enable CPU profiling")
	memProf := flag.Bool("m", false, "enable memory profiling")

	flag.Parse()

	if *cpuProf {
		defer profile.Start().Stop()
	}

	if *memProf {
		defer profile.Start(profile.MemProfile).Stop()
	}

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if *filename == "" {
		log.Fatalf("invalid file name")
	}

	if *line == 0 || *char == 0 {
		log.Fatalf("invalid pos")
	}

	req := request{Filename: *filename, Line: *line, Char: *char}
	if err := run(req); err != nil {
		logrus.Fatalf(err.Error())
	}
}

type request struct {
	Filename  string
	Line      int
	Char      int
	jlibPaths []string
}

func run(req request) error {
	f, err := os.Open(req.Filename)
	if err != nil {
		return err
	}

	nodeCache := locate.NewNodeCache()

	response, err := lexical.HoverAtLocation(req.Filename, f, req.Line, req.Char, req.jlibPaths, nodeCache)
	if err != nil {
		return err
	}

	spew.Dump(response)

	return nil
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ", ")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

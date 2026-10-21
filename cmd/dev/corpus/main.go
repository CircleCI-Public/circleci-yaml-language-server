// Command corpus checks the parser against a corpus of configs: this repo's
// testdata, any configs survey fetched, and the reference corpus, a local
// directory of the reference compiler's test cases.
//
// Usage:
//
//	go run ./cmd/dev/corpus parse [-corpus dir] [-repo dir] > run.jsonl
//	go run ./cmd/dev/corpus templates [-corpus dir] > run.jsonl
//
// parse writes one JSON line for each problem it finds in a config: a panic,
// a difference between ParseForCompile and ParseFromContent, or a when,
// unless or with_tool_cache step that CompileInfo doesn't hold. The lines
// have the shape survey diagnose writes, so survey compare diffs two runs.
//
// templates renders the reference corpus's replacer fixtures, and evaluates
// the workflow conditions of its logic cases, writing a line for each that
// differs from the reference compiler.
//
// Environment:
//
//	REFERENCE_CORPUS_DIR  the reference corpus, read for every build.yml
//	                      under it, unless -corpus is given
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	var err error
	switch os.Args[1] {
	case "parse":
		err = parse(os.Args[2:])
	case "templates":
		err = templates(os.Args[2:])
	default:
		usage()
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "corpus:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: corpus parse [-corpus dir] [-repo dir]")
	fmt.Fprintln(os.Stderr, "       corpus templates [-corpus dir]")
	os.Exit(2)
}

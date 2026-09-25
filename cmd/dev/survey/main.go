// Command survey runs the language server's diagnostics over the CircleCI
// configs of whole GitHub organizations, to measure how a change moves the
// errors it reports on real configs, and to find false ones.
//
// Usage:
//
//	go run ./cmd/dev/survey fetch [-out dir] [-since 2024-09-01] [org...]
//	go run ./cmd/dev/survey diagnose [dir] > run.jsonl
//	go run ./cmd/dev/survey compare before.jsonl after.jsonl
//
// fetch downloads the YAML files under .circleci/ of each repository in the
// organizations (circleci and CircleCI-Public by default) that isn't archived
// or a fork and was pushed to since the date, into dir/owner__repo/. The
// files can come from private repositories, so dir defaults to bin/survey,
// which git ignores; don't commit them.
//
// diagnose writes one JSON line for each diagnostic on each file under dir.
// compare prints how many errors each run has, in how many files, and which
// errors one has that the other doesn't.
//
// Environment:
//
//	GH_TOKEN, GITHUB_TOKEN  a GitHub token, which fetch needs to list an
//	                        organization's private repositories, and which
//	                        diagnose uses to fetch private orbs from GitHub
//	CIRCLE_TOKEN            used by diagnose when set, for private orbs,
//	                        contexts and the like
package main

import (
	"fmt"
	"os"
	"strings"
)

// defaultDir is where fetch writes and diagnose reads.
const defaultDir = "bin/survey"

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	var err error
	switch os.Args[1] {
	case "fetch":
		err = fetch(os.Args[2:])
	case "diagnose":
		err = diagnose(os.Args[2:])
	case "compare":
		err = compare(os.Args[2:])
	default:
		usage()
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "survey:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: survey fetch [-out dir] [-since date] [org...] | diagnose [dir] | compare before.jsonl after.jsonl")
	os.Exit(2)
}

// gitHubToken is the GitHub token in the environment, under the names the
// GitHub CLI reads.
func gitHubToken() string {
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token
	}
	return os.Getenv("GITHUB_TOKEN")
}

func isYAML(name string) bool {
	return strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")
}

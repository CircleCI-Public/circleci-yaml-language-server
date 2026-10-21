package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/google/go-cmp/cmp"
	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

// row is one problem in a config, with the fields survey compare reads.
type row struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Severity int    `json:"severity"`
	Message  string `json:"message"`
	Text     string `json:"text"`
	// Check is panic, equivalence or coverage.
	Check string `json:"check"`
}

func parse(args []string) error {
	flags := flag.NewFlagSet("parse", flag.ExitOnError)
	corpus := flags.String("corpus", os.Getenv("REFERENCE_CORPUS_DIR"), "the reference corpus")
	repo := flags.String("repo", ".", "the root of this repo")
	_ = flags.Parse(args)

	configs, err := repoConfigs(*repo)
	if err != nil {
		return err
	}
	if *corpus == "" {
		fmt.Fprintln(os.Stderr, "REFERENCE_CORPUS_DIR is unset, so only this repo's configs are checked")
	} else {
		reference, err := referenceConfigs(*corpus)
		if err != nil {
			return err
		}
		configs = append(configs, reference...)
	}

	enc := json.NewEncoder(os.Stdout)
	failing := map[string]int{}
	for _, c := range configs {
		content, err := os.ReadFile(c.path())
		if err != nil {
			return err
		}
		checks := map[string]bool{}
		for _, r := range checkParse(c.name, content) {
			checks[r.Check] = true
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		for check := range checks {
			failing[check]++
		}
	}

	fmt.Fprintf(os.Stderr, "%d configs: %d panic, %d not equivalent, %d with coverage mismatches\n",
		len(configs), failing["panic"], failing["equivalence"], failing["coverage"])
	return nil
}

// checkParse parses a config with ParseFromContent and ParseForCompile, and
// returns what's wrong.
func checkParse(file string, content []byte) []row {
	lines := strings.Split(string(content), "\n")
	var res []row
	report := func(check string, line int, message string) {
		text := ""
		if line > 0 && line <= len(lines) {
			text = strings.TrimSpace(lines[line-1])
		}
		res = append(res, row{file, line, int(protocol.DiagnosticSeverityError), message, text, check})
	}

	settings := &session.Settings{}
	want, err := parseSafely("ParseFromContent", func() (parser.YamlDocument, error) {
		return parser.ParseFromContent(content, settings, "", protocol.Position{})
	})
	if err != nil {
		report("panic", 0, err.Error())
		return res
	}
	defer want.Close()
	got, err := parseSafely("ParseForCompile", func() (parser.YamlDocument, error) {
		return parser.ParseForCompile(content, file, settings)
	})
	if err != nil {
		report("panic", 0, err.Error())
		return res
	}
	defer got.Close()

	// The protocol's Optional and DiagnosticTags hide their values.
	protocolFields := cmp.Exporter(func(t reflect.Type) bool { return t.PkgPath() == "go.lsp.dev/protocol" })
	for _, field := range []struct {
		name      string
		got, want any
	}{
		{"Jobs", got.Jobs, want.Jobs},
		{"Commands", got.Commands, want.Commands},
		{"Workflows", got.Workflows, want.Workflows},
		{"Conditions", got.Conditions, want.Conditions},
		{"StepConditions", got.StepConditions, want.StepConditions},
		{"FilterExpressions", got.FilterExpressions, want.FilterExpressions},
		{"FlattenedSteps", got.FlattenedSteps, want.FlattenedSteps},
		{"Diagnostics", *got.Diagnostics, *want.Diagnostics},
	} {
		if diff := cmp.Diff(field.want, field.got, protocolFields); diff != "" {
			report("equivalence", 0, fmt.Sprintf("ParseForCompile differs in %s (-ParseFromContent +ParseForCompile)\n%s", field.name, diff))
		}
	}

	var root yaml.Node
	if yaml.Unmarshal(content, &root) != nil {
		// The reference compiler rejects it too, so there are no steps to
		// count.
		return res
	}
	written := map[condAt]int{}
	writtenConds(&root, written)
	recorded := map[condAt]int{}
	recordedConds(got.Compile(), map[protocol.Range]bool{}, recorded)

	var at []condAt
	for key := range written {
		at = append(at, key)
	}
	for key := range recorded {
		if _, ok := written[key]; !ok {
			at = append(at, key)
		}
	}
	slices.SortFunc(at, func(a, b condAt) int {
		if a.line != b.line {
			return a.line - b.line
		}
		return strings.Compare(a.kind, b.kind)
	})
	for _, key := range at {
		switch n := written[key] - recorded[key]; {
		case n > 0:
			report("coverage", key.line, fmt.Sprintf("%d %s step(s) not in CompileInfo", n, key.kind))
		case n < 0:
			report("coverage", key.line, fmt.Sprintf("CompileInfo has %d %s step(s) not written here", -n, key.kind))
		}
	}
	return res
}

func parseSafely(name string, parse func() (parser.YamlDocument, error)) (doc parser.YamlDocument, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s panics: %v", name, r)
		}
	}()
	return parse()
}

// condAt is where a conditional step's key is, with lines 1-indexed.
type condAt struct {
	line int
	kind string
}

// writtenConds counts the conditional steps in node: the items of any
// sequence that map when, unless or with_tool_cache to a mapping. Aliases
// aren't followed, so each is counted where it's written.
func writtenConds(node *yaml.Node, found map[condAt]int) {
	if node.Kind == yaml.SequenceNode {
		for _, item := range node.Content {
			if item.Kind != yaml.MappingNode || len(item.Content) != 2 || item.Content[1].Kind != yaml.MappingNode {
				continue
			}
			switch key := item.Content[0]; key.Value {
			case "when", "unless", "with_tool_cache":
				found[condAt{key.Line, key.Value}]++
			}
		}
	}
	for _, child := range node.Content {
		writtenConds(child, found)
	}
}

// recordedConds counts the conditions in info's step lists and its inline
// orbs'. A nested list is both recorded and held by its parent, so each
// condition is counted once by its range.
func recordedConds(info *parser.CompileInfo, seen map[protocol.Range]bool, found map[condAt]int) {
	var walk func(nodes []parser.StepNode)
	walk = func(nodes []parser.StepNode) {
		for _, node := range nodes {
			if node.Cond == nil {
				continue
			}
			if !seen[node.Cond.Range] {
				seen[node.Cond.Range] = true
				found[condAt{int(node.Cond.Range.Start.Line) + 1, node.Cond.Kind}]++
			}
			walk(node.Cond.Steps)
		}
	}
	for _, nodes := range info.Steps {
		walk(nodes)
	}
	for _, orb := range info.Orbs {
		recordedConds(orb, seen, found)
	}
}

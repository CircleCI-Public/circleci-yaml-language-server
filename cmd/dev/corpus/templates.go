package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/compile"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

func templates(args []string) error {
	flags := flag.NewFlagSet("templates", flag.ExitOnError)
	corpus := flags.String("corpus", os.Getenv("REFERENCE_CORPUS_DIR"), "the reference corpus")
	_ = flags.Parse(args)
	if *corpus == "" {
		return fmt.Errorf("REFERENCE_CORPUS_DIR is unset")
	}

	rendered, cases, err := checkTemplates(filepath.Join(*corpus, "replacer"))
	if err != nil {
		return err
	}
	conditions, workflows, err := checkWorkflowConditions(filepath.Join(*corpus, "logic"))
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	for _, r := range slices.Concat(rendered, conditions) {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "%d of %d template cases and %d of %d workflows differ\n", len(rendered), cases, len(conditions), workflows)
	return nil
}

// templateCase is one case of a replacer fixture: the variables, and what
// the template renders to with them.
type templateCase struct {
	env  compile.Env
	want string
}

var (
	templateEnd  = regexp.MustCompile(`^={5,}$`)
	caseEnd      = regexp.MustCompile(`^-{5,}$`)
	lineEndings  = regexp.MustCompile(`\r?\n`)
	edn          = regexp.MustCompile(`^\s*(?:"((?:[^"\\]|\\.)*)"|(nil|true|false)|(-?\d+))`)
	ednSeparator = regexp.MustCompile(`^[\s,]*`)
)

// readFixture reads a replacer fixture: the template, a line of =, then
// cases separated by lines of -. A case is an EDN map of variables, a blank
// line, and the expected output.
func readFixture(content string) (string, []templateCase, error) {
	lines := lineEndings.Split(content, -1)
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	end := slices.IndexFunc(lines, templateEnd.MatchString)
	if end < 0 {
		return "", nil, fmt.Errorf("no line of = after the template")
	}
	text := strings.Join(lines[:end], "\n")

	var cases []templateCase
	body := lines[end+1:]
	for len(body) != 0 {
		n := slices.IndexFunc(body, caseEnd.MatchString)
		if n < 0 {
			n = len(body)
		}
		c := body[:n]
		blank := slices.IndexFunc(c, func(s string) bool { return strings.TrimSpace(s) == "" })
		if blank < 0 {
			return "", nil, fmt.Errorf("a case has no blank line after its variables")
		}
		env, err := readEDNMap(strings.Join(c[:blank], "\n"))
		if err != nil {
			return "", nil, err
		}
		cases = append(cases, templateCase{env: env, want: strings.Join(c[blank+1:], "\n")})
		if n == len(body) {
			break
		}
		body = body[n+1:]
	}
	return text, cases, nil
}

// readEDNMap reads an EDN map of strings to strings, nil, booleans or
// integers, which is all the fixtures use.
func readEDNMap(s string) (compile.Env, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return nil, fmt.Errorf("not an EDN map: %s", s)
	}
	s = s[1 : len(s)-1]
	env := compile.Env{}
	var values []any
	for {
		s = s[len(ednSeparator.FindString(s)):]
		if s == "" {
			break
		}
		m := edn.FindStringSubmatch(s)
		if m == nil {
			return nil, fmt.Errorf("can't read EDN value at %q", s)
		}
		s = s[len(m[0]):]
		switch {
		case strings.HasPrefix(strings.TrimLeftFunc(m[0], unicode.IsSpace), `"`):
			v, err := strconv.Unquote(`"` + m[1] + `"`)
			if err != nil {
				return nil, err
			}
			values = append(values, v)
		case m[2] == "nil":
			values = append(values, nil)
		case m[2] != "":
			values = append(values, m[2] == "true")
		default:
			n, err := strconv.ParseInt(m[3], 10, 64)
			if err != nil {
				return nil, err
			}
			values = append(values, n)
		}
	}
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("an EDN map has a key with no value")
	}
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("an EDN map key isn't a string: %v", values[i])
		}
		env[key] = values[i+1]
	}
	return env, nil
}

// checkTemplates renders every case of every replacer fixture in dir, and
// returns those that differ, and how many there are.
func checkTemplates(dir string) ([]row, int, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	if err != nil {
		return nil, 0, err
	}
	if len(files) == 0 {
		return nil, 0, fmt.Errorf("no replacer fixtures in %s", dir)
	}
	var res []row
	total := 0
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, 0, err
		}
		name := "replacer/" + filepath.Base(file)
		text, cases, err := readFixture(string(content))
		if err != nil {
			return nil, 0, fmt.Errorf("%s: %w", name, err)
		}
		total += len(cases)
		for i, c := range cases {
			got, err := compile.RenderString(text, c.env, compile.At{})
			switch {
			case err != nil:
				res = append(res, templateRow(name, i, err.Error()))
			case got != c.want:
				res = append(res, templateRow(name, i, fmt.Sprintf("rendered %q, want %q", got, c.want)))
			}
		}
	}
	return res, total, nil
}

func templateRow(file string, i int, message string) row {
	return row{File: file, Severity: 1, Message: fmt.Sprintf("case %d: %s", i+1, message), Check: "template"}
}

// checkWorkflowConditions evaluates the when and unless of each workflow in
// every build.yml under dir that compiles, and returns those whose outcome
// isn't the reference compiler's: a workflow is kept in config.yml if, and
// only if, its conditions allow it. The variables are the pipeline
// parameters' defaults, which is all the reference compiler gives most of
// these cases. It also returns how many workflows it checked.
func checkWorkflowConditions(dir string) ([]row, int, error) {
	configs, err := referenceConfigs(dir)
	if err != nil {
		return nil, 0, err
	}
	var res []row
	total := 0
	for _, c := range configs {
		caseDir := filepath.Dir(c.path())
		if exists(filepath.Join(caseDir, "errors.txt")) || exists(filepath.Join(caseDir, "parse_errors.txt")) {
			continue
		}
		name := "logic/" + c.name
		build, err := os.ReadFile(c.path())
		if err != nil {
			return nil, 0, err
		}
		compiled, err := os.ReadFile(filepath.Join(caseDir, "config.yml"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, 0, err
		}
		rows, n, err := checkWorkflows(name, build, compiled)
		if err != nil {
			return nil, 0, fmt.Errorf("%s: %w", name, err)
		}
		res = append(res, rows...)
		total += n
	}
	return res, total, nil
}

func checkWorkflows(name string, build, compiled []byte) ([]row, int, error) {
	var source struct {
		Parameters map[string]struct {
			Default any `yaml:"default"`
		} `yaml:"parameters"`
		Workflows map[string]any `yaml:"workflows"`
	}
	if err := yaml.Unmarshal(build, &source); err != nil {
		return nil, 0, err
	}
	var out struct {
		Workflows map[string]any `yaml:"workflows"`
	}
	if err := yaml.Unmarshal(compiled, &out); err != nil {
		return nil, 0, err
	}

	env := compile.Env{"pipeline.id": "00000000-0000-0000-0000-000000000000"}
	for param, p := range source.Parameters {
		v := p.Default
		if i, ok := v.(int); ok {
			v = int64(i)
		}
		env["pipeline.parameters."+param] = v
	}

	doc, err := parser.ParseForCompile(build, name, &session.Settings{})
	if err != nil {
		return nil, 0, err
	}
	defer doc.Close()
	info := doc.Compile()

	var res []row
	total := 0
	for workflow := range source.Workflows {
		if workflow == "version" {
			continue
		}
		total++
		keep := true
		var evalErr error
		for _, cond := range info.WorkflowConditions[workflow] {
			ok, err := compile.EvalWorkflowWhen(cond.Condition, env, compile.At{})
			if err != nil {
				evalErr = err
				break
			}
			if (cond.Kind == "when") != ok {
				keep = false
			}
		}
		_, kept := out.Workflows[workflow]
		line := 0
		if conds := info.WorkflowConditions[workflow]; len(conds) != 0 {
			line = int(conds[0].Range.Start.Line) + 1
		}
		switch {
		case evalErr != nil:
			res = append(res, row{File: name, Line: line, Severity: 1, Message: fmt.Sprintf("workflow %s: %s", workflow, evalErr), Check: "condition"})
		case keep != kept:
			res = append(res, row{File: name, Line: line, Severity: 1, Message: fmt.Sprintf("workflow %s: kept is %t, want %t", workflow, keep, kept), Check: "condition"})
		}
	}
	return res, total, nil
}

func exists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

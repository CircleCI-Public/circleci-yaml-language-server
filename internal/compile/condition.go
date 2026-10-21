package compile

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/CircleCI-Public/expr/interpreters"
	"github.com/CircleCI-Public/expr/parsers"
	"github.com/CircleCI-Public/expr/scanners"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/template"
)

// maxLogicDepth is how deeply the reference compiler lets logic statements
// nest.
const maxLogicDepth = 100

// EvalCondition evaluates a when or unless step's condition, a logic
// statement such as `and: [...]`, as written. Strings in it are rendered as
// templates first.
func EvalCondition(v ast.ParameterValue, env Env, at At) (bool, error) {
	return logicTruthy(maxLogicDepth, plain(v), env, at)
}

// EvalWorkflowWhen evaluates a workflow's when or unless. A string with no
// variables is an expression, such as `pipeline.git.branch == "main"`; any
// other value is a logic statement.
func EvalWorkflowWhen(v ast.ParameterValue, env Env, at At) (bool, error) {
	if text, ok := v.Value.(string); ok && len(template.References(text)) == 0 {
		return evalExpression(text, env, at)
	}
	return EvalCondition(v, env, at)
}

func evalExpression(text string, env Env, at At) (bool, error) {
	fail := func(message string) (bool, error) {
		return false, at.errorf("Error evaluating workflow filter expression:\n%s", message)
	}
	if _, problem := template.Expression(text); problem != nil {
		return fail(problem.Message)
	}
	toks, err := scanners.New(text).Scan()
	if err != nil {
		return fail(err.Error())
	}
	expression, err := parsers.New(toks).Parse()
	if err != nil {
		return fail(err.Error())
	}
	ok, err := interpreters.New(scalars(env)).Interpret(expression)
	var exprErr interpreters.Error
	if errors.As(err, &exprErr) {
		return fail(firstLine(exprErr.AsErrorMessage(text)))
	}
	if err != nil {
		return fail(err.Error())
	}
	return ok, nil
}

// logicMap is a mapping as written, with its keys in order, since a logic
// statement is its first key.
type logicMap struct {
	keys   []string
	values map[string]any
}

// plain is v's value with lists as []any, mappings as logicMap and integers
// as int64.
func plain(v ast.ParameterValue) any {
	switch value := v.Value.(type) {
	case []ast.ParameterValue:
		out := make([]any, len(value))
		for i, item := range value {
			out[i] = plain(item)
		}
		return out
	case map[string]ast.ParameterValue:
		m := logicMap{values: make(map[string]any, len(value))}
		for key, item := range value {
			m.keys = append(m.keys, key)
			m.values[key] = plain(item)
		}
		slices.SortFunc(m.keys, func(a, b string) int {
			return comparePosition(value[a].Range.Start.Line, value[a].Range.Start.Character,
				value[b].Range.Start.Line, value[b].Range.Start.Character)
		})
		return m
	case int:
		return int64(value)
	}
	return v.Value
}

func comparePosition(lineA, charA, lineB, charB uint32) int {
	if lineA != lineB {
		return int(lineA) - int(lineB)
	}
	return int(charA) - int(charB)
}

// logicTruthy is whether v resolves to anything but false, 0, "" or nil.
func logicTruthy(depth int, v any, env Env, at At) (bool, error) {
	v, err := resolve(depth, v, env, at)
	if err != nil {
		return false, err
	}
	switch v := v.(type) {
	case nil:
		return false, nil
	case bool:
		return v, nil
	case int64:
		return v != 0, nil
	case int:
		return v != 0, nil
	case string:
		return v != "", nil
	}
	return true, nil
}

// resolve evaluates v if it's a logic statement, renders it if it's a
// string, and otherwise gives it back as written.
func resolve(depth int, v any, env Env, at At) (any, error) {
	if depth == 0 {
		return nil, at.errorf("Error expanding. Logic is nested too deeply.")
	}
	switch v := v.(type) {
	case string:
		return Render(v, env, at)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			r, err := resolve(depth-1, item, env, at)
			if err != nil {
				return nil, err
			}
			out[i] = r
		}
		return out, nil
	case logicMap:
		if len(v.keys) == 0 {
			return v.plainMap(depth, env, at)
		}
		return resolveStatement(depth-1, v.keys[0], v.values[v.keys[0]], v, env, at)
	}
	return v, nil
}

func resolveStatement(depth int, op string, operand any, m logicMap, env Env, at At) (any, error) {
	list := func() ([]any, error) {
		if operand == nil {
			return nil, nil
		}
		clauses, ok := operand.([]any)
		if !ok {
			return nil, at.errorf("`%s` condition requires a list, got: %s", op, prStr(operand))
		}
		return clauses, nil
	}

	switch op {
	case "and":
		clauses, err := list()
		if err != nil || len(clauses) == 0 {
			return false, err
		}
		for _, c := range clauses {
			if ok, err := logicTruthy(depth, c, env, at); err != nil || !ok {
				return false, err
			}
		}
		return true, nil
	case "or":
		clauses, err := list()
		if err != nil {
			return false, err
		}
		for _, c := range clauses {
			if ok, err := logicTruthy(depth, c, env, at); err != nil || ok {
				return ok, err
			}
		}
		return false, nil
	case "not":
		ok, err := logicTruthy(depth, operand, env, at)
		return !ok, err
	case "equal":
		clauses, err := list()
		if err != nil || len(clauses) == 0 {
			return false, err
		}
		var first any
		for i, c := range clauses {
			r, err := resolve(depth, c, env, at)
			if err != nil {
				return false, err
			}
			r = normalise(r)
			if i == 0 {
				first = r
			} else if !reflect.DeepEqual(first, r) {
				return false, nil
			}
		}
		return true, nil
	case "matches":
		return matches(depth, operand, env, at)
	}
	return m.plainMap(depth+1, env, at)
}

func matches(depth int, operand any, env Env, at At) (bool, error) {
	m, _ := operand.(logicMap)
	pattern, err := resolve(depth, m.values["pattern"], env, at)
	if err != nil {
		return false, err
	}
	value, err := resolve(depth, m.values["value"], env, at)
	if err != nil {
		return false, err
	}
	p, ok := pattern.(string)
	if !ok {
		return false, at.errorf("Invalid regular expression: %s", prStr(pattern))
	}
	if len(p) >= 2 && strings.HasPrefix(p, "/") && strings.HasSuffix(p, "/") {
		p = p[1 : len(p)-1]
	}
	// Go's regexps are RE2, as the reference compiler's newer matching is.
	re, err := regexp.Compile(`^(?:` + p + `)$`)
	if err != nil {
		return false, at.errorf("Invalid regular expression: %s", p)
	}
	if value == nil {
		value = ""
	}
	s, ok := value.(string)
	if !ok {
		return false, at.errorf("Invalid regular expression value")
	}
	return re.MatchString(s), nil
}

// plainMap is m with its values rendered, as a mapping that isn't a logic
// statement is taken as it is.
func (m logicMap) plainMap(depth int, env Env, at At) (any, error) {
	out := make(map[string]any, len(m.values))
	for key, v := range m.values {
		r, err := resolve(depth-1, v, env, at)
		if err != nil {
			return nil, err
		}
		out[key] = r
	}
	return out, nil
}

// normalise makes equal values compare equal with reflect.DeepEqual.
func normalise(v any) any {
	switch v := v.(type) {
	case int:
		return int64(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = normalise(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = normalise(item)
		}
		return out
	}
	return v
}

// prStr writes v roughly as Clojure's pr-str does, for errors that quote it.
func prStr(v any) string {
	switch v := v.(type) {
	case nil:
		return "nil"
	case string:
		return strconv.Quote(v)
	case logicMap:
		parts := make([]string, len(v.keys))
		for i, key := range v.keys {
			parts[i] = ":" + key + " " + prStr(v.values[key])
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return fmt.Sprint(v)
}

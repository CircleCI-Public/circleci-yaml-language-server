package compile

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/CircleCI-Public/expr/interpreters"
	"github.com/CircleCI-Public/expr/parsers"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/suggest"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/template"
)

// Render renders text as a template. A text that's a single tag using a
// variable gives the tag's value, typed, and nil if a variable is unknown, as
// the reference compiler keeps configs working that rely on that. Any other
// text gives a string.
func Render(text string, env Env, at At) (any, error) {
	if !strings.Contains(text, "<<") {
		return text, nil
	}
	t, problem := template.Parse(text)
	if problem != nil {
		return nil, at.errorf("%s", problem.Message)
	}
	if tag, ok := onlyTag(t, text); ok {
		v, err := evaluate(tag, text, env, at)
		if errors.Is(err, errUnknown) {
			return nil, nil
		}
		return v, err
	}
	return RenderString(text, env, at)
}

// onlyTag is t's tag, if it's all of t and uses a variable.
func onlyTag(t template.Template, text string) (template.Tag, bool) {
	if len(t) != 1 {
		return template.Tag{}, false
	}
	tag, ok := t[0].(template.Tag)
	return tag, ok && len(template.References(text)) != 0
}

// RenderString renders text as a template, to a string. An unknown variable
// renders as "".
func RenderString(text string, env Env, at At) (string, error) {
	if !strings.Contains(text, "<<") {
		return text, nil
	}
	text = trimStandalone(text)
	t, problem := template.Parse(text)
	if problem != nil {
		return "", at.errorf("%s", problem.Message)
	}
	var b strings.Builder
	if err := render(&b, t, text, env, at); err != nil {
		return "", err
	}
	return b.String(), nil
}

func render(b *strings.Builder, t template.Template, text string, env Env, at At) error {
	for _, block := range t {
		switch block := block.(type) {
		case template.Text:
			b.WriteString(block.Value)
		case template.Tag:
			v, err := evaluate(block, text, env, at)
			if errors.Is(err, errUnknown) {
				continue
			}
			if err != nil {
				return err
			}
			s, err := textOf(v)
			if err != nil {
				return at.errorf("%s", err)
			}
			b.WriteString(s)
		case template.Section:
			if sectionTruthy(env[block.Name]) != block.Inverted {
				if err := render(b, block.Body, text, env, at); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// standaloneSection is a line holding only a section's start or end tag.
var standaloneSection = regexp.MustCompile(`^\s*<<[/#^]\s*[\w-]+(?:\.[\w-]+)+\s*>>$`)

// trimStandalone drops the indentation and newline of each line that holds
// only a section tag, so that the section adds no blank line. Every newline
// comes out as \n.
func trimStandalone(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var b strings.Builder
	last := len(lines) - 1
	for i, line := range lines {
		if standaloneSection.MatchString(line) {
			b.WriteString(strings.TrimSpace(line))
			continue
		}
		b.WriteString(line)
		if i != last {
			b.WriteString("\n")
		}
	}
	return b.String()
}

var errUnknown = errors.New("unknown variable")

// evaluate is the value of tag, in text. A bare name gives its value as it
// is, whatever its type, and errUnknown if env doesn't have it.
func evaluate(tag template.Tag, text string, env Env, at At) (any, error) {
	if ident, ok := tag.Expr.(parsers.Identifier); tag.Expr == nil || ok {
		name := tag.Name
		if ok {
			name = ident.Name.Lexeme
		}
		v, found := env[name]
		if !found {
			return nil, errUnknown
		}
		return v, nil
	}

	v, err := interpreters.New(scalars(env)).Evaluate(tag.Expr)
	var exprErr interpreters.Error
	if errors.As(err, &exprErr) && exprErr.Type == interpreters.UNKNOWN_VARIABLE {
		name := exprErr.Token.Lexeme
		if v, found := env[name]; found {
			return nil, at.errorf("%s is a %s, which can't be used in an expression", name, kind(v))
		}
		return nil, errUnknown
	}
	if errors.As(err, &exprErr) {
		return nil, at.errorf("%s", firstLine(exprErr.AsErrorMessage(text[tag.Start+2:tag.End-2])))
	}
	if err != nil {
		return nil, at.errorf("%s", err)
	}
	return v.Unbox(), nil
}

// scalars is env's variables that an expression can use.
func scalars(env Env) map[string]interpreters.Val {
	out := make(map[string]interpreters.Val, len(env))
	for name, v := range env {
		if boxed, err := interpreters.BoxVal(v); err == nil {
			out[name] = boxed
		}
	}
	return out
}

func firstLine(message string) string {
	line, _, _ := strings.Cut(message, "\n")
	return strings.TrimSuffix(line, ":")
}

// textOf is how v reads in text. The reference compiler writes a list or map
// in Clojure's syntax, which is never what anyone meant, so it's an error
// here.
func textOf(v any) (string, error) {
	switch v := v.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case bool:
		return strconv.FormatBool(v), nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		return formatDouble(v), nil
	}
	return "", fmt.Errorf("a %s can't be written into text", kind(v))
}

// formatDouble writes f as Java's Double.toString does.
func formatDouble(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	if a := math.Abs(f); a == 0 || (a >= 1e-3 && a < 1e7) {
		s := strconv.FormatFloat(f, 'f', -1, 64)
		if !strings.Contains(s, ".") {
			s += ".0"
		}
		return s
	}
	mantissa, exponent, _ := strings.Cut(strconv.FormatFloat(f, 'E', -1, 64), "E")
	if !strings.Contains(mantissa, ".") {
		mantissa += ".0"
	}
	n, _ := strconv.Atoi(exponent)
	return mantissa + "E" + strconv.Itoa(n)
}

func kind(v any) string {
	switch reflect.ValueOf(v).Kind() {
	case reflect.Slice:
		return "list"
	case reflect.Map:
		return "map"
	}
	return fmt.Sprintf("%T", v)
}

// sectionTruthy is whether a section shows its body: all but nil, false, ""
// and an empty list do, so 0, "   " and an empty map do.
func sectionTruthy(v any) bool {
	switch v := v.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	}
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Slice {
		return rv.Len() != 0
	}
	return true
}

// Unknown is an error naming the variables text uses that env doesn't have,
// with the names they're probably typos of, or nil if there are none.
func Unknown(text string, env Env, at At) error {
	names := make([]string, 0, len(env))
	for name := range env {
		names = append(names, name)
	}
	var unknown []string
	seen := map[string]bool{}
	for _, ref := range template.References(text) {
		if _, found := env[ref.Name]; found || seen[ref.Name] {
			continue
		}
		seen[ref.Name] = true
		if near := suggest.Closest(ref.Name, names); near != "" {
			unknown = append(unknown, fmt.Sprintf("%s (did you mean %s?)", ref.Name, near))
		} else {
			unknown = append(unknown, ref.Name)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	return at.errorf("Unknown variable(s): %s", strings.Join(unknown, ", "))
}

// Unescape turns each escaped `\<<` or `\>>` into `<<` or `>>`, leaving one
// whose backslash is itself escaped. It's the last thing compiling does to
// every string.
func Unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && (i == 0 || s[i-1] != '\\') &&
			(strings.HasPrefix(s[i+1:], "<<") || strings.HasPrefix(s[i+1:], ">>")) {
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// RenderPartial renders what it can of text, leaving each tag or section
// that uses a variable env doesn't have as written, so text can be rendered
// before all its variables are known. Text that isn't a valid template is
// left as written.
func RenderPartial(text string, env Env) string {
	if !strings.Contains(text, "<<") {
		return text
	}
	t, problem := template.Parse(text)
	if problem != nil {
		return text
	}
	refs := template.References(text)
	var b strings.Builder
	renderPartial(&b, t, text, refs, env)
	return b.String()
}

func renderPartial(b *strings.Builder, t template.Template, text string, refs []template.Reference, env Env) {
	known := func(start, end int) bool {
		for _, ref := range refs {
			if _, found := env[ref.Name]; ref.Start >= start && ref.End <= end && !found {
				return false
			}
		}
		return true
	}
	for _, block := range t {
		switch block := block.(type) {
		case template.Text:
			b.WriteString(block.Value)
		case template.Tag:
			if known(block.Start, block.End) {
				if v, err := evaluate(block, text, env, At{}); err == nil {
					if s, err := textOf(v); err == nil {
						b.WriteString(s)
						continue
					}
				}
			}
			b.WriteString(text[block.Start:block.End])
		case template.Section:
			if _, found := env[block.Name]; !found {
				b.WriteString(text[block.Start:block.End])
				continue
			}
			if sectionTruthy(env[block.Name]) != block.Inverted {
				renderPartial(b, block.Body, text, refs, env)
			}
		}
	}
}

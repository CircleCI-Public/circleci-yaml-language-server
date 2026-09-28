package validate

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/template"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamlbool"
)

var templateRegex = regexp.MustCompile(`<<.*?>>`)

// ValidateConditions checks the conditions and filters written as strings.
// Without a << >> template, a workflow's `when` or `unless`, or a job's
// `filters`, is an expression, and one that doesn't parse is an error. A
// step condition that isn't one is always true.
func (val Validate) ValidateConditions() {
	for _, condition := range val.Doc.Conditions {
		if slices.Contains(val.Doc.StepConditions, condition) {
			val.warnAlwaysTrueStepCondition(condition)
		} else {
			val.checkExpression(condition, "Invalid condition expression")
		}
		val.warnTemplatedComparison(condition)
	}
	for _, filter := range val.Doc.FilterExpressions {
		val.checkExpression(filter, "Invalid filter expression")
	}
}

// warnTemplatedComparison warns about a condition that compares a << >>
// template outside it, such as `<< parameters.env >> == "prod"`. The template
// makes it a string, which is always true, so the comparison is never made.
func (val Validate) warnTemplatedComparison(condition ast.TextAndRange) {
	outside := templateRegex.ReplaceAllString(condition.Text, "")
	looksLikeComparison := strings.Contains(outside, "==") || strings.Contains(outside, "!=") ||
		strings.ContainsAny(outside, "<>")
	if outside == condition.Text || !looksLikeComparison {
		return
	}
	val.addDiagnostic(diagnostic.Warning(condition.Range,
		"A << >> template makes this condition a logic statement that is always true, "+
			"so the comparison is not evaluated. Use a logic statement such as `equal:` "+
			"to compare a parameter value."))
}

func (val Validate) checkExpression(value ast.TextAndRange, message string) {
	if !val.isExpression(value) {
		return
	}
	if _, problem := template.Expression(value.Text); problem != nil {
		val.addDiagnostic(diagnostic.Error(val.rangeWithin(value.Range, problem.Start, problem.End),
			fmt.Sprintf("%s: %s", message, problem.Message)))
	}
}

// warnAlwaysTrueStepCondition warns about a step condition that is a string
// but not an expression, which is always true unless it's empty. So is one
// that uses a name that's not a value, such as `always`, which the compiler
// takes as a string.
func (val Validate) warnAlwaysTrueStepCondition(condition ast.TextAndRange) {
	if !val.isExpression(condition) {
		return
	}
	names, problem := template.Expression(condition.Text)
	if problem == nil && !slices.ContainsFunc(names, func(name template.Reference) bool {
		return !strings.Contains(name.Name, ".")
	}) {
		return
	}
	val.addDiagnostic(diagnostic.Warning(condition.Range, fmt.Sprintf(
		"Condition `%s` is treated as always true. Use a boolean (`true`/`false`), an expression, "+
			"or a logic statement such as `equal:`.", strings.TrimSpace(condition.Text))))
}

// isExpression reports whether a value is read as an expression: a string
// without a << >> template, which the compiler renders first. A YAML boolean
// or number isn't a string.
func (val Validate) isExpression(value ast.TextAndRange) bool {
	text := strings.TrimSpace(value.Text)
	if text == "" || strings.Contains(text, "<<") {
		return false
	}
	if val.isQuoted(value.Range) {
		return true
	}
	lower := strings.ToLower(text)
	if yamlbool.IsValid(lower) || lower == "null" || lower == "~" {
		return false
	}
	_, err := strconv.ParseFloat(text, 64)
	return err != nil
}

func (val Validate) isQuoted(r protocol.Range) bool {
	start := position.ToIndex(r.Start, val.Doc.Content)
	return start < len(val.Doc.Content) && (val.Doc.Content[start] == '"' || val.Doc.Content[start] == '\'')
}

// rangeWithin is the range of a part of a scalar, given as byte offsets into
// its unquoted text. A scalar over several lines is folded, so the offsets
// don't match the lines, and the whole of it is given.
func (val Validate) rangeWithin(r protocol.Range, start, end int) protocol.Range {
	if r.Start.Line != r.End.Line {
		return r
	}
	from := position.ToIndex(r.Start, val.Doc.Content)
	if val.isQuoted(r) {
		from++
	}
	return protocol.Range{
		Start: position.FromIndex(from+start, val.Doc.Content),
		End:   position.FromIndex(from+end, val.Doc.Content),
	}
}

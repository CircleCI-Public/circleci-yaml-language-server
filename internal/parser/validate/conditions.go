package validate

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

var templateRegex = regexp.MustCompile(`<<.*?>>`)

// ValidateConditions warns about a condition that compares a << >> template
// outside it, such as `<< parameters.env >> == "prod"`. The template makes it
// a string, which is always true, so the comparison is never made.
func (val Validate) ValidateConditions() {
	val.warnAlwaysTrueStepConditions()

	for _, condition := range val.Doc.Conditions {
		outside := templateRegex.ReplaceAllString(condition.Text, "")
		looksLikeComparison := strings.Contains(outside, "==") || strings.Contains(outside, "!=") ||
			strings.ContainsAny(outside, "<>")
		if outside == condition.Text || !looksLikeComparison {
			continue
		}
		val.addDiagnostic(diagnostic.Warning(condition.Range,
			"A << >> template makes this condition a logic statement that is always true, "+
				"so the comparison is not evaluated. Use a logic statement such as `equal:` "+
				"to compare a parameter value."))
	}
}

// runWhenValues are the values of a run step's `when`, which are sometimes
// written as a step condition instead.
var runWhenValues = []string{"always", "on_success", "on_fail"}

var quotedRegex = regexp.MustCompile(`"[^"]*"|'[^']*'`)

// warnAlwaysTrueStepConditions warns about a step condition that is a string
// but not an expression, which is always true unless it's empty. Only the
// strings that can't be an expression are caught: an environment variable,
// which the condition can't read, or a value of a run step's `when`.
func (val Validate) warnAlwaysTrueStepConditions() {
	for _, condition := range val.Doc.StepConditions {
		text := strings.TrimSpace(condition.Text)
		if text == "" || templateRegex.MatchString(text) {
			continue
		}
		if !strings.Contains(quotedRegex.ReplaceAllString(text, ""), "$") && !slices.Contains(runWhenValues, text) {
			continue
		}
		val.addDiagnostic(diagnostic.Warning(condition.Range, fmt.Sprintf(
			"Condition `%s` is treated as always true. Use a boolean (`true`/`false`), an expression, "+
				"or a logic statement such as `equal:`.", text)))
	}
}

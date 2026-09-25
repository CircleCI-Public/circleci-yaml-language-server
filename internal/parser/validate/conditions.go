package validate

import (
	"regexp"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

var templateRegex = regexp.MustCompile(`<<.*?>>`)

// ValidateConditions warns about a condition that compares a << >> template
// outside it, such as `<< parameters.env >> == "prod"`. The template makes it
// a string, which is always true, so the comparison is never made.
func (val Validate) ValidateConditions() {
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

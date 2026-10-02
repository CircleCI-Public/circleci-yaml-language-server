package complete

import (
	"context"
	"regexp"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
)

// mappingKey is a key, or a list item's key, with its body on the lines
// under it.
var mappingKey = regexp.MustCompile(`^\s*(?:-\s+)?([\w/.-]+)\s*:\s*$`)

// invocationMappingKeys are the keys of each mapping in a job invocation,
// by the path to the mapping from the invocation.
var invocationMappingKeys = map[string][]string{
	"filters":          {"branches", "tags"},
	"filters.branches": {"only", "ignore"},
	"filters.tags":     {"only", "ignore"},
	"matrix":           {"parameters", "exclude", "alias"},
}

// completeInvocationMapping offers the keys a mapping in a job invocation
// doesn't have yet, such as its filters or its matrix, when the cursor is
// at a key of one, and says whether it is.
func (ch *CompletionHandler) completeInvocationMapping(ctx context.Context, invocations []ast.JobInvocation) bool {
	lines, parent := ch.keyParent()
	var path []string
	for line := parent; line != -1; line = parentLine(lines, line) {
		if invocation := invocationNamedOn(line, invocations); invocation != nil && stepWithBody.MatchString(lines[line]) {
			if len(path) == 0 {
				return false
			}
			ch.offerInvocationMappingKeys(ctx, invocation, strings.Join(path, "."), parent)
			return true
		}

		match := mappingKey.FindStringSubmatch(lines[line])
		if match == nil {
			return false
		}
		path = append([]string{match[1]}, path...)
	}
	return false
}

func (ch *CompletionHandler) offerInvocationMappingKeys(ctx context.Context, invocation *ast.JobInvocation, path string, mappingLine int) {
	keys := invocationMappingKeys[path]
	if path == "matrix.parameters" {
		keys = ch.jobParameterNames(ctx, invocation)
	}

	present := ch.stepBodyKeys(mappingLine)
	for _, key := range keys {
		if !present[key] {
			ch.addCompletionItemField(key)
		}
	}
}
